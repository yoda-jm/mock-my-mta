package storage

import "net/mail"

// StorageService defines the interface for storage operations that smtp.Server depends on.
// This includes methods for storing messages and potentially other methods from the Storage interface.
type StorageService interface {
	// Set stores an email message and returns its UUID or an error.
	Set(message *mail.Message) (string, error)

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
