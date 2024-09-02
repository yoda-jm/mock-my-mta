package matcher

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	testData := []struct {
		name          string
		query         string
		expectedType  string
		expectedValue interface{}
		expectedError error
	}{
		// OK cases
		{"has attachment", "has:attachment", "AttachmentMatch", nil, nil},
		{"mailbox", "mailbox:recipient@example.com", "MailboxMatch", "recipient@example.com", nil},
		{"before", "before:2020-02-01", "BeforeMatch", time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC), nil},
		{"after", "after:2020-03-01", "AfterMatch", time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC), nil},
		{"from", "from:sender@example.com", "FromMatch", "sender@example.com", nil},
		{"older_than", "older_than:2h", "OlderThanMatch", 2 * time.Hour, nil},
		{"newer_than", "newer_than:2h", "NewerThanMatch", 2 * time.Hour, nil},
		{"older_than_days", "older_than:2days", "OlderThanMatch", 2 * 24 * time.Hour, nil},
		{"newer_than_days", "newer_than:2days", "NewerThanMatch", 2 * 24 * time.Hour, nil},
		{"older_than_days", "older_than:2d", "OlderThanMatch", 2 * 24 * time.Hour, nil},
		{"newer_than_days", "newer_than:2d", "NewerThanMatch", 2 * 24 * time.Hour, nil},
		{"subject", "subject:important", "SubjectMatch", "important", nil},
		{"plain_text", "important", "PlainTextMatch", "important", nil},
		{"plain_text_quote", "\"important thing\"", "PlainTextMatch", "important thing", nil},
		{"empty query", "", "", nil, nil},
		{"empty quote", "\"\"", "", nil, nil},
		// Error cases
		{"has something", "has:something", "", nil, InvalidQueryError{}},
		{"before invalid date", "before:2020-02-30", "", nil, InvalidQueryError{}},
		{"after invalid date", "after:2020-02-30", "", nil, InvalidQueryError{}},
		{"older_than invalid duration", "older_than:2f30m", "", nil, InvalidQueryError{}},
		{"newer_than invalid duration", "newer_than:2f30m", "", nil, InvalidQueryError{}},
		{"unknown key", "unknown:some-value", "", nil, InvalidQueryError{}},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			matchers, err := ParseQuery(data.query)
			if err != nil && data.expectedError == nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && data.expectedError != nil {
				t.Errorf("Expected error: %v", data.expectedError)
			}
			if err != nil && data.expectedError != nil {
				// check that err and data.expectedError have the same type
				if fmt.Sprintf("%T", err) != fmt.Sprintf("%T", data.expectedError) {
					t.Errorf("Expected error type %T, got %T", data.expectedError, err)
				}
				return
			}
			if data.expectedValue == nil && len(matchers) == 0 {
				return
			}
			if len(matchers) != 1 {
				t.Errorf("Expected 1 matcher, got %v", len(matchers))
			}
			matcher := matchers[0]
			// check if the type is correct
			// check if the value is correct
			switch m := matcher.(type) {
			case AttachmentMatch:
				// nothing to check
				if data.expectedType != "AttachmentMatch" {
					t.Errorf("Expected AttachmentMatch, got %T", m)
				}
			case MailboxMatch:
				if data.expectedType != "MailboxMatch" {
					t.Errorf("Expected MailboxMatch, got %T", m)
				}
				if m.GetMailbox() != data.expectedValue {
					t.Errorf("Expected %v, got %v", data.expectedValue, m.GetMailbox())
				}
			case BeforeMatch:
				if data.expectedType != "BeforeMatch" {
					t.Errorf("Expected BeforeMatch, got %T", m)
				}
				if m.GetDate() != data.expectedValue {
					t.Errorf("Expected %v, got %v", data.expectedValue, m.GetDate())
				}
			case AfterMatch:
				if data.expectedType != "AfterMatch" {
					t.Errorf("Expected AfterMatch, got %T", m)
				}
				if m.GetDate() != data.expectedValue {
					t.Errorf("Expected %v, got %v", data.expectedValue, m.GetDate())
				}
			case FromMatch:
				if data.expectedType != "FromMatch" {
					t.Errorf("Expected FromMatch, got %T", m)
				}
				if m.GetFrom() != data.expectedValue {
					t.Errorf("Expected %v, got %v", data.expectedValue, m.GetFrom())
				}
			case OlderThanMatch:
				if data.expectedType != "OlderThanMatch" {
					t.Errorf("Expected OlderThanMatch, got %T", m)
				}
				if m.GetDuration() != data.expectedValue {
					t.Errorf("Expected %v, got %v", data.expectedValue, m.GetDuration())
				}
			case NewerThanMatch:
				if data.expectedType != "NewerThanMatch" {
					t.Errorf("Expected NewerThanMatch, got %T", m)
				}
				if m.GetDuration() != data.expectedValue {
					t.Errorf("Expected %v, got %v", data.expectedValue, m.GetDuration())
				}
			case SubjectMatch:
				if data.expectedType != "SubjectMatch" {
					t.Errorf("Expected SubjectMatch, got %T", m)
				}
				if m.GetSubject() != data.expectedValue {
					t.Errorf("Expected %v, got %v", data.expectedValue, m.GetSubject())
				}
			case PlainTextMatch:
				if data.expectedType != "PlainTextMatch" {
					t.Errorf("Expected PlainTextMatch, got %T", m)
				}
				if m.GetText() != data.expectedValue {
					t.Errorf("Expected %v, got %v", data.expectedValue, m.GetText())
				}
			default:
			}

		})
	}
}

