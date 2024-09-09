package storage

import (
	"fmt"
	"net/mail"
	"time"
)

// Storage is an interface that defines the methods that a storage engine must implement.
type Storage interface {
	// GetMailboxes returns a list of mailboxes.
	GetMailboxes() ([]Mailbox, error)

	// GetEmailByID returns an email by ID.
	GetEmailByID(emailID string) (EmailHeader, error)
	// DeleteAllEmails deletes all emails.
	DeleteAllEmails() error
	// DeleteEmailByID deletes an email by ID.
	DeleteEmailByID(emailID string) error
	// GetBodyVersion returns the body of an email by ID and version.
	GetBodyVersion(emailID string, version EmailVersionType) (string, error)

	// GetAttachments returns a list of attachments for an email.
	GetAttachments(emailID string) ([]AttachmentHeader, error)
	// GetAttachment returns an attachment by ID.
	GetAttachment(emailID string, attachmentID string) (Attachment, error)

	// SearchEmails searches for emails with pagination. It also returns the total number of matches.
	SearchEmails(query string, page, pageSize int) ([]EmailHeader, int, error)
}

type storageLayer interface {
	Storage

	// load loads the storage based on the root storage
	load(rootStorage Storage) error
	// setWithID inserts a new email into the storage.
	setWithID(emailID string, message *mail.Message) error
}

type Mailbox struct {
	Name string `json:"name"`
}

type EmailAddress struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type EmailHeader struct {
	ID             string         `json:"id"`
	From           EmailAddress   `json:"from"`
	Tos            []EmailAddress `json:"tos"`
	CCs            []EmailAddress `json:"ccs"`
	Subject        string         `json:"subject"`
	Date           time.Time      `json:"date"`
	HasAttachments bool           `json:"has_attachments"`
	Preview        string         `json:"preview"`
	BodyVersions   []string       `json:"body_versions"`
}

type AttachmentHeader struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
}

type Attachment struct {
	AttachmentHeader
	Data []byte `json:"-"`
}

// SortFieldEnum represents the available fields for sorting.
type EmailVersionType int

// Enum values for SortFieldEnum.
const (
	EmailVersionRaw EmailVersionType = iota
	EmailVersionPlainText
	EmailVersionHtml
	EmailVersionWatchHtml
)

func ParseEmailVersionType(str string) (EmailVersionType, error) {
	switch str {
	case "raw":
		return EmailVersionRaw, nil
	case "plain-text":
		return EmailVersionPlainText, nil
	case "html":
		return EmailVersionHtml, nil
	case "watch-html":
		return EmailVersionWatchHtml, nil
	default:
		return EmailVersionRaw, fmt.Errorf("cannot parse email version type %q", str)
	}
}
