package storage

import (
	"mock-my-mta/log"
	"net/mail"
)

type memoryStorage struct {
}

// MemoryStorage implements the Storage interface
var _ Storage = &memoryStorage{}

func newMemoryStorage() (*memoryStorage, error) {
	log.Logf(log.INFO, "using memory storage")
	return &memoryStorage{}, nil
}

// DeleteEmailByID implements Storage.
func (*memoryStorage) DeleteAllEmails() error {
	return newUnimplementedMethodInLayerError("DeleteAllEmails", "memoryStorage")
}

// DeleteEmailByID implements Storage.
func (*memoryStorage) DeleteEmailByID(emailID string) error {
	return newUnimplementedMethodInLayerError("DeleteEmailByID", "memoryStorage")
}

// GetAttachment implements Storage.
func (*memoryStorage) GetAttachment(emailID string, attachmentID string) (Attachment, error) {
	return Attachment{}, newUnimplementedMethodInLayerError("GetAttachment", "memoryStorage")
}

// GetAttachments implements Storage.
func (*memoryStorage) GetAttachments(emailID string) ([]AttachmentHeader, error) {
	return nil, newUnimplementedMethodInLayerError("GetAttachments", "memoryStorage")
}

// GetBodyVersion implements Storage.
func (*memoryStorage) GetBodyVersion(emailID string, version EmailVersionType) (string, error) {
	return "", newUnimplementedMethodInLayerError("GetBodyVersion", "memoryStorage")
}

// GetEmailByID implements Storage.
func (*memoryStorage) GetEmailByID(emailID string) (EmailHeader, error) {
	return EmailHeader{}, newUnimplementedMethodInLayerError("GetEmailByID", "memoryStorage")
}

// GetMailboxes implements Storage.
func (*memoryStorage) GetMailboxes() ([]Mailbox, error) {
	return nil, newUnimplementedMethodInLayerError("GetMailboxes", "memoryStorage")
}

// SearchEmails implements Storage.
func (*memoryStorage) SearchEmails(query string, page int, pageSize int) ([]EmailHeader, int, error) {
	return nil, 0, newUnimplementedMethodInLayerError("SearchEmails", "memoryStorage")
}

// Load loads the storage based on the root storage
func (*memoryStorage) load(rootStorage Storage) error {
	// FIXME: implement
	return nil
}

// setWithID inserts a new email into the storage.
func (*memoryStorage) setWithID(emailID string, message *mail.Message) error {
	return newUnimplementedMethodInLayerError("setWithID", "memoryStorage")
}
