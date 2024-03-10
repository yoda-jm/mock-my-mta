package storage

import (
	"database/sql"
	"strings"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"github.com/google/uuid"

	"mock-my-mta/email"
	"mock-my-mta/log"
)

type sqlitePhysicalStorage struct {
	db *sql.DB
}

// check that the sqlightPhysicalStorage implements the PhysicalLayer interface
var _ PhysicalLayer = &sqlitePhysicalStorage{}

func newSqliteStorage() (*sqlitePhysicalStorage, error) {
	return &sqlitePhysicalStorage{}, nil
}

// Delete implements PhysicalLayer.
func (sp *sqlitePhysicalStorage) Delete(id uuid.UUID) error {
	_, err := sp.db.Exec("DELETE FROM emails WHERE id = ?", id.String())
	return err
}

// Find implements PhysicalLayer.
func (sp *sqlitePhysicalStorage) Find(matchOptions MatchOption, sortOptions SortOption, value string) ([]uuid.UUID, error) {
	// FIXME: optmise this
	return defaultSearch(sp, matchOptions, sortOptions, value)
}

// List implements PhysicalLayer.
func (sp *sqlitePhysicalStorage) List() ([]uuid.UUID, error) {
	rows, err := sp.db.Query("SELECT id FROM emails")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		uuid, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, uuid)
	}
	return ids, nil
}

// Populate implements PhysicalLayer.
func (sp *sqlitePhysicalStorage) Populate(underlying PhysicalLayer, parameters map[string]string) error {
	log.Logf(log.INFO, "populating sqlite layer")

	// Open the database
	databasePath, ok := parameters["database"]
	if !ok {
		return ErrMissingParameter{parameter: "database"}
	}
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return err
	}
	sp.db = db

	// Create the table
	_, err = sp.db.Exec("CREATE TABLE IF NOT EXISTS emails (id TEXT PRIMARY KEY, sender TEXT, recipients TEXT, subject TEXT, raw TEXT, date INTEGER, hasAttachment BOOLEAN)")
	if err != nil {
		return err
	}

	// clear the table
	_, err = sp.db.Exec("DELETE FROM emails")
	if err != nil {
		return err
	}

	// Populate database
	if underlying != nil {
		ids, err := underlying.List()
		if err != nil {
			return err
		}
		for _, id := range ids {
			emailData, err := underlying.Read(id)
			if err != nil {
				return err
			}
			sp.Write(emailData)
		}
	}
	return nil
}

// Read implements PhysicalLayer.
func (sp *sqlitePhysicalStorage) Read(id uuid.UUID) (*EmailData, error) {
	row := sp.db.QueryRow("SELECT raw, date FROM emails WHERE id = ?", id.String())
	var raw string
	var date int64
	if err := row.Scan(&raw, &date); err != nil {
		return nil, err
	}
	email, err := email.Parse([]byte(raw))
	if err != nil {
		return nil, err
	}
	return &EmailData{
		ID:           id,
		Email:        email,
		ReceivedTime: time.Unix(date, 0),
	}, nil
}

// Write implements PhysicalLayer.
func (sp *sqlitePhysicalStorage) Write(emailData *EmailData) error {
	raw, err := emailData.Email.GetBody(email.EmailVersionRaw)
	if err != nil {
		return err
	}
	receivedTime := emailData.ReceivedTime.Unix()
	subject := emailData.Email.GetSubject()
	sender := emailData.Email.GetSender()
	recipients := strings.Join(emailData.Email.GetRecipients(), ",")
	hasAttachment := len(emailData.Email.GetAttachments()) > 0
	_, err = sp.db.Exec("INSERT INTO emails (id, sender, recipients, subject, raw, date, hasAttachment) VALUES (?, ?, ?, ?, ?, ?, ?)",
		emailData.ID.String(), sender, recipients, subject, raw, receivedTime, hasAttachment)
	return err
}
