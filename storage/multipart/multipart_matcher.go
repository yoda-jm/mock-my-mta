package multipart

import (
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
		searchText := strings.ToLower(mt.GetText())
		// check if the body contains the string (case insensitive)
		bodyVersions := multipart.GetBodyVersions()
		for _, bodyVersion := range bodyVersions {
			body, err := multipart.GetBody(bodyVersion)
			if err != nil {
				continue
			}
			if strings.Contains(strings.ToLower(body), searchText) {
				return true
			}
		}
		// check if the subject contains the string
		if strings.Contains(strings.ToLower(multipart.GetSubject()), searchText) {
			return true
		}
		// check if the from contains the string
		if strings.Contains(strings.ToLower(multipart.GetFrom().Address), searchText) {
			return true
		}
		// check if the recipients contain the string
		for _, recipient := range multipart.GetRecipients() {
			if strings.Contains(strings.ToLower(recipient.Address), searchText) {
				return true
			}
		}
		return false
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
		// Unknown matcher type — skip rather than crash
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
