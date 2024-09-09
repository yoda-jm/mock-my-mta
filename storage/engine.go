package storage

import (
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
)

type Engine struct {
	storages []storageLayer
}

// Engine must implement the Storage interface
var _ Storage = &Engine{}

// error that say that this layer does not implement the method
type unimplementedMethodInLayerError struct {
	methodName    string
	phytisalLayer string
}

func (e *unimplementedMethodInLayerError) Error() string {
	return fmt.Sprintf("method %s is not implemented in layer %s", e.methodName, e.phytisalLayer)
}

func newUnimplementedMethodInLayerError(methodName string, phytisalLayer string) error {
	return &unimplementedMethodInLayerError{methodName: methodName, phytisalLayer: phytisalLayer}
}

func NewEngine(storagesConfiguration []StorageLayerConfiguration) (*Engine, error) {
	engine := &Engine{}
	// construct storages
	for _, storage := range storagesConfiguration {
		switch storage.Type {
		case "MEMORY":
			physical, err := newMemoryStorage()
			if err != nil {
				return nil, err
			}
			engine.storages = append(engine.storages, physical)
		case "SQLITE":
			databaseFilename, ok := storage.Parameters["database"]
			if !ok {
				return nil, fmt.Errorf("missing database parameter for SQLITE storage")
			}
			physical, err := newSqliteStorage(databaseFilename)
			if err != nil {
				return nil, err
			}
			engine.storages = append(engine.storages, physical)
		case "FILESYSTEM":
			folder, ok := storage.Parameters["folder"]
			if !ok {
				return nil, fmt.Errorf("missing folder parameter for FILESYSTEM storage")
			}
			filesystemType, ok := storage.Parameters["type"]
			if !ok {
				return nil, fmt.Errorf("missing type parameter for FILESYSTEM storage")
			}
			physical, err := newFilesystemStorage(folder, filesystemType)
			if err != nil {
				return nil, err
			}
			engine.storages = append(engine.storages, physical)
		default:
			return nil, fmt.Errorf("unknown storage type: %s", storage.Type)
		}
	}
	// load the storages in reverse order
	rootLayer := engine.storages[len(engine.storages)-1]
	err := engine.load(rootLayer)
	if err != nil {
		return nil, err
	}
	return engine, nil
}

// Load loads the storage based on the root storage
func (e *Engine) load(rootStorage Storage) error {
	for i := len(e.storages) - 1; i >= 0; i-- {
		storage := e.storages[i]
		if i == len(e.storages)-1 {
			// the root layer is the last one
			if err := storage.load(nil); err != nil {
				return err
			}
		} else {
			if err := storage.load(rootStorage); err != nil {
				return err
			}
		}
	}
	return nil
}

// DeleteAllEmails implements Storage.
func (e *Engine) DeleteAllEmails() error {
	var errors []error
	for _, storage := range e.storages {
		err := storage.DeleteAllEmails()
		if err != nil {
			// check if the method is implemented
			if _, ok := err.(*unimplementedMethodInLayerError); ok {
				continue
			}
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("errors: %v", errors)
	}
	return nil
}

// DeleteEmailByID implements Storage.
func (e *Engine) DeleteEmailByID(emailID string) error {
	for _, storage := range e.storages {
		err := storage.DeleteEmailByID(emailID)
		if err != nil {
			// check if the method is implemented
			if _, ok := err.(*unimplementedMethodInLayerError); ok {
				continue
			}
			return err
		}
	}
	return nil
}

// GetAttachment implements Storage.
func (e *Engine) GetAttachment(emailID string, attachmentID string) (Attachment, error) {
	for _, storage := range e.storages {
		attachment, err := storage.GetAttachment(emailID, attachmentID)
		if err != nil {
			// check if the method is implemented
			if _, ok := err.(*unimplementedMethodInLayerError); ok {
				continue
			}
			return Attachment{}, err
		}
		return attachment, nil
	}
	return Attachment{}, fmt.Errorf("no storage layer implements GetAttachment")
}

// GetAttachments implements Storage.
func (e *Engine) GetAttachments(emailID string) ([]AttachmentHeader, error) {
	for _, storage := range e.storages {
		attachments, err := storage.GetAttachments(emailID)
		if err != nil {
			// check if the method is implemented
			if _, ok := err.(*unimplementedMethodInLayerError); ok {
				continue
			}
			return nil, err
		}
		return attachments, nil
	}
	return nil, fmt.Errorf("no storage layer implements GetAttachments")
}

// GetBodyVersion implements Storage.
func (e *Engine) GetBodyVersion(emailID string, version EmailVersionType) (string, error) {
	for _, storage := range e.storages {
		body, err := storage.GetBodyVersion(emailID, version)
		if err != nil {
			// check if the method is implemented
			if _, ok := err.(*unimplementedMethodInLayerError); ok {
				continue
			}
			return "", err
		}
		return body, nil
	}
	return "", fmt.Errorf("no storage layer implements GetBodyVersion")
}

// GetEmailByID implements Storage.
func (e *Engine) GetEmailByID(emailID string) (EmailHeader, error) {
	for _, storage := range e.storages {
		emailHeader, err := storage.GetEmailByID(emailID)
		if err != nil {
			// check if the method is implemented
			if _, ok := err.(*unimplementedMethodInLayerError); ok {
				continue
			}
			return EmailHeader{}, err
		}
		return emailHeader, nil
	}
	return EmailHeader{}, fmt.Errorf("no storage layer implements GetEmailByID")
}

// GetMailboxes implements Storage.
func (e *Engine) GetMailboxes() ([]Mailbox, error) {
	for _, storage := range e.storages {
		mailboxes, err := storage.GetMailboxes()
		if err != nil {
			// check if the method is implemented
			if _, ok := err.(*unimplementedMethodInLayerError); ok {
				continue
			}
			return nil, err
		}
		return mailboxes, nil
	}
	return nil, fmt.Errorf("no storage layer implements GetMailboxes")
}

// SearchEmails implements Storage.
func (e *Engine) SearchEmails(query string, page int, pageSize int) ([]EmailHeader, int, error) {
	for _, storage := range e.storages {
		emailHeaders, totalMatches, err := storage.SearchEmails(query, page, pageSize)
		if err != nil {
			// check if the method is implemented
			if _, ok := err.(*unimplementedMethodInLayerError); ok {
				continue
			}
			return nil, 0, err
		}
		return emailHeaders, totalMatches, nil
	}
	return nil, 0, fmt.Errorf("no storage layer implements SearchEmails")
}

// Set inserts a new email into the storage.
func (e *Engine) Set(message *mail.Message) error {
	// generate a new ID
	emailID := uuid.New().String()
	// if date header are not present, use the current time
	if _, exists := message.Header["Date"]; !exists {
		message.Header["Date"] = []string{time.Now().Format(time.RFC1123Z)}
	}
	// retrieve the date attribute as time.Time
	date, err := message.Header.Date()
	if err != nil {
		// use current time
		date = time.Now()
		message.Header["Date"] = []string{date.Format(time.RFC1123Z)}
	}
	// prefix ID with RFC date time
	emailID = date.Format(time.RFC3339) + "-" + emailID
	return e.setWithID(emailID, message)
}

// setWithID inserts a new email into the storage.
func (e *Engine) setWithID(emailID string, message *mail.Message) error {
	for _, storage := range e.storages {
		err := storage.setWithID(emailID, message)
		if err != nil {
			// check if the method is implemented
			if _, ok := err.(*unimplementedMethodInLayerError); ok {
				continue
			}
			return err
		}
	}
	return nil
}