func TestParseMixed(t *testing.T) {
	// Test for mailbox:inbox has:attachment
	query := "mailbox:recipient@example.com has:attachment sometext \"important and quoted\" before:2020-02-01 after:2020-03-01 from:sender@example.com older_than:2h subject:important"
	matchers, err := ParseQuery(query)
	if err != nil {
		t.Errorf("Error parsing query: %v", err)
	}
	expectedMatchers := []interface{}{
		newMailboxMatch("recipient@example.com"),
		newAttachmentMatch(),
		newPlainTextMatch("sometext"),
		newPlainTextMatch("important and quoted"),
		newBeforeMatch(time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)),
		newAfterMatch(time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC)),
		newFromMatch("sender@example.com"),
		newOlderThanMatch(2 * time.Hour),
		newSubjectMatch("important"),
	}
	// check that all the expected matchers are present and no extra matchers are present
	for _, expectedMatcher := range expectedMatchers {
		found := false
		for _, matcher := range matchers {
			if matcher == expectedMatcher {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected matcher %v not found", expectedMatcher)
		}
	}
	for _, matcher := range matchers {
		found := false
		for _, expectedMatcher := range expectedMatchers {
			if matcher == expectedMatcher {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Unexpected matcher %v found", matcher)
		}
	}
}

func TestParseQuotedSubject(t *testing.T) {
	// Test for subject:"important and quoted"
	query := "subject:\"important and quoted\""
	matchers, err := ParseQuery(query)
	if err != nil {
		t.Errorf("Error parsing query: %v", err)
	}
	if len(matchers) != 1 {
		t.Errorf("Expected 1 matcher, got %v", len(matchers))
	}
	subjectMatch, ok := matchers[0].(SubjectMatch)
	if !ok {
		t.Errorf("Expected SubjectMatch, got %T", matchers[0])
	}
	if subjectMatch.GetSubject() != "important and quoted" {
		t.Errorf("Expected subject important and quoted, got %v", subjectMatch.GetSubject())
	}
}

func TestInvalidQueryError(t *testing.T) {
	query := "mailbox:recipient@example.com has:attachment sometext \"important and quoted\" before:2020-02-01 after:2020-03-01 from:sender@example.com older_than:2h subject:important"
	errorString := "some error message"
	// check that the error message contains both the query and the error string
	err := newInvalidQueryError(query, errorString)
	errorMessage := err.Error()
	// check that the error message contains the query (quoted)
	if !strings.Contains(errorMessage, fmt.Sprintf("%q", query)) {
		t.Errorf("Expected error message to contain query, got %v", err.Error())
	}
	// check that the error message contains the error string
	if !strings.Contains(errorMessage, errorString) {
		t.Errorf("Expected error message to contain error string, got %v", err.Error())
	}
}

func TestParseCustomDuration(t *testing.T) {
	testCases := []struct {
		name     string
		duration string
		expected time.Duration
		err	     error
	}{
		// cutom duration cases
		{"1d", "1d", 24 * time.Hour, nil},
		{"3d", "3d", 3 * 24 * time.Hour, nil},
		{"1day", "1day", 24 * time.Hour, nil},
		{"3days", "3days", 3 * 24 * time.Hour, nil},
		{"1week", "1week", 7 * 24 * time.Hour, nil},
		{"1w", "1w", 7 * 24 * time.Hour, nil},
		{"3w", "3w", 3 * 7 * 24 * time.Hour, nil},
		{"3weeks", "3weeks", 3 * 7 * 24 * time.Hour, nil},
		{"1month", "1month", 30 * 24 * time.Hour, nil},
		{"3months", "3months", 3 * 30 * 24 * time.Hour, nil},
		{"1y", "1y", 365 * 24 * time.Hour, nil},
		{"3y", "3y", 3 * 365 * 24 * time.Hour, nil},
		{"1year", "1year", 365 * 24 * time.Hour, nil},
		{"3years", "3years", 3 * 365 * 24 * time.Hour, nil},
		{"invalid", "invalid", 0, fmt.Errorf("time: invalid duration \"%v\"", "invalid")},
		// standard duration cases
		{"1h", "1h", 1 * time.Hour, nil},
		{"1m", "1m", 1 * time.Minute, nil},
		{"1s", "1s", 1 * time.Second, nil},
		{"1ms", "1ms", 1 * time.Millisecond, nil},
		{"1us", "1us", 1 * time.Microsecond, nil},
		{"1ns", "1ns", 1 * time.Nanosecond, nil},
		// combined standard duration cases
		{"1h1m", "1h1m", 1 * time.Hour + 1 * time.Minute, nil},
		{"1h1m1s", "1h1m1s", 1 * time.Hour + 1 * time.Minute + 1 * time.Second, nil},
		{"1h1m1s1ms", "1h1m1s1ms", 1 * time.Hour + 1 * time.Minute + 1 * time.Second + 1 * time.Millisecond, nil},
		{"1h1m1s1ms1us", "1h1m1s1ms1us", 1 * time.Hour + 1 * time.Minute + 1 * time.Second + 1 * time.Millisecond + 1 * time.Microsecond, nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			duration, err := parseCustomDuration(tc.duration)
			if err != nil && tc.err == nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && tc.err != nil {
				t.Errorf("Expected error: %v", tc.err)
			}
			if err != nil && tc.err != nil {
				if err.Error() != tc.err.Error() {
					t.Errorf("Expected error %v, got %v", tc.err, err)
				}
				return
			}
			if duration != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, duration)
			}
		})
	}
}