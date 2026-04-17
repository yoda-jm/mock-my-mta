package storage

import (
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
)

// Engine orchestrates multiple storage layers with scope-based routing.
// Each method is routed to the subset of layers that declared the relevant scope.
type Engine struct {
	allLayers    []storageLayer // all layers in config order
	readLayers   []storageLayer // scope: read — GetEmailByID, GetBodyVersion, GetAttachments, GetAttachment
	searchLayers []storageLayer // scope: search — SearchEmails, GetMailboxes
	writeLayers  []storageLayer // scope: write/cache/all — DeleteEmailByID, DeleteAllEmails, Set
	rawLayers    []storageLayer // scope: raw — GetRawEmail
}

// Engine must implement the Storage interface
var _ Storage = &Engine{}

// error that says this layer does not implement the method
type unimplementedMethodInLayerError struct {
	methodName    string
	physicalLayer string
}

func (e *unimplementedMethodInLayerError) Error() string {
	return fmt.Sprintf("method %s is not implemented in layer %s", e.methodName, e.physicalLayer)
}

func newUnimplementedMethodInLayerError(methodName string, physicalLayer string) error {
	return &unimplementedMethodInLayerError{methodName: methodName, physicalLayer: physicalLayer}
}

func isUnimplemented(err error) bool {
	_, ok := err.(*unimplementedMethodInLayerError)
	return ok
}

func NewEngine(storagesConfiguration []StorageLayerConfiguration) (*Engine, error) {
	engine := &Engine{}

	// Backward compatibility: if no scope specified, default to "all"
	for i := range storagesConfiguration {
		if len(storagesConfiguration[i].Scope) == 0 {
			storagesConfiguration[i].Scope = []string{ScopeAll}
		}
	}

	// Construct storage layers
	for _, cfg := range storagesConfiguration {
		var layer storageLayer
		var err error

		switch cfg.Type {
		case "MEMORY":
			layer, err = newMemoryStorage()
		case "SQLITE":
			dbFile, ok := cfg.Parameters["database"]
			if !ok {
				return nil, fmt.Errorf("missing database parameter for SQLITE storage")
			}
			layer, err = newSqliteStorage(dbFile)
		case "FILESYSTEM":
			folder, ok := cfg.Parameters["folder"]
			if !ok {
				return nil, fmt.Errorf("missing folder parameter for FILESYSTEM storage")
			}
			fsType, ok := cfg.Parameters["type"]
			if !ok {
				return nil, fmt.Errorf("missing type parameter for FILESYSTEM storage")
			}
			layer, err = newFilesystemStorage(folder, fsType)
		default:
			return nil, fmt.Errorf("unknown storage type: %s", cfg.Type)
		}

		if err != nil {
			return nil, err
		}

		engine.allLayers = append(engine.allLayers, layer)

		// Build per-scope routing tables
		if cfg.hasScope(ScopeRead) {
			engine.readLayers = append(engine.readLayers, layer)
		}
		if cfg.hasScope(ScopeSearch) {
			engine.searchLayers = append(engine.searchLayers, layer)
		}
		if cfg.isWritable() {
			engine.writeLayers = append(engine.writeLayers, layer)
		}
		if cfg.hasScope(ScopeRaw) {
			engine.rawLayers = append(engine.rawLayers, layer)
		}
	}

	// Load layers: root (last) first, then others hydrate from root
	if len(engine.allLayers) > 0 {
		rootLayer := engine.allLayers[len(engine.allLayers)-1]
		if err := engine.load(rootLayer); err != nil {
			return nil, err
		}
	}

	return engine, nil
}

// load initializes all layers. The root layer (last) loads with nil;
// all other layers receive the root so they can hydrate from it.
func (e *Engine) load(rootStorage Storage) error {
	for i := len(e.allLayers) - 1; i >= 0; i-- {
		layer := e.allLayers[i]
		if i == len(e.allLayers)-1 {
			if err := layer.load(nil); err != nil {
				return err
			}
		} else {
			if err := layer.load(rootStorage); err != nil {
				return err
			}
		}
	}
	return nil
}

// --- Read scope (first-match-wins) ---

func (e *Engine) GetEmailByID(emailID string) (EmailHeader, error) {
	for _, s := range e.readLayers {
		result, err := s.GetEmailByID(emailID)
		if err != nil {
			if isUnimplemented(err) {
				continue
			}
			return EmailHeader{}, err
		}
		return result, nil
	}
	return EmailHeader{}, fmt.Errorf("no storage layer implements GetEmailByID")
}

