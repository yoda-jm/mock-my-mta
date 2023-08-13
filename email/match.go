package email

import (
	"strings"
)

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

// Match checks if the email matches the provided criteria.
func (e *Email) Match(option MatchOption, value string) bool {
	if option.HasAttachment && len(e.attachments) == 0 {
		// email has no attachnemnt
		return false
	}
	if option.Field == MatchNoField {
		return true
	}

	// we want to match on a field
	fieldValue := getMatchField(option.Field, e)
	if fieldValue == "" {
		return false
	}

	matchValue := value
	if !option.CaseSensitive {
		fieldValue = strings.ToLower(fieldValue)
		matchValue = strings.ToLower(value)
	}

	switch option.Type {
	case AllMatch:
		return true
	case ExactMatch:
		if option.Field == MatchRecipientField {
			return e.exactMatchRecipient(matchValue)
		}
		return fieldValue == matchValue
	case ContainsMatch:
		return strings.Contains(fieldValue, matchValue)
	default:
		return false
	}
}

// getMatchField retrieves the value of the specified field in the email.
func getMatchField(field MatchFieldEnum, e *Email) string {
	switch field {
	case MatchSubjectField:
		return e.GetSubject()
	case MatchBodyField:
		body, _ := e.GetBody(EmailVersionTxt)
		return body
	case MatchSenderField:
		return e.GetSender()
	case MatchRecipientField:
		return strings.Join(e.GetRecipients(), ",")
	default:
		return ""
	}
}

// exactMatchRecipient checks if the provided recipient matches exactly one recipient of the email.
func (e *Email) exactMatchRecipient(recipient string) bool {
	for _, r := range e.GetRecipients() {
		if r == recipient {
			return true
		}
	}
	return false
}
