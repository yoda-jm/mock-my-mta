package storage

import (
	"fmt"
)

// SortType represents the type sorting
type SortType int

// Enum values for SortType.
const (
	Ascending SortType = iota
	Descending
)

func ParseSortType(str string) (SortType, error) {
	switch str {
	case "asc":
		return Ascending, nil
	case "desc":
		return Descending, nil
	default:
		return Ascending, fmt.Errorf("cannot sort type %q", str)
	}
}

// SortFieldEnum represents the available fields for sorting.
type SortFieldEnum int

// Enum values for SortFieldEnum.
const (
	SortSubjectField SortFieldEnum = iota
	SortSenderField
	SortDateField
)

func ParseSortFieldEnum(str string) (SortFieldEnum, error) {
	switch str {
	case "subject":
		return SortSubjectField, nil
	case "sender":
		return SortSenderField, nil
	case "date":
		return SortDateField, nil
	default:
		return SortDateField, fmt.Errorf("cannot sort field type %q", str)
	}
}

type SortOption struct {
	Field     SortFieldEnum
	Direction SortType
}

// getSortField retrieves the value of the specified field in the email.
func getSortField(field SortFieldEnum, e EmailData) string {
	switch field {
	case SortSubjectField:
		return e.Email.GetSubject()
	case SortSenderField:
		return e.Email.GetSender()
	case SortDateField:
		return e.ReceivedTime.String()
	default:
		return ""
	}
}
