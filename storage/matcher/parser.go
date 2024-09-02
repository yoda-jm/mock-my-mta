package matcher

import (
	"fmt"
	"mock-my-mta/log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// invalid query error
type InvalidQueryError struct {
	query string
	err   string
}

func newInvalidQueryError(query, err string) InvalidQueryError {
	return InvalidQueryError{query: query, err: err}
}

func (e InvalidQueryError) Error() string {
	return fmt.Sprintf("invalid query %q: %v", e.query, e.err)
}

// extract matchers from the query
func ParseQuery(query string) ([]interface{}, error) {
	const LAYOUT_DATE = "2006-01-02"
	keyValuePairs, plainTexts := tokenizeQuery(query)

	matchers := make([]interface{}, 0)
	for _, keyValue := range keyValuePairs {
		for key, value := range keyValue {
			switch key {
			case "mailbox":
				// Search for emails in the specified mailbox
				log.Logf(log.DEBUG, "searching for mailbox %v", value)
				matchers = append(matchers, newMailboxMatch(value))
			case "has":
				switch value {
				case "attachment":
					// Search for emails that have the specified attribute
					log.Logf(log.DEBUG, "searching for emails with attachments")
					matchers = append(matchers, newAttachmentMatch())
				default:
					return nil, newInvalidQueryError(query, fmt.Sprintf("unknown search attribute for 'has': %v", value))
				}
			case "before":
				// search for emails with date before
				valueDate, err := time.Parse(LAYOUT_DATE, value)
				if err != nil {
					return nil, newInvalidQueryError(query, fmt.Sprintf("invalid date format: %v", value))
				}
				log.Logf(log.DEBUG, "searching for emails before %v", value)
				matchers = append(matchers, newBeforeMatch(valueDate))
			case "after":
				// search for emails with date after
				valueDate, err := time.Parse(LAYOUT_DATE, value)
				if err != nil {
					return nil, newInvalidQueryError(query, fmt.Sprintf("invalid date format: %v", value))
				}
				log.Logf(log.DEBUG, "searching for emails after %v", value)
				matchers = append(matchers, newAfterMatch(valueDate))
			case "from":
				// search for emails from the specified address
				log.Logf(log.DEBUG, "searching for emails from %v", value)
				matchers = append(matchers, newFromMatch(value))
			case "older_than":
				// search for emails older than the specified duration
				duration, err := parseCustomDuration(value)
				if err != nil {
					return nil, newInvalidQueryError(query, fmt.Sprintf("invalid duration format: %v", value))
				}
				log.Logf(log.DEBUG, "searching for emails older than %v", duration)
				matchers = append(matchers, newOlderThanMatch(duration))
			case "newer_than":
				// search for emails newer than the specified duration
				duration, err := parseCustomDuration(value)
				if err != nil {
					return nil, newInvalidQueryError(query, fmt.Sprintf("invalid duration format: %v", value))
				}
				log.Logf(log.DEBUG, "searching for emails newer than %v", duration)
				matchers = append(matchers, newNewerThanMatch(duration))
			case "subject":
				// search for emails with the specified word in the subject
				log.Logf(log.DEBUG, "searching for emails with subject %v", value)
				matchers = append(matchers, newSubjectMatch(value))
			default:
				return nil, newInvalidQueryError(query, fmt.Sprintf("unknown search key: %v", key))
			}
		}
	}

	for _, plainText := range plainTexts {
		if plainText == "" {
			continue
		}
		// Search for emails that contain the plain text
		log.Logf(log.DEBUG, "searching for plain text %q", plainText)
		matchers = append(matchers, newPlainTextMatch(plainText))
	}
	return matchers, nil
}

func parseCustomDuration(value string) (time.Duration, error) {
	// regexps map (key: regexp, value: hour multiplier)
	regexps := map[*regexp.Regexp]int{
		regexp.MustCompile(`^(\d+)\s*(d|day|days)$`): 24,
		regexp.MustCompile(`^(\d+)\s*(w|week|weeks)$`): 24 * 7,
		regexp.MustCompile(`^(\d+)\s*(month|months)$`): 24 * 30,
		regexp.MustCompile(`^(\d+)\s*(y|year|years)$`): 24 * 365,
	}
	for re, multiplier := range regexps {
		if re.MatchString(value) {
			matches := re.FindStringSubmatch(value)
			days, err := strconv.ParseInt(matches[1], 10, 32)
			if err != nil {
				return 0, err
			}
			return time.Duration(days) * time.Duration(multiplier) * time.Hour, nil
		}
	}
	// if it doesn't match any of the above, parse it as a time.Duration
	return time.ParseDuration(value)
}

// tokenizeQuery parses the input string into a slice of key-value pairs and plain text elements.
func tokenizeQuery(query string) ([]map[string]string, []string) {
	var keyValuePairs []map[string]string
	var plainTexts []string

	// Regex pattern to extract key:value pairs and quoted/non-quoted text
	pattern := `(\w+:\s*"[^"]+"|\w+:\s*\S+|"[^"]+"|\S+)`

	re := regexp.MustCompile(pattern)
	matches := re.FindAllString(query, -1)
	for _, match := range matches {
		// Split only at the first occurrence of ':'
		splitIndex := strings.Index(match, ":")
		if splitIndex != -1 {
			key := match[:splitIndex]
			value := strings.TrimSpace(match[splitIndex+1:])
			// Remove quotes if they exist
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = strings.Trim(value, "\"")
			}
			keyValuePair := make(map[string]string)
			keyValuePair[key] = value
			keyValuePairs = append(keyValuePairs, keyValuePair)
		} else if strings.HasPrefix(match, "\"") && strings.HasSuffix(match, "\"") {
			// Handle standalone quoted strings
			plainTexts = append(plainTexts, strings.Trim(match, "\""))
		} else {
			// Generic word handling
			plainTexts = append(plainTexts, match)
		}
	}

	return keyValuePairs, plainTexts
}
