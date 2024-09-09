package main

import (
	"bytes"
	"fmt"
	"net/mail"
	"net/smtp"

	"github.com/chrj/smtpd"

	"mock-my-mta/log"
	"mock-my-mta/storage"
)

type smtpServer struct {
	server             *smtpd.Server
	addr               string
	relayConfiguration RelayConfiguration

	storageEngine *storage.Engine
}

func newSmtpServer(addr string, storageEngine *storage.Engine, relayConfiguration RelayConfiguration) *smtpServer {
	s := &smtpServer{
		addr:               addr,
		relayConfiguration: relayConfiguration,
		storageEngine:      storageEngine,
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

func (s *smtpServer) ListenAndServe() error {
	log.Logf(log.INFO, "starting smtp server on %v", s.addr)
	return s.server.ListenAndServe(s.addr)
}

func (s *smtpServer) Shutdown() error {
	log.Logf(log.INFO, "stopping smtp server...", s.addr)
	return s.server.Shutdown(true)
}

func (s *smtpServer) recipientChecker(peer smtpd.Peer, addr string) error {
	log.Logf(log.DEBUG, "received recipent %v", addr)
	return nil
}

func (s *smtpServer) senderChecker(peer smtpd.Peer, addr string) error {
	log.Logf(log.DEBUG, "received sender %v", addr)
	return nil
}

func (s *smtpServer) heloChecker(peer smtpd.Peer, name string) error {
	log.Logf(log.DEBUG, "received HELO from %v", name)
	return nil
}

func (s *smtpServer) connectionChecker(peer smtpd.Peer) error {
	log.Logf(log.DEBUG, "new connection from %v", peer.Addr)
	return nil
}

func (s *smtpServer) handler(peer smtpd.Peer, env smtpd.Envelope) error {
	log.Logf(log.DEBUG, "peer=%+v", peer)
	log.Logf(log.DEBUG, "envelope=%+v", env)

	// create new byte reader from env.Data
	br := bytes.NewReader(env.Data)
	message, err := mail.ReadMessage(br)
	if err != nil {
		return err
	}
	err = s.storageEngine.Set(message)
	if err != nil {
		return err
	}
	if s.relayConfiguration.Enabled {
		var auth smtp.Auth
		switch s.relayConfiguration.AuthMode {
		case RelayAuthModeNone:
			auth = nil
		case RelayAuthModePlain:
			auth = smtp.PlainAuth("", s.relayConfiguration.Username, s.relayConfiguration.Password, s.relayConfiguration.Addr)
		case RelayAuthModeLogin:
			auth = newLoginAuth(s.relayConfiguration.Username, s.relayConfiguration.Password)
		case RelayAuthModeCramMD5:
			auth = smtp.CRAMMD5Auth(s.relayConfiguration.Username, s.relayConfiguration.Password)
		default:
			return fmt.Errorf("unsupported relay auth mode: %v", s.relayConfiguration.AuthMode)
		}
		err = smtp.SendMail(s.relayConfiguration.Addr, auth, env.Sender, env.Recipients, env.Data)
		if err != nil {
			log.Logf(log.ERROR, "failed to relay message: %v", err)
		}
	}
	return nil
}

// LoginAuth implements the smtp.Auth interface for the LOGIN authentication mechanism
var _ smtp.Auth = &LoginAuth{}

type LoginAuth struct {
	username, password string
}

// Next implements smtp.Auth.
func (l *LoginAuth) Next(fromServer []byte, more bool) (toServer []byte, err error) {
	if more {
		return nil, fmt.Errorf("unexpected continuation from server")
	}
	if bytes.Contains(fromServer, []byte("Username:")) {
		return []byte(l.username), nil
	}
	if bytes.Contains(fromServer, []byte("Password:")) {
		return []byte(l.password), nil
	}
	return nil, fmt.Errorf("unexpected server challenge: %q", fromServer)
}

// Start implements smtp.Auth.
func (l *LoginAuth) Start(server *smtp.ServerInfo) (proto string, toServer []byte, err error) {
	return "LOGIN", []byte{}, nil
}

func newLoginAuth(username, password string) smtp.Auth {
	return &LoginAuth{username, password}
}
