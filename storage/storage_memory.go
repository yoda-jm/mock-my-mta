package storage

import (
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"sort"
	"strings"
	"sync"

	"mock-my-mta/log"
	"mock-my-mta/storage/matcher"
	"mock-my-mta/storage/multipart"
)

// memoryStorage is an in-memory cache for parsed email data.
// It stores pre-parsed EmailHeaders, body versions, and attachment metadata
// so that reads don't require re-parsing .eml files from disk.
//
// Volatile: all data is lost on restart and rebuilt via load(rootStorage).
type memoryStorage struct {
	mu          sync.RWMutex
	headers     map[string]EmailHeader
	bodies      map[string]map[EmailVersionType]string
	attachments map[string][]AttachmentHeader
	attachment  map[string]Attachment // keyed by "emailID/attachmentID"
	rawEmails   map[string][]byte     // raw .eml bytes (for root mode or raw scope)
}

// memoryStorage implements the storageLayer interface
var _ storageLayer = &memoryStorage{}

func newMemoryStorage() (*memoryStorage, error) {
	log.Logf(log.INFO, "using memory storage")
	return &memoryStorage{
		headers:     make(map[string]EmailHeader),
		bodies:      make(map[string]map[EmailVersionType]string),
		attachments: make(map[string][]AttachmentHeader),
		attachment:  make(map[string]Attachment),
		rawEmails:   make(map[string][]byte),
	}, nil
}

