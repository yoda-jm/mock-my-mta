package multipart

import (
	"log"
	"mock-my-mta/storage/matcher"
	"strings"
	"time"
)

func (multipart Multipart) match(m interface{}) bool {
	switch mt := m.(type) {
	case matcher.MailboxMatch:
		for _, recipent := range multipart.GetRecipients() {
			// case insensitive match
			if strings.EqualFold(recipent.Address, mt.GetMailbox()) {
				return true
			}
		}
		return false
	case matcher.AttachmentMatch:
		return multipart.HasAttachments()
	case matcher.PlainTextMatch:
		return true // FIXME: implement
	case matcher.BeforeMatch:
		if multipart.GetDate().Before(mt.GetDate()) {
			return true
		}
		return false
	case matcher.AfterMatch:
		if multipart.GetDate().After(mt.GetDate()) {
			return true
		}
		return false
	case matcher.FromMatch:
		// case insensitive match
		if strings.EqualFold(multipart.GetFrom().Address, mt.GetFrom()) {
			return true
		}
		return false
	case matcher.NewerThanMatch:
		if time.Since(multipart.GetDate()) < mt.GetDuration() {
			return true
		}
		return false
	case matcher.OlderThanMatch:
		if time.Since(multipart.GetDate()) > mt.GetDuration() {
			return true
		}
		return false
	case matcher.SubjectMatch:
		// check if the subject contains the string (case insensitive)
		if strings.Contains(strings.ToLower(multipart.GetSubject()), strings.ToLower(mt.GetSubject())) {
			return true
		}
		return false
	default:
		log.Fatalf("Unknown matcher type: %T", m)
		return false
	}
}

func (multipart Multipart) MatchAll(matchers []interface{}) bool {
	for _, m := range matchers {
		if !multipart.match(m) {
			return false
		}
	}
	return true
}
