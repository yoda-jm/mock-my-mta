package storage

import (
	"github.com/google/uuid"

	"mock-my-mta/email"
)

type Cache interface {
	Fill(Physical) error
	Store(EmailData)
	Load(uuid.UUID) (*EmailData, bool)
	Delete(uuid.UUID)
	Find(email.MatchOption, SortOption, string) []uuid.UUID
}
