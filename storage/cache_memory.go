package storage

import (
	"sort"
	"sync"

	"github.com/google/uuid"

	"mock-my-mta/email"
)

type MemoryCache struct {
	mutex sync.RWMutex // protects data
	data  map[uuid.UUID]EmailData
}

func newMemoryCache() Cache {
	return &MemoryCache{
		data: make(map[uuid.UUID]EmailData),
	}
}

func (mc *MemoryCache) Fill(physical Physical) error {
	uuids, err := physical.List()
	if err != nil {
		return err
	}

	for _, uuid := range uuids {
		emailData, err := physical.Read(uuid)
		if err != nil {
			return err
		}
		mc.Store(*emailData)
	}
	return nil
}

func (mc *MemoryCache) Store(emailData EmailData) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.data[emailData.ID] = emailData
}

func (mc *MemoryCache) Load(id uuid.UUID) (*EmailData, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	emailData, ok := mc.data[id]
	return &emailData, ok
}

func (mc *MemoryCache) Delete(id uuid.UUID) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	delete(mc.data, id)
}

// WalkFunc defines the function signature for the walk function.
type WalkFunc func(id uuid.UUID)

// walk walks through all UUIDs in the storage and calls the walkFunc for each UUID.
func (mc *MemoryCache) walk(walkFunc WalkFunc) {
	for id := range mc.data {
		walkFunc(id)
	}
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

// Find searches for UUIDs of emails in the storage based on the provided matching and sorting options.
func (mc *MemoryCache) Find(matchOptions email.MatchOption, sortOptions SortOption, value string) []uuid.UUID {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	pairs := make([]pair, 0, len(mc.data))
	mc.walk(func(id uuid.UUID) {
		if emailData, ok := mc.data[id]; ok && emailData.Email.Match(matchOptions, value) {
			pairs = append(pairs, pair{uuid: id, fieldValue: getSortField(sortOptions.Field, mc.data[id])})
		}
	})

	sortPairs(pairs, sortOptions.Direction)
	return pairsToIdSlice(pairs)
}
