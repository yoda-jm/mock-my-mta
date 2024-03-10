package storage

import (
	"sort"
	"sync"

	"github.com/google/uuid"

	"mock-my-mta/email"
	"mock-my-mta/log"
)

// memoryFinderPhysicalStorage is a physical storage providing
// to search and sort emails in memory.
type memoryFinderPhysicalStorage struct {
	mutex sync.RWMutex // protects data
	data  map[uuid.UUID]EmailData
}

// check that the memoryFinderPhysicalStorage implements the PhysicalLayer interface
var _ PhysicalLayer = &memoryFinderPhysicalStorage{}

func newMemoryFinderStorage() (*memoryFinderPhysicalStorage, error) {
	return &memoryFinderPhysicalStorage{
		data: make(map[uuid.UUID]EmailData),
	}, nil
}

// Delete implements PhysicalLayer.
func (mf *memoryFinderPhysicalStorage) Delete(id uuid.UUID) error {
	mf.mutex.Lock()
	defer mf.mutex.Unlock()

	delete(mf.data, id)
	return nil
}

// Find implements PhysicalLayer.
func (mf *memoryFinderPhysicalStorage) Find(matchOptions email.MatchOption, sortOptions SortOption, value string) ([]uuid.UUID, error) {
	mf.mutex.RLock()
	defer mf.mutex.RUnlock()

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
func (*memoryFinderPhysicalStorage) List() ([]uuid.UUID, error) {
	return nil, unimplementedMethodInLayer{}
}

// Load implements PhysicalLayer.
func (mf *memoryFinderPhysicalStorage) Populate(underlying PhysicalLayer, parameters map[string]string) error {
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
func (*memoryFinderPhysicalStorage) Read(uuid.UUID) (*EmailData, error) {
	return nil, unimplementedMethodInLayer{}
}

// Write implements PhysicalLayer.
func (mf *memoryFinderPhysicalStorage) Write(emailData *EmailData) error {
	mf.mutex.Lock()
	defer mf.mutex.Unlock()

	mf.data[emailData.ID] = *emailData
	return nil
}

type pair struct {
	uuid       uuid.UUID
	fieldValue string
}

func sortPairs(pairs []pair, direction SortType) {
	sort.Slice(pairs, func(i, j int) bool {
		if direction == Ascending {
			return pairs[i].fieldValue < pairs[j].fieldValue
		}
		return pairs[i].fieldValue > pairs[j].fieldValue // For Descending
	})
}

func pairsToIdSlice(pairs []pair) []uuid.UUID {
	uuids := make([]uuid.UUID, 0, len(pairs))
	for _, pair := range pairs {
		uuids = append(uuids, pair.uuid)
	}
	return uuids
}

// WalkFunc defines the function signature for the walk function.
type WalkFunc func(id uuid.UUID)

// walk walks through all UUIDs in the storage and calls the walkFunc for each UUID.
func (mf *memoryFinderPhysicalStorage) walk(walkFunc WalkFunc) {
	for id := range mf.data {
		walkFunc(id)
	}
}