// load hydrates the memory cache from the root storage layer.
func (m *memoryStorage) load(rootStorage Storage) error {
	if rootStorage == nil {
		return nil // we are the root — nothing to load from
	}

	log.Logf(log.INFO, "memory storage: loading from root storage")

	// Fetch all emails from root (no pagination limit)
	emails, _, err := rootStorage.SearchEmails("", 1, -1)
	if err != nil {
		log.Logf(log.WARNING, "memory storage: could not load from root: %v", err)
		return nil // non-fatal — we'll populate on writes
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, header := range emails {
		m.headers[header.ID] = header

		// Load body versions
		m.bodies[header.ID] = make(map[EmailVersionType]string)
		for _, versionName := range header.BodyVersions {
			version, err := ParseEmailVersionType(versionName)
			if err != nil {
				continue
			}
			body, err := rootStorage.GetBodyVersion(header.ID, version)
			if err != nil {
				continue
			}
			m.bodies[header.ID][version] = body
		}

		// Load attachments
		atts, err := rootStorage.GetAttachments(header.ID)
		if err == nil {
			m.attachments[header.ID] = atts
			for _, att := range atts {
				fullAtt, err := rootStorage.GetAttachment(header.ID, att.ID)
				if err == nil {
					m.attachment[header.ID+"/"+att.ID] = fullAtt
				}
			}
		}
	}

	// Load raw emails
	for id := range m.headers {
		raw, err := rootStorage.GetRawEmail(id)
		if err == nil {
			m.rawEmails[id] = raw
		}
	}

	log.Logf(log.INFO, "memory storage: loaded %d emails from root", len(m.headers))
	return nil
}

// setWithID parses the email and caches all derived data.
func (m *memoryStorage) setWithID(emailID string, message *mail.Message) error {
	// Serialize raw email first (consumes message.Body)
	rawBytes, err := serializeMessage(message)
	if err != nil {
		log.Logf(log.WARNING, "memory storage: could not serialize raw email %s: %v", emailID, err)
	}

	// Re-parse from raw bytes since message.Body was consumed
	freshMsg, err := mail.ReadMessage(bytes.NewReader(rawBytes))
	if err != nil {
		return fmt.Errorf("memory storage: cannot re-parse email %s: %v", emailID, err)
	}

	mp, err := multipart.New(freshMsg)
	if err != nil {
		return fmt.Errorf("memory storage: cannot parse email %s: %v", emailID, err)
	}

	header := newEmailHeaderFromMultipart(emailID, mp)

	// Cache body versions
	bodies := make(map[EmailVersionType]string)
	for _, versionName := range header.BodyVersions {
		if versionName == "raw" {
			continue // raw is served from filesystem
		}
		version, err := ParseEmailVersionType(versionName)
		if err != nil {
			continue
		}
		body, err := mp.GetBody(versionName)
		if err == nil {
			bodies[version] = body
		}
	}

	// Cache attachments
	var attHeaders []AttachmentHeader
	for attID, node := range mp.GetAttachments() {
		attHeaders = append(attHeaders, AttachmentHeader{
			ID:          attID,
			ContentType: node.GetContentType(),
			Filename:    node.GetFilename(),
			Size:        node.GetSize(),
		})
		m.mu.Lock()
		m.attachment[emailID+"/"+attID] = Attachment{
			AttachmentHeader: AttachmentHeader{
				ID:          attID,
				ContentType: node.GetContentType(),
				Filename:    node.GetFilename(),
				Size:        node.GetSize(),
			},
			Data: []byte(node.GetDecodedBody()),
		}
		m.mu.Unlock()
	}

	m.mu.Lock()
	m.headers[emailID] = header
	m.bodies[emailID] = bodies
	m.attachments[emailID] = attHeaders
	if rawBytes != nil {
		m.rawEmails[emailID] = rawBytes
	}
	m.mu.Unlock()

	return nil
}

// --- Read methods ---

func (m *memoryStorage) GetEmailByID(emailID string) (EmailHeader, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	header, ok := m.headers[emailID]
	if !ok {
		return EmailHeader{}, fmt.Errorf("email not found in memory cache: %s", emailID)
	}
	return header, nil
}

func (m *memoryStorage) GetBodyVersion(emailID string, version EmailVersionType) (string, error) {
	if version == EmailVersionRaw {
		// Raw is served from filesystem — not cached in memory
		return "", newUnimplementedMethodInLayerError("GetBodyVersion(raw)", "memoryStorage")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	versions, ok := m.bodies[emailID]
	if !ok {
		return "", fmt.Errorf("email not found in memory cache: %s", emailID)
	}
	body, ok := versions[version]
	if !ok {
		return "", nil // version doesn't exist for this email
	}
	return body, nil
}

func (m *memoryStorage) GetAttachments(emailID string) ([]AttachmentHeader, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	atts, ok := m.attachments[emailID]
	if !ok {
		return nil, fmt.Errorf("email not found in memory cache: %s", emailID)
	}
	return atts, nil
}

func (m *memoryStorage) GetAttachment(emailID string, attachmentID string) (Attachment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	att, ok := m.attachment[emailID+"/"+attachmentID]
	if !ok {
		return Attachment{}, fmt.Errorf("attachment not found in memory cache: %s/%s", emailID, attachmentID)
	}
	return att, nil
}

// --- Write methods ---

func (m *memoryStorage) DeleteAllEmails() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.headers = make(map[string]EmailHeader)
	m.bodies = make(map[string]map[EmailVersionType]string)
	m.attachments = make(map[string][]AttachmentHeader)
	m.attachment = make(map[string]Attachment)
	m.rawEmails = make(map[string][]byte)
	return nil
}

func (m *memoryStorage) DeleteEmailByID(emailID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Clean up attachment entries
	if atts, ok := m.attachments[emailID]; ok {
		for _, att := range atts {
			delete(m.attachment, emailID+"/"+att.ID)
		}
	}
	delete(m.headers, emailID)
	delete(m.bodies, emailID)
	delete(m.attachments, emailID)
	delete(m.rawEmails, emailID)
	return nil
}

// --- Not implemented (handled by other layers) ---

func (m *memoryStorage) SearchEmails(query string, page int, pageSize int) ([]EmailHeader, int, error) {
	matchers, err := matcher.ParseQuery(query)
	if err != nil {
		return nil, 0, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []EmailHeader
	for id := range m.headers {
		raw, ok := m.rawEmails[id]
		if !ok {
			continue
		}
		msg, err := mail.ReadMessage(bytes.NewReader(raw))
		if err != nil {
			continue
		}
		mp, err := multipart.New(msg)
		if err != nil {
			continue
		}
		if mp.MatchAll(matchers) {
			results = append(results, m.headers[id])
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Date.After(results[j].Date)
	})

	totalMatches := len(results)
	if page < 1 {
		return nil, 0, fmt.Errorf("invalid page number: %v", page)
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if pageSize < 0 {
		end = len(results)
	}
	if start > len(results) {
		start = len(results)
	}
	if end > len(results) {
		end = len(results)
	}
	return results[start:end], totalMatches, nil
}

func (m *memoryStorage) GetMailboxes() ([]Mailbox, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	recipients := make(map[string]bool)
	for _, header := range m.headers {
		for _, to := range header.Tos {
			if to.Address != "" {
				recipients[to.Address] = true
			}
		}
	}

	mailboxes := make([]Mailbox, 0, len(recipients))
	for addr := range recipients {
		mailboxes = append(mailboxes, Mailbox{Name: addr})
	}
	sort.Slice(mailboxes, func(i, j int) bool {
		return mailboxes[i].Name < mailboxes[j].Name
	})
	return mailboxes, nil
}

func (m *memoryStorage) GetRawEmail(emailID string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	raw, ok := m.rawEmails[emailID]
	if !ok {
		return nil, fmt.Errorf("raw email not found in memory cache: %s", emailID)
	}
	return raw, nil
}

// newEmailHeaderFromMultipart builds an EmailHeader from a parsed Multipart.
// This is duplicated from storage_filesystem.go to avoid import cycles.
func newEmailHeaderFromMultipart(ID string, mp *multipart.Multipart) EmailHeader {
	from := mp.GetFrom()
	tos := mp.GetTos()
	ccs := mp.GetCCs()

	tosAddrs := make([]EmailAddress, len(tos))
	for i, a := range tos {
		tosAddrs[i] = EmailAddress{Name: a.Name, Address: a.Address}
	}
	ccsAddrs := make([]EmailAddress, len(ccs))
	for i, a := range ccs {
		ccsAddrs[i] = EmailAddress{Name: a.Name, Address: a.Address}
	}

	return EmailHeader{
		ID:             ID,
		From:           EmailAddress{Name: from.Name, Address: from.Address},
		Tos:            tosAddrs,
		CCs:            ccsAddrs,
		Subject:        mp.GetSubject(),
		Date:           mp.GetDate(),
		HasAttachments: mp.HasAttachments(),
		Preview:        mp.GetPreview(),
		BodyVersions:   append(mp.GetBodyVersions(), "raw"),
	}
}

// serializeMessage converts a mail.Message back to raw bytes.
func serializeMessage(message *mail.Message) ([]byte, error) {
	var buf bytes.Buffer
	for key, values := range message.Header {
		for _, value := range values {
			buf.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}
	}
	buf.WriteString("\n")
	if message.Body != nil {
		_, err := io.Copy(&buf, message.Body)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// containsString checks if a string slice contains a value (case-insensitive).
func containsString(slice []string, val string) bool {
	val = strings.ToLower(val)
	for _, s := range slice {
		if strings.ToLower(s) == val {
			return true
		}
	}
	return false
}
