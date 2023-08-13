package storage

import (
	"time"

	"github.com/google/uuid"

	"mock-my-mta/email"
	"mock-my-mta/log"
)

// Storage represents a map of UUID to email data.
type Storage struct {
	physical Physical
	cache    Cache
}

// EmailData represents the received time, UUID, and email.
type EmailData struct {
	ID           uuid.UUID
	ReceivedTime time.Time
	Email        *email.Email
}

// NewStorage creates a new Storage instance by loading email data from the specified folder.
func NewStorage(physicalStr string) (*Storage, error) {
	physical, err := newPhysical(physicalStr)
	if err != nil {
		return nil, err
	}
	storage := &Storage{
		physical: physical,
		cache:    newMemoryCache(),
	}

	err = storage.cache.Fill(storage.physical)
	if err != nil {
		return nil, err
	}

	return storage, nil
}

// Set adds or updates the email data in the storage and saves it as JSON in the folder.
func (s *Storage) Set(message []byte) error {
	email, err := email.Parse(message)
	if err != nil {
		return err
	}
	emailData := EmailData{
		ID:           uuid.New(),
		ReceivedTime: time.Now(),
		Email:        email,
	}

	log.Logf(log.INFO, "writting email %v", emailData.ID)

	err = s.physical.Write(&emailData)
	if err != nil {
		return err
	}

	s.cache.Store(emailData)

	return nil
}

// Get retrieves the email data for the specified UUID.
func (s *Storage) Get(id uuid.UUID) (*EmailData, bool) {
	return s.cache.Load(id)
}

// Get retrieves the email data for the specified UUID.
func (s *Storage) Delete(id uuid.UUID) error {
	log.Logf(log.INFO, "deleting email %v", id)

	err := s.physical.Delete(id)
	if err != nil {
		return err
	}

	s.cache.Delete(id)
	return nil
}

// GetAll retrieves all UUIDs of email data in the storage sorted according to sorting options.
func (s *Storage) GetAll(sortOptions SortOption) []uuid.UUID {
	return s.cache.Find(email.MatchOption{Type: email.AllMatch}, sortOptions, "")
}

// Find searches for UUIDs of emails in the storage based on the provided matching and sorting options.
func (s *Storage) Find(matchOptions email.MatchOption, sortOptions SortOption, value string) []uuid.UUID {
	return s.cache.Find(matchOptions, sortOptions, value)
}
