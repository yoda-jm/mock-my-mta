package storage

import (
	"strings"

	"mock-my-mta/email"
)

// Match checks if the email matches the provided criteria.
func matchEmailData(e *email.Email, option MatchOption, value string) bool {
	if option.HasAttachment && len(e.GetAttachments()) == 0 {
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
			return exactMatchRecipient(e.GetRecipients(), matchValue)
		}
		return fieldValue == matchValue
	case ContainsMatch:
		return strings.Contains(fieldValue, matchValue)
	default:
		return false
	}
}

// getMatchField retrieves the value of the specified field in the email.
func getMatchField(field MatchFieldEnum, e *email.Email) string {
	switch field {
	case MatchSubjectField:
		return e.GetSubject()
	case MatchBodyField:
		body, _ := e.GetBody(email.EmailVersionTxt)
		return body
	case MatchSenderField:
		return e.GetSender()
	case MatchRecipientField:
		return strings.Join(e.GetRecipients(), ",")
	default:
		return ""
	}
}
