package multipart

import (
	"strings"
	"time"
)

type MultipartMatcher interface {
	Match(email *Multipart) bool
}

type MailboxMatch struct {
	mailbox string
}

func NewMailboxMatch(mailbox string) MailboxMatch {
	return MailboxMatch{mailbox: mailbox}
}

func (m MailboxMatch) Match(email *Multipart) bool {
	recipents := email.GetRecipients()
	for _, recipent := range recipents {
		// case insensitive match
		if strings.EqualFold(recipent.Address, m.mailbox) {
			return true
		}
	}
	return false
}

type AttachmentMatch struct {
}

func NewAttachmentMatch() AttachmentMatch {
	return AttachmentMatch{}
}

func (a AttachmentMatch) Match(email *Multipart) bool {
	return email.HasAttachments()
}

type PlainTextMatch struct {
	plainText string
}

func NewPlainTextMatch(plainText string) PlainTextMatch {
	return PlainTextMatch{plainText: plainText}
}

func (p PlainTextMatch) Match(email *Multipart) bool {
	// FIXME: implement, for now always return true for testing (to be able to test the other parts)
	return true
}

type BeforeMatch struct {
	date time.Time
}

func NewBeforeMatch(date time.Time) BeforeMatch {
	return BeforeMatch{date: date}
}

func (b BeforeMatch) Match(email *Multipart) bool {
	if email.GetDate().Before(b.date) {
		return true
	}
	return false
}

type AfterMatch struct {
	date time.Time
}

func NewAfterMatch(date time.Time) AfterMatch {
	return AfterMatch{date: date}
}

func (a AfterMatch) Match(email *Multipart) bool {
	if email.GetDate().After(a.date) {
		return true
	}
	return false
}

type FromMatch struct {
	from string
}

func NewFromMatch(from string) FromMatch {
	return FromMatch{from: from}
}

func (f FromMatch) Match(email *Multipart) bool {
	// case insensitive match
	if strings.EqualFold(email.GetFrom().Address, f.from) {
		return true
	}
	return false
}

type NewerThanMatch struct {
	duration time.Duration
}

func NewNewerThanMatch(duration time.Duration) NewerThanMatch {
	return NewerThanMatch{duration: duration}
}

func (n NewerThanMatch) Match(email *Multipart) bool {
	if time.Since(email.GetDate()) < n.duration {
		return true
	}
	return false
}

type OlderThanMatch struct {
	duration time.Duration
}

func NewOlderThanMatch(duration time.Duration) OlderThanMatch {
	return OlderThanMatch{duration: duration}
}

func (o OlderThanMatch) Match(email *Multipart) bool {
	if time.Since(email.GetDate()) > o.duration {
		return true
	}
	return false
}

type SubjectMatch struct {
	subject string
}

func NewSubjectMatch(subject string) SubjectMatch {
	return SubjectMatch{subject: subject}
}

func (s SubjectMatch) Match(email *Multipart) bool {
	// check if the subject contains the string (case insensitive)
	if strings.Contains(strings.ToLower(email.GetSubject()), strings.ToLower(s.subject)) {
		return true
	}
	return false
}
