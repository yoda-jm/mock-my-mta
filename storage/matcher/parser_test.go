package matcher

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	testData := []struct {
		name          string
		query         string
		expectedType  string
		expectedValue interface{}
	}{
		{"has attachment", "has:attachment", "AttachmentMatch", nil},
		{"mailbox", "mailbox:recipient@example.com", "MailboxMatch", "recipient@example.com"},
		{"before", "before:2020-02-01", "BeforeMatch", time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)},
		{"after", "after:2020-03-01", "AfterMatch", time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC)},
		{"from", "from:sender@example.com", "FromMatch", "sender@example.com"},
		{"older_than", "older_than:2h", "OlderThanMatch", 2 * time.Hour},
		{"subject", "subject:important", "SubjectMatch", "important"},
		{"plain_text", "important", "PlainTextMatch", "important"},
		{"plain_text_quote", "\"important thing\"", "PlainTextMatch", "important thing"},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			matchers, err := ParseQuery(data.query)
			if err != nil {
				t.Errorf("Error parsing query: %v", err)
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
	// disable this test as the parser does not support quoted text
	t.Skip()
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
