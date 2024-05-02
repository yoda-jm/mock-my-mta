package matcher

import (
	"time"
)

type MailboxMatch struct {
	mailbox string
}

func newMailboxMatch(mailbox string) MailboxMatch {
	return MailboxMatch{mailbox: mailbox}
}

func (m MailboxMatch) GetMailbox() string {
	return m.mailbox
}

type AttachmentMatch struct {
}

func newAttachmentMatch() AttachmentMatch {
	return AttachmentMatch{}
}

type PlainTextMatch struct {
	text string
}

func newPlainTextMatch(text string) PlainTextMatch {
	return PlainTextMatch{text: text}
}

func (p PlainTextMatch) GetText() string {
	return p.text
}

type BeforeMatch struct {
	date time.Time
}

func newBeforeMatch(date time.Time) BeforeMatch {
	return BeforeMatch{date: date}
}

func (b BeforeMatch) GetDate() time.Time {
	return b.date
}

type AfterMatch struct {
	date time.Time
}

func newAfterMatch(date time.Time) AfterMatch {
	return AfterMatch{date: date}
}

func (a AfterMatch) GetDate() time.Time {
	return a.date
}

type FromMatch struct {
	from string
}

func newFromMatch(from string) FromMatch {
	return FromMatch{from: from}
}

func (f FromMatch) GetFrom() string {
	return f.from
}

type NewerThanMatch struct {
	duration time.Duration
}

func newNewerThanMatch(duration time.Duration) NewerThanMatch {
	return NewerThanMatch{duration: duration}
}

func (n NewerThanMatch) GetDuration() time.Duration {
	return n.duration
}

type OlderThanMatch struct {
	duration time.Duration
}

func newOlderThanMatch(duration time.Duration) OlderThanMatch {
	return OlderThanMatch{duration: duration}
}

func (o OlderThanMatch) GetDuration() time.Duration {
	return o.duration
}

type SubjectMatch struct {
	subject string
}

func newSubjectMatch(subject string) SubjectMatch {
	return SubjectMatch{subject: subject}
}

func (s SubjectMatch) GetSubject() string {
	return s.subject
}
