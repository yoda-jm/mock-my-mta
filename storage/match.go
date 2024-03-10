package storage

// MatchType represents the type of match to perform.
type MatchType int

// Enum values for MatchType.
const (
	ExactMatch MatchType = iota
	ContainsMatch
	AllMatch
)

// MatchOption represents the matching options.
type MatchOption struct {
	Field         MatchFieldEnum
	Type          MatchType
	CaseSensitive bool
	HasAttachment bool
}

// MatchFieldEnum represents the available fields for matching.
type MatchFieldEnum int

// Enum values for MatchFieldEnum.
const (
	MatchNoField MatchFieldEnum = iota
	MatchSubjectField
	MatchBodyField
	MatchSenderField
	MatchRecipientField
)

// exactMatchRecipient checks if the provided recipient matches exactly one recipient of the email.
func exactMatchRecipient(recipients []string, recipient string) bool {
	for _, r := range recipients {
		if r == recipient {
			return true
		}
	}
	return false
}
