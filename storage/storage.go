package storage

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"mock-my-mta/email"
)

// EmailData represents the received time, UUID, and email.
type EmailData struct {
	ID           uuid.UUID
	ReceivedTime time.Time
	Email        *email.Email
}

type ErrNotFound struct {
	id uuid.UUID
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("email with ID %v not found", e.id)
}

type Storage interface {
	Get(id uuid.UUID) (*EmailData, error)
	Set(message []byte) error
	Find(matchOptions email.MatchOption, sortOptions SortOption, value string) ([]uuid.UUID, error)
	Delete(id uuid.UUID) error
}

type PhysicalLayer interface {
	Populate(underlying PhysicalLayer, parameters map[string]string) error
	List() ([]uuid.UUID, error)

	Read(uuid.UUID) (*EmailData, error)
	Write(*EmailData) error
	Find(matchOptions email.MatchOption, sortOptions SortOption, value string) ([]uuid.UUID, error)
	Delete(uuid.UUID) error
}

// GetAll retrieves all UUIDs of email data in the storage sorted according to sorting options.
func GetAll(storage Storage, sortOptions SortOption) ([]uuid.UUID, error) {
	return storage.Find(email.MatchOption{Type: email.AllMatch}, sortOptions, "")
}
