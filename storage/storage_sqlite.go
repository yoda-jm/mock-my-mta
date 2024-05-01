package storage

import (
	"mock-my-mta/log"
	"net/mail"
)

type sqliteStorage struct {
	databaseFilename string
}

// SqliteStorage implements the Storage interface
var _ Storage = &sqliteStorage{}

func newSqliteStorage(databaseFilename string) (*sqliteStorage, error) {
	log.Logf(log.INFO, "using sqlite storage with database %v", databaseFilename)
	return &sqliteStorage{
		databaseFilename: databaseFilename,
	}, nil
}

// DeleteEmailByID implements Storage.
func (*sqliteStorage) DeleteEmailByID(emailID string) error {
	return newUnimplementedMethodInLayerError("DeleteEmailByID", "sqliteStorage")
}

// GetAttachment implements Storage.
func (*sqliteStorage) GetAttachment(emailID string, attachmentID string) (Attachment, error) {
	return Attachment{}, newUnimplementedMethodInLayerError("GetAttachment", "sqliteStorage")
}

// GetAttachments implements Storage.
func (*sqliteStorage) GetAttachments(emailID string) ([]AttachmentHeader, error) {
	return nil, newUnimplementedMethodInLayerError("GetAttachments", "sqliteStorage")
}

// GetBodyVersion implements Storage.
func (*sqliteStorage) GetBodyVersion(emailID string, version EmailVersionType) (string, error) {
	return "", newUnimplementedMethodInLayerError("GetBodyVersion", "sqliteStorage")
}

// GetEmailByID implements Storage.
func (*sqliteStorage) GetEmailByID(emailID string) (EmailHeader, error) {
	return EmailHeader{}, newUnimplementedMethodInLayerError("GetEmailByID", "sqliteStorage")
}

// GetMailboxes implements Storage.
func (*sqliteStorage) GetMailboxes() ([]Mailbox, error) {
	return nil, newUnimplementedMethodInLayerError("GetMailboxes", "sqliteStorage")
}

// SearchEmails implements Storage.
func (*sqliteStorage) SearchEmails(query string, page int, pageSize int) ([]EmailHeader, int, error) {
	return nil, 0, newUnimplementedMethodInLayerError("SearchEmails", "sqliteStorage")
}

// Load loads the storage based on the root storage
func (*sqliteStorage) load(rootStorage Storage) error {
	// FIXME: implement
	return nil
}

// setWithID inserts a new email into the storage.
func (*sqliteStorage) setWithID(emailID string, message *mail.Message) error {
	return newUnimplementedMethodInLayerError("setWithID", "sqliteStorage")
}
