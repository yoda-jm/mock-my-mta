package storage

import (
	"sort"

	"github.com/google/uuid"

	"mock-my-mta/email"
)

// defaultSearch is performing a search by only using the physical storage without using the Find method of the storage.
func defaultSearch(physicalStorage PhysicalLayer, matchOptions email.MatchOption, sortOptions SortOption, value string) ([]uuid.UUID, error) {
	ids, err := physicalStorage.List()
	if err != nil {
		return nil, err
	}

	pairs := make([]pair, 0, len(ids))
	for _, id := range ids {
		emailData, err := physicalStorage.Read(id)
		if err != nil {
			return nil, err
		}
		if emailData.Email.Match(matchOptions, value) {
			pairs = append(pairs, pair{uuid: id, fieldValue: getSortField(sortOptions.Field, *emailData)})
		}
	}

	sortPairs(pairs, sortOptions.Direction)
	return pairsToIdSlice(pairs), nil
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
