package smtp

import (
	"bytes"
	"crypto/ecdsa"
	mathrand "math/rand"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"net/mail"
	"net/smtp"
	"time"

	"github.com/chrj/smtpd"

	"mock-my-mta/log"
	"mock-my-mta/storage"
)

// SmtpBehavior defines runtime-configurable SMTP behavior for chaos testing.
type SmtpBehavior struct {
	RejectRate    int    // percentage (0-100) of emails to reject
	RejectMessage string // 5xx error message
	DelayMs       int    // delay in milliseconds before accepting
	BounceRate    int    // percentage (0-100) of emails to bounce after accepting
	BounceMessage string // DSN bounce message
}

type Server struct {
	server        *smtpd.Server
	configuration Configuration

	storageEngine storage.StorageService
	onNewEmail    func(emailID string)   // callback for WebSocket notifications
	getBehavior   func() SmtpBehavior    // callback to get current SMTP behavior settings
}

// SetOnNewEmail registers a callback invoked when a new email is stored.
func (s *Server) SetOnNewEmail(fn func(emailID string)) {
	s.onNewEmail = fn
}

// SetGetBehavior registers a callback to read current SMTP behavior settings.
func (s *Server) SetGetBehavior(fn func() SmtpBehavior) {
	s.getBehavior = fn
}

func NewServer(config Configuration, storageEngine storage.StorageService) *Server {
	s := &Server{
		configuration: config,
		storageEngine: storageEngine,
	}
	tlsConfig := generateSelfSignedTLS()

	s.server = &smtpd.Server{
		WelcomeMessage:    "MockMyMTA ESMTP ready",
		Hostname:          "localhost",
		Handler:           s.handler,
		ConnectionChecker: s.connectionChecker,
		HeloChecker:       s.heloChecker,
		SenderChecker:     s.senderChecker,
		RecipientChecker:  s.recipientChecker,
		TLSConfig:         tlsConfig,
		ForceTLS:          false, // STARTTLS available but not required
		MaxMessageSize:    config.MaxMessageSize,
	}
	// Only require AUTH when explicitly configured. The chrj/smtpd library
	// returns 530 when Authenticator is set, so leaving it nil lets clients
	// send without credentials (the common case for a mock server).
	if config.RequireAuth {
		s.server.Authenticator = s.authenticator
	}
	if config.MaxMessageSize > 0 {
		log.Logf(log.INFO, "SMTP max message size: %d bytes", config.MaxMessageSize)
	}
	return s
}

func (s *Server) ListenAndServe() error {
	log.Logf(log.INFO, "starting smtp server on %v", s.configuration.Addr)
	return s.server.ListenAndServe(s.configuration.Addr)
}

func (s *Server) Shutdown() error {
	log.Logf(log.INFO, "stopping smtp server...", s.configuration.Addr)
	return s.server.Shutdown(true)
}

func (s *Server) recipientChecker(peer smtpd.Peer, addr string) error {
	log.Logf(log.DEBUG, "received recipent %v", addr)
	return nil
}

func (s *Server) senderChecker(peer smtpd.Peer, addr string) error {
	log.Logf(log.DEBUG, "received sender %v", addr)
	return nil
}

func (s *Server) heloChecker(peer smtpd.Peer, name string) error {
	log.Logf(log.DEBUG, "received HELO from %v", name)
	return nil
}

// authenticator accepts any username/password combination.
// This is a mock server — authentication always succeeds.
func (s *Server) authenticator(peer smtpd.Peer, username string, password string) error {
	log.Logf(log.DEBUG, "AUTH from %v: user=%v (accepted)", peer.Addr, username)
	return nil
}

func (s *Server) connectionChecker(peer smtpd.Peer) error {
	log.Logf(log.DEBUG, "new connection from %v", peer.Addr)
	return nil
}

