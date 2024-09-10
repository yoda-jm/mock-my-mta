package smtp

import (
	"bytes"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"

	"github.com/chrj/smtpd"

	"mock-my-mta/log"
	"mock-my-mta/storage"
)

type Server struct {
	server        *smtpd.Server
	configuration Configuration

	storageEngine *storage.Engine
}

func NewServer(config Configuration, storageEngine *storage.Engine) *Server {
	s := &Server{
		configuration: config,
		storageEngine: storageEngine,
	}
	s.server = &smtpd.Server{
		WelcomeMessage:    "FAKE SMTPD GO",
		Handler:           s.handler,
		ConnectionChecker: s.connectionChecker,
		HeloChecker:       s.heloChecker,
		SenderChecker:     s.senderChecker,
		RecipientChecker:  s.recipientChecker,
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

func (s *Server) connectionChecker(peer smtpd.Peer) error {
	log.Logf(log.DEBUG, "new connection from %v", peer.Addr)
	return nil
}

func (s *Server) handler(peer smtpd.Peer, env smtpd.Envelope) error {
	log.Logf(log.DEBUG, "peer=%+v", peer)
	log.Logf(log.DEBUG, "envelope=%+v", env)

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
	return smtp.SendMail(relayConfiguration.Addr, auth, envelope.Sender, envelope.Recipients, envelope.Data)
}

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
