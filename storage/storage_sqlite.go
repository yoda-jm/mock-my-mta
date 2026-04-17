package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/mail"
	"sort"
	"strings"

	"mock-my-mta/log"
	"mock-my-mta/storage/matcher"
	"mock-my-mta/storage/multipart"

	_ "modernc.org/sqlite"
)

type sqliteStorage struct {
	db               *sql.DB
	databaseFilename string
}

// sqliteStorage implements the storageLayer interface
var _ storageLayer = &sqliteStorage{}

func newSqliteStorage(databaseFilename string) (*sqliteStorage, error) {
	log.Logf(log.INFO, "using sqlite storage with database %v", databaseFilename)

	db, err := sql.Open("sqlite", databaseFilename)
	if err != nil {
		return nil, fmt.Errorf("cannot open sqlite database %s: %v", databaseFilename, err)
	}

	// Enable WAL mode for better concurrent read performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Logf(log.WARNING, "sqlite: could not enable WAL mode: %v", err)
	}

	// Create tables
	if err := createTables(db); err != nil {
		return nil, err
	}

	return &sqliteStorage{db: db, databaseFilename: databaseFilename}, nil
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS emails (
			id TEXT PRIMARY KEY,
			sender_name TEXT DEFAULT '',
			sender_address TEXT DEFAULT '',
			subject TEXT DEFAULT '',
			date DATETIME,
			has_attachments BOOLEAN DEFAULT FALSE,
			preview TEXT DEFAULT '',
			recipients_json TEXT DEFAULT '[]',
			ccs_json TEXT DEFAULT '[]',
			body_versions_json TEXT DEFAULT '[]',
			raw_email BLOB
		);
		CREATE INDEX IF NOT EXISTS idx_emails_date ON emails(date);
		CREATE INDEX IF NOT EXISTS idx_emails_sender ON emails(sender_address);
		CREATE INDEX IF NOT EXISTS idx_emails_subject ON emails(subject);
	`)
	return err
}

// load hydrates from root storage (if this is not the root).
func (s *sqliteStorage) load(rootStorage Storage) error {
	if rootStorage == nil {
		return nil // we are root
	}

	// Check if we already have data (persistent — survives restart)
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM emails").Scan(&count)
	if count > 0 {
		log.Logf(log.INFO, "sqlite storage: already has %d emails, skipping reload from root", count)
		return nil
	}

	log.Logf(log.INFO, "sqlite storage: loading from root storage")
	emails, _, err := rootStorage.SearchEmails("", 1, -1)
	if err != nil {
		log.Logf(log.WARNING, "sqlite storage: could not load from root: %v", err)
		return nil
	}

	for _, header := range emails {
		raw, _ := rootStorage.GetRawEmail(header.ID)
		s.insertEmailHeader(header, raw)
	}

	log.Logf(log.INFO, "sqlite storage: loaded %d emails from root", len(emails))
	return nil
}

func (s *sqliteStorage) insertEmailHeader(header EmailHeader, raw []byte) error {
	recipientsJSON, _ := json.Marshal(header.Tos)
	ccsJSON, _ := json.Marshal(header.CCs)
	versionsJSON, _ := json.Marshal(header.BodyVersions)

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO emails (id, sender_name, sender_address, subject, date, has_attachments, preview, recipients_json, ccs_json, body_versions_json, raw_email)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		header.ID, header.From.Name, header.From.Address, header.Subject,
		header.Date, header.HasAttachments, header.Preview,
		string(recipientsJSON), string(ccsJSON), string(versionsJSON), raw,
	)
	return err
}

// setWithID stores email metadata and raw bytes in SQLite.
func (s *sqliteStorage) setWithID(emailID string, rawEmail []byte) error {
	msg, err := mail.ReadMessage(strings.NewReader(string(rawEmail)))
	if err != nil {
		return fmt.Errorf("sqlite storage: cannot parse email %s: %v", emailID, err)
	}

	mp, err := multipart.New(msg)
	if err != nil {
		return fmt.Errorf("sqlite storage: cannot parse email %s: %v", emailID, err)
	}

	header := newEmailHeaderFromMultipart(emailID, mp)
	return s.insertEmailHeader(header, rawEmail)
}

// --- Search methods ---

func (s *sqliteStorage) SearchEmails(query string, page int, pageSize int) ([]EmailHeader, int, error) {
	if page < 1 {
		return nil, 0, fmt.Errorf("invalid page number: %v", page)
	}

	// For complex queries with matchers, we fall back to loading all and filtering
	// because the matcher system uses multipart parsing not SQL
	matchers, err := matcher.ParseQuery(query)
	if err != nil {
		return nil, 0, err
	}

	// Simple case: no query — use SQL pagination directly
	if len(matchers) == 0 {
		return s.searchAllSQL(page, pageSize)
	}

	// Complex case: load all headers, filter with matchers using raw email
	return s.searchWithMatchers(matchers, page, pageSize)
}

func (s *sqliteStorage) searchAllSQL(page, pageSize int) ([]EmailHeader, int, error) {
	var total int
	s.db.QueryRow("SELECT COUNT(*) FROM emails").Scan(&total)

	var rows *sql.Rows
	var err error
	if pageSize < 0 {
		rows, err = s.db.Query("SELECT id, sender_name, sender_address, subject, date, has_attachments, preview, recipients_json, ccs_json, body_versions_json FROM emails ORDER BY date DESC")
	} else {
		offset := (page - 1) * pageSize
		rows, err = s.db.Query("SELECT id, sender_name, sender_address, subject, date, has_attachments, preview, recipients_json, ccs_json, body_versions_json FROM emails ORDER BY date DESC LIMIT ? OFFSET ?", pageSize, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return s.scanEmailHeaders(rows, total)
}

func (s *sqliteStorage) searchWithMatchers(matchers []interface{}, page, pageSize int) ([]EmailHeader, int, error) {
	rows, err := s.db.Query("SELECT id, raw_email FROM emails ORDER BY date DESC")
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var allResults []EmailHeader
	for rows.Next() {
		var id string
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			continue
		}
		if raw == nil {
			continue
		}
		msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
		if err != nil {
			continue
		}
		mp, err := multipart.New(msg)
		if err != nil {
			continue
		}
		if mp.MatchAll(matchers) {
			allResults = append(allResults, newEmailHeaderFromMultipart(id, mp))
		}
	}

	totalMatches := len(allResults)
	start := (page - 1) * pageSize
	end := start + pageSize
	if pageSize < 0 {
		end = len(allResults)
		start = 0
	}
	if start > len(allResults) {
		start = len(allResults)
	}
	if end > len(allResults) {
		end = len(allResults)
	}
	return allResults[start:end], totalMatches, nil
}

func (s *sqliteStorage) scanEmailHeaders(rows *sql.Rows, total int) ([]EmailHeader, int, error) {
	var headers []EmailHeader
	for rows.Next() {
		var h EmailHeader
		var recipientsJSON, ccsJSON, versionsJSON string
		err := rows.Scan(&h.ID, &h.From.Name, &h.From.Address, &h.Subject,
			&h.Date, &h.HasAttachments, &h.Preview,
			&recipientsJSON, &ccsJSON, &versionsJSON)
		if err != nil {
			continue
		}
		json.Unmarshal([]byte(recipientsJSON), &h.Tos)
		json.Unmarshal([]byte(ccsJSON), &h.CCs)
		json.Unmarshal([]byte(versionsJSON), &h.BodyVersions)
		headers = append(headers, h)
	}
	return headers, total, nil
}

func (s *sqliteStorage) GetMailboxes() ([]Mailbox, error) {
	rows, err := s.db.Query("SELECT DISTINCT recipients_json FROM emails")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recipients := make(map[string]bool)
	for rows.Next() {
		var recipientsJSON string
		if err := rows.Scan(&recipientsJSON); err != nil {
			continue
		}
		var addrs []EmailAddress
		json.Unmarshal([]byte(recipientsJSON), &addrs)
		for _, a := range addrs {
			if a.Address != "" {
				recipients[a.Address] = true
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

// --- Read methods (for root/all scope) ---

func (s *sqliteStorage) GetEmailByID(emailID string) (EmailHeader, error) {
	row := s.db.QueryRow("SELECT id, sender_name, sender_address, subject, date, has_attachments, preview, recipients_json, ccs_json, body_versions_json FROM emails WHERE id = ?", emailID)

	var h EmailHeader
	var recipientsJSON, ccsJSON, versionsJSON string
	err := row.Scan(&h.ID, &h.From.Name, &h.From.Address, &h.Subject,
		&h.Date, &h.HasAttachments, &h.Preview,
		&recipientsJSON, &ccsJSON, &versionsJSON)
	if err != nil {
		return EmailHeader{}, fmt.Errorf("email not found in sqlite: %s", emailID)
	}
	json.Unmarshal([]byte(recipientsJSON), &h.Tos)
	json.Unmarshal([]byte(ccsJSON), &h.CCs)
	json.Unmarshal([]byte(versionsJSON), &h.BodyVersions)
	return h, nil
}

func (s *sqliteStorage) GetBodyVersion(emailID string, version EmailVersionType) (string, error) {
	// SQLite stores raw email — parse and extract body version on demand
	raw, err := s.GetRawEmail(emailID)
	if err != nil {
		return "", err
	}
	if version == EmailVersionRaw {
		return string(raw), nil
	}
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		return "", err
	}
	mp, err := multipart.New(msg)
	if err != nil {
		return "", err
	}
	versionStr, _ := emailVersionToString(version)
	return mp.GetBody(versionStr)
}

func (s *sqliteStorage) GetAttachments(emailID string) ([]AttachmentHeader, error) {
	raw, err := s.GetRawEmail(emailID)
	if err != nil {
		return nil, err
	}
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}
	mp, err := multipart.New(msg)
	if err != nil {
		return nil, err
	}
	var headers []AttachmentHeader
	for id, node := range mp.GetAttachments() {
		headers = append(headers, AttachmentHeader{
			ID: id, ContentType: node.GetContentType(),
			Filename: node.GetFilename(), Size: node.GetSize(),
		})
	}
	return headers, nil
}

func (s *sqliteStorage) GetAttachment(emailID string, attachmentID string) (Attachment, error) {
	raw, err := s.GetRawEmail(emailID)
	if err != nil {
		return Attachment{}, err
	}
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		return Attachment{}, err
	}
	mp, err := multipart.New(msg)
	if err != nil {
		return Attachment{}, err
	}
	node, found := mp.GetAttachment(attachmentID)
	if !found {
		return Attachment{}, fmt.Errorf("attachment not found: %s/%s", emailID, attachmentID)
	}
	return Attachment{
		AttachmentHeader: AttachmentHeader{
			ID: attachmentID, ContentType: node.GetContentType(),
			Filename: node.GetFilename(), Size: node.GetSize(),
		},
		Data: []byte(node.GetDecodedBody()),
	}, nil
}

func (s *sqliteStorage) GetRawEmail(emailID string) ([]byte, error) {
	var raw []byte
	err := s.db.QueryRow("SELECT raw_email FROM emails WHERE id = ?", emailID).Scan(&raw)
	if err != nil {
		return nil, fmt.Errorf("raw email not found in sqlite: %s", emailID)
	}
	return raw, nil
}

// --- Write methods ---

func (s *sqliteStorage) DeleteAllEmails() error {
	_, err := s.db.Exec("DELETE FROM emails")
	return err
}

func (s *sqliteStorage) DeleteEmailByID(emailID string) error {
	result, err := s.db.Exec("DELETE FROM emails WHERE id = ?", emailID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("email not found in sqlite: %s", emailID)
	}
	return nil
}

// emailVersionToString converts EmailVersionType to string.
func emailVersionToString(v EmailVersionType) (string, error) {
	switch v {
	case EmailVersionRaw:
		return "raw", nil
	case EmailVersionPlainText:
		return "plain-text", nil
	case EmailVersionHtml:
		return "html", nil
	case EmailVersionWatchHtml:
		return "watch-html", nil
	default:
		return "", fmt.Errorf("unknown version type: %d", v)
	}
}