func (s *Server) handler(peer smtpd.Peer, env smtpd.Envelope) error {
	log.Logf(log.DEBUG, "peer=%+v", peer)
	log.Logf(log.DEBUG, "envelope=%+v", env)

	// Apply behavior settings (chaos testing)
	if s.getBehavior != nil {
		behavior := s.getBehavior()

		// Delay
		if behavior.DelayMs > 0 {
			log.Logf(log.DEBUG, "injecting %dms delay", behavior.DelayMs)
			time.Sleep(time.Duration(behavior.DelayMs) * time.Millisecond)
		}

		// Rejection
		if behavior.RejectRate > 0 {
			if mathrand.Intn(100) < behavior.RejectRate {
				log.Logf(log.INFO, "rejecting email (chaos: %d%% reject rate)", behavior.RejectRate)
				return &smtpd.Error{Code: 550, Message: behavior.RejectMessage}
			}
		}
	}

	// create new byte reader from env.Data
	br := bytes.NewReader(env.Data)
	message, err := mail.ReadMessage(br)
	if err != nil {
		return err
	}
	uuid, err := s.storageEngine.Set(message)
	if err != nil {
		return err
	}
	// Notify connected WebSocket clients
	if s.onNewEmail != nil {
		s.onNewEmail(uuid)
	}
	for _, relayConfiguration := range s.configuration.Relays {
		switch {
		case !relayConfiguration.Enabled:
			// skip disabled relays
			continue
		case !relayConfiguration.AutoRelay:
			// skip non-auto relays
			continue
		}
		log.Logf(log.INFO, "relaying message to %v", relayConfiguration.Addr)
		err = RelayMessage(relayConfiguration, uuid, newEnvelope(env))
		if err != nil {
			log.Logf(log.ERROR, "failed to relay message: %v", err)
		}
	}
	return nil
}

type Envelope struct {
	Sender     string
	Recipients []string
	Data       []byte
}

func newEnvelope(env smtpd.Envelope) Envelope {
	return Envelope{
		Sender:     env.Sender,
		Recipients: env.Recipients,
		Data:       env.Data,
	}
}

func RelayMessage(relayConfiguration RelayConfiguration, uuid string, envelope Envelope) error {
	var auth smtp.Auth
	switch relayConfiguration.Mechanism {
	case RelayAuthModeNone:
		auth = nil
	case RelayAuthModePlain:
		host, _, err := net.SplitHostPort(relayConfiguration.Addr)
		if err != nil {
			return err
		}
		auth = smtp.PlainAuth("", relayConfiguration.Username, relayConfiguration.Password, host)
	case RelayAuthModeLogin:
		auth = newLoginAuth(relayConfiguration.Username, relayConfiguration.Password)
	case RelayAuthModeCramMD5:
		auth = smtp.CRAMMD5Auth(relayConfiguration.Username, relayConfiguration.Password)
	default:
		return fmt.Errorf("unsupported relay auth mode: %v", relayConfiguration.Mechanism)
	}
	log.Logf(log.INFO, "relaying message %v (addr=%v auth=%v, sender=%v, recipients=%v)", uuid, relayConfiguration.Addr, relayConfiguration.Mechanism, envelope.Sender, envelope.Recipients)
	return smtpSendMailFn(relayConfiguration.Addr, auth, envelope.Sender, envelope.Recipients, envelope.Data)
}

var smtpSendMailFn = smtp.SendMail

// LoginAuth implements the smtp.Auth interface for the LOGIN authentication mechanism
var _ smtp.Auth = &loginAuth{}

type loginAuth struct {
	username, password string
}

// Next implements smtp.Auth.
func (l *loginAuth) Next(fromServer []byte, more bool) (toServer []byte, err error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(l.username), nil
		case "Password:":
			return []byte(l.password), nil
		default:
			return nil, fmt.Errorf("unexpected server challenge: %q", fromServer)
		}
	}
	return nil, nil
}

// Start implements smtp.Auth.
func (l *loginAuth) Start(server *smtp.ServerInfo) (proto string, toServer []byte, err error) {
	return "LOGIN", []byte{}, nil
}

func newLoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

// generateSelfSignedTLS creates a self-signed TLS certificate for STARTTLS.
// The certificate is generated in memory — no files written to disk.
func generateSelfSignedTLS() *tls.Config {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Logf(log.ERROR, "failed to generate TLS key: %v", err)
		return nil
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "MockMyMTA"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour), // 10 years
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		log.Logf(log.ERROR, "failed to generate TLS certificate: %v", err)
		return nil
	}

	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}

	log.Logf(log.INFO, "generated self-signed TLS certificate for STARTTLS")
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
}
