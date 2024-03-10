package storage

import (
	"sync"

	"github.com/google/uuid"

	"mock-my-mta/email"
	"mock-my-mta/log"
)

// memoryPhysicalStorage is a physical storage providing
// to search and sort emails in memory.
type memoryPhysicalStorage struct {
	mutex sync.RWMutex // protects data
	data  map[uuid.UUID]EmailData
}

// check that the memoryPhysicalStorage implements the PhysicalLayer interface
var _ PhysicalLayer = &memoryPhysicalStorage{}

func newMemoryStorage() (*memoryPhysicalStorage, error) {
	return &memoryPhysicalStorage{
		data: make(map[uuid.UUID]EmailData),
	}, nil
}

// Delete implements PhysicalLayer.
func (mf *memoryPhysicalStorage) Delete(id uuid.UUID) error {
	mf.mutex.Lock()
	defer mf.mutex.Unlock()

	delete(mf.data, id)
	return nil
}

// Find implements PhysicalLayer.
func (mf *memoryPhysicalStorage) Find(matchOptions email.MatchOption, sortOptions SortOption, value string) ([]uuid.UUID, error) {
	mf.mutex.RLock()
	defer mf.mutex.RUnlock()

	// custom implementation of the find method, so that we don't need
	// to construct the list of all UUIDs in the storage.
	pairs := make([]pair, 0, len(mf.data))
	mf.walk(func(id uuid.UUID) {
		if emailData, ok := mf.data[id]; ok && emailData.Email.Match(matchOptions, value) {
			pairs = append(pairs, pair{uuid: id, fieldValue: getSortField(sortOptions.Field, mf.data[id])})
		}
	})

	sortPairs(pairs, sortOptions.Direction)
	return pairsToIdSlice(pairs), nil
}

// List implements PhysicalLayer.
func (mf *memoryPhysicalStorage) List() ([]uuid.UUID, error) {
	mf.mutex.RLock()
	defer mf.mutex.RUnlock()

	ids := make([]uuid.UUID, 0, len(mf.data))
	mf.walk(func(id uuid.UUID) {
		ids = append(ids, id)
	})
	return ids, nil
}

// Load implements PhysicalLayer.
func (mf *memoryPhysicalStorage) Populate(underlying PhysicalLayer, parameters map[string]string) error {
	log.Logf(log.INFO, "populating memory finder layer")
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
			mf.Write(emailData)
		}
	}
	return nil
}

// Read implements PhysicalLayer.
func (mf *memoryPhysicalStorage) Read(id uuid.UUID) (*EmailData, error) {
	mf.mutex.RLock()
	defer mf.mutex.RUnlock()

	emailData, found := mf.data[id]
	if !found {
		return nil, ErrNotFound{id: id}
	}
	return &emailData, nil
}

// Write implements PhysicalLayer.
func (mf *memoryPhysicalStorage) Write(emailData *EmailData) error {
	mf.mutex.Lock()
	defer mf.mutex.Unlock()

	mf.data[emailData.ID] = *emailData
	return nil
}

// WalkFunc defines the function signature for the walk function.
type WalkFunc func(id uuid.UUID)

// walk walks through all UUIDs in the storage and calls the walkFunc for each UUID.
func (mf *memoryPhysicalStorage) walk(walkFunc WalkFunc) {
	for id := range mf.data {
		walkFunc(id)
	}
}
