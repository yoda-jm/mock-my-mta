package matcher

import (
	"fmt"
	"mock-my-mta/log"
	"regexp"
	"strings"
	"time"
)

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
					return nil, fmt.Errorf("unknown search attribute for 'has': %v", value)
				}
			case "before":
				// search for emails with date before
				valueDate, err := time.Parse(LAYOUT_DATE, value)
				if err != nil {
					return nil, fmt.Errorf("invalid date format: %v", value)
				}
				log.Logf(log.DEBUG, "searching for emails before %v", value)
				matchers = append(matchers, newBeforeMatch(valueDate))
			case "after":
				// search for emails with date after
				valueDate, err := time.Parse(LAYOUT_DATE, value)
				if err != nil {
					return nil, fmt.Errorf("invalid date format: %v", value)
				}
				log.Logf(log.DEBUG, "searching for emails after %v", value)
				matchers = append(matchers, newAfterMatch(valueDate))
			case "from":
				// search for emails from the specified address
				log.Logf(log.DEBUG, "searching for emails from %v", value)
				matchers = append(matchers, newFromMatch(value))
			case "older_than":
				// search for emails older than the specified duration
				duration, err := time.ParseDuration(value)
				if err != nil {
					return nil, fmt.Errorf("invalid duration format: %v", value)
				}
				log.Logf(log.DEBUG, "searching for emails older than %v", duration)
				matchers = append(matchers, newOlderThanMatch(duration))
			case "newer_than":
				// search for emails newer than the specified duration
				duration, err := time.ParseDuration(value)
				if err != nil {
					return nil, fmt.Errorf("invalid duration format: %v", value)
				}
				log.Logf(log.DEBUG, "searching for emails newer than %v", duration)
				matchers = append(matchers, newNewerThanMatch(duration))
			case "subject":
				// search for emails with the specified word in the subject
				log.Logf(log.DEBUG, "searching for emails with subject %v", value)
				matchers = append(matchers, newSubjectMatch(value))
			default:
				return nil, fmt.Errorf("unknown search key: %v", key)
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

// tokenizeQuery parses the input string into a slice of key-value pairs and plain text elements.
func tokenizeQuery(query string) ([]map[string]string, []string) {
	var keyValuePairs []map[string]string
	var plainTexts []string

	// Regex pattern to extract key:value pairs and quoted/non-quoted text
	pattern := `(\w+:[^\s"]+|"[^"]*"|\S+)`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllString(query, -1)

	for _, match := range matches {
		if strings.Contains(match, ":") && !strings.HasPrefix(match, "\"") {
			// Split the first occurrence of ':' to separate key and value
			split := strings.SplitN(match, ":", 2)
			key := split[0]
			value := split[1]

			// Check if the value is quoted and remove quotes if needed
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = strings.Trim(value, "\"")
			}

			keyValuePair := make(map[string]string)
			keyValuePair[key] = value
			keyValuePairs = append(keyValuePairs, keyValuePair)
		} else if strings.HasPrefix(match, "\"") && strings.HasSuffix(match, "\"") {
			// Remove the quotes for plain text matches
			plainTexts = append(plainTexts, strings.Trim(match, "\""))
		} else {
			plainTexts = append(plainTexts, match)
		}
	}

	return keyValuePairs, plainTexts
}
