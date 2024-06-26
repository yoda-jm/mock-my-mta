package main

import (
	"bytes"
	"net/mail"
	"net/smtp"

	"github.com/chrj/smtpd"

	"mock-my-mta/log"
	"mock-my-mta/storage"
)

type smtpServer struct {
	server       *smtpd.Server
	addr         string
	relayAddress string

	storageEngine *storage.Engine
}

func newSmtpServer(addr string, storageEngine *storage.Engine, relayAddress string) *smtpServer {
	s := &smtpServer{
		addr:          addr,
		relayAddress:  relayAddress,
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
	if s.relayAddress != "" {
		err = smtp.SendMail(s.relayAddress, nil, env.Sender, env.Recipients, env.Data)
		if err != nil {
			log.Logf(log.ERROR, "failed to relay message: %v", err)
		}
	}
	return nil
}