func (e *Engine) GetBodyVersion(emailID string, version EmailVersionType) (string, error) {
	for _, s := range e.readLayers {
		body, err := s.GetBodyVersion(emailID, version)
		if err != nil {
			if isUnimplemented(err) {
				continue
			}
			return "", err
		}
		return body, nil
	}
	return "", fmt.Errorf("no storage layer implements GetBodyVersion")
}

func (e *Engine) GetAttachments(emailID string) ([]AttachmentHeader, error) {
	for _, s := range e.readLayers {
		atts, err := s.GetAttachments(emailID)
		if err != nil {
			if isUnimplemented(err) {
				continue
			}
			return nil, err
		}
		return atts, nil
	}
	return nil, fmt.Errorf("no storage layer implements GetAttachments")
}

func (e *Engine) GetAttachment(emailID string, attachmentID string) (Attachment, error) {
	for _, s := range e.readLayers {
		att, err := s.GetAttachment(emailID, attachmentID)
		if err != nil {
			if isUnimplemented(err) {
				continue
			}
			return Attachment{}, err
		}
		return att, nil
	}
	return Attachment{}, fmt.Errorf("no storage layer implements GetAttachment")
}

// --- Search scope (first-match-wins) ---

func (e *Engine) SearchEmails(query string, page int, pageSize int) ([]EmailHeader, int, error) {
	for _, s := range e.searchLayers {
		headers, total, err := s.SearchEmails(query, page, pageSize)
		if err != nil {
			if isUnimplemented(err) {
				continue
			}
			return nil, 0, err
		}
		return headers, total, nil
	}
	return nil, 0, fmt.Errorf("no storage layer implements SearchEmails")
}

func (e *Engine) GetMailboxes() ([]Mailbox, error) {
	for _, s := range e.searchLayers {
		mailboxes, err := s.GetMailboxes()
		if err != nil {
			if isUnimplemented(err) {
				continue
			}
			return nil, err
		}
		return mailboxes, nil
	}
	return nil, fmt.Errorf("no storage layer implements GetMailboxes")
}

// --- Raw scope (first-match-wins) ---

func (e *Engine) GetRawEmail(emailID string) ([]byte, error) {
	for _, s := range e.rawLayers {
		raw, err := s.GetRawEmail(emailID)
		if err != nil {
			if isUnimplemented(err) {
				continue
			}
			return nil, err
		}
		return raw, nil
	}
	return nil, fmt.Errorf("no storage layer implements GetRawEmail")
}

// --- Write scope (propagate to all writable layers) ---

func (e *Engine) DeleteAllEmails() error {
	var errors []error
	for _, s := range e.writeLayers {
		err := s.DeleteAllEmails()
		if err != nil {
			if isUnimplemented(err) {
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

func (e *Engine) DeleteEmailByID(emailID string) error {
	for _, s := range e.writeLayers {
		err := s.DeleteEmailByID(emailID)
		if err != nil {
			if isUnimplemented(err) {
				continue
			}
			return err
		}
	}
	return nil
}

// Set inserts a new email into the storage. Writes to ALL writable layers.
// The message body is serialized to bytes once; layers receive the immutable
// []byte and parse only what they need (zero-copy for filesystem writes).
func (e *Engine) Set(message *mail.Message) (string, error) {
	emailID := uuid.New().String()
	if _, exists := message.Header["Date"]; !exists {
		message.Header["Date"] = []string{time.Now().Format(time.RFC1123Z)}
	}
	date, err := message.Header.Date()
	if err != nil {
		date = time.Now()
		message.Header["Date"] = []string{date.Format(time.RFC1123Z)}
	}
	emailID = date.Format(time.RFC3339) + "-" + emailID

	// Serialize once — all layers share this immutable byte slice
	rawBytes, err := serializeMessage(message)
	if err != nil {
		return "", fmt.Errorf("cannot serialize email: %v", err)
	}

	return emailID, e.setWithID(emailID, rawBytes)
}

func (e *Engine) setWithID(emailID string, rawEmail []byte) error {
	for _, s := range e.writeLayers {
		err := s.setWithID(emailID, rawEmail)
		if err != nil {
			if isUnimplemented(err) {
				continue
			}
			return err
		}
	}
	return nil
}
