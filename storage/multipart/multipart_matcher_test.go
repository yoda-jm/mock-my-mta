package multipart

import (
	"mock-my-mta/storage/matcher"
	"net/mail"
	"strings"
	"testing"
)

const simpleEmailMatcher = `From: sender@example.com
To: to1@example.com, to2@example.com
Cc: cc1@example.com, cc2@example.com
Subject: This is the subject of the email
Date: Sat, 03 Nov 1979 00:00:00 +0000
Message-ID: <1234567890@mock-my-mta>
Content-Type: text/plain

Hello, this is a simple email.
`

func TestMatch(t *testing.T) {
	// read the email
	email, err := mail.ReadMessage(strings.NewReader(simpleEmailMatcher))
	if err != nil {
		t.Fatal(err)
	}
	// parse the email
	multipart, err := New(email)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		input         interface{}
		expectedValue bool
	}{
		// Mailbox matchers
		{"match-mailbox-to", mustParseQuery(t, "mailbox:to1@example.com"), true},
		{"match-mailbox-to-case", mustParseQuery(t, "mailbox:TO1@EXAMPLE.COM"), true},
		{"match-mailbox-cc", mustParseQuery(t, "mailbox:cc1@example.com"), true},
		{"not-match-mailbox", mustParseQuery(t, "mailbox:sender@example.com"), false},
		{"not-match-mailbox", mustParseQuery(t, "mailbox:unknown@example.com"), false},
		// Attachment matchers
		{"match-attachment", mustParseQuery(t, "has:attachment"), false},
		// Plain text matchers
		// FIXME: This is not working as expected
		//{"match-plain-text", mustParseQuery(t, "important"), true},
		//{"match-plain-text-quote", mustParseQuery(t, "\"important\""), true},
		//{"not-match-plain-text", mustParseQuery(t, "unknown"), false},
		// Before matchers
		{"match-before", mustParseQuery(t, "before:1979-11-04"), true},
		{"not-match-before", mustParseQuery(t, "before:1979-11-02"), false},
		// After matchers
		{"match-after", mustParseQuery(t, "after:1979-11-02"), true},
		{"not-match-after", mustParseQuery(t, "after:1979-11-04"), false},
		// From matchers
		{"match-from", mustParseQuery(t, "from:sender@example.com"), true},
		{"not-match-from", mustParseQuery(t, "from:unknown@example.com"), false},
		// Newer matchers
		{"match-newer", mustParseQuery(t, "newer_than:438000h"), true},
		{"not-match-newer", mustParseQuery(t, "newer_than:1h"), false},
		// Older matchers
		{"match-older", mustParseQuery(t, "older_than:1h"), true},
		{"not-match-older", mustParseQuery(t, "older_than:438000h"), false},
		// Subject matchers
		{"match-subject", mustParseQuery(t, "subject:subject"), true},
		{"match-subject-case", mustParseQuery(t, "subject:SUBJECT"), true},
		{"match-subject-quote", mustParseQuery(t, "subject:\"of the email\""), true},
		{"match-subject-quote-case", mustParseQuery(t, "subject:\"OF THE EMAIL\""), true},
		{"not-match-subject", mustParseQuery(t, "subject:unknown"), false},
		{"not-match-subject-quote", mustParseQuery(t, "subject:\"oof the email\""), false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Logf("Running test %s", test.name)
			actual := multipart.match(test.input)
			if actual != test.expectedValue {
				t.Errorf("expected %v, got %v", test.expectedValue, actual)
			}
		})
	}
}

func TestMatchAll(t *testing.T) {
	// read the email
	email, err := mail.ReadMessage(strings.NewReader(simpleEmailMatcher))
	if err != nil {
		t.Fatal(err)
	}
	// parse the email
	multipart, err := New(email)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		input         []interface{}
		expectedValue bool
	}{
		{"match-all", []interface{}{
			mustParseQuery(t, "mailbox:to1@example.com"),
			mustParseQuery(t, "from: sender@example.com"),
		}, true},
		{"match-one-first", []interface{}{
			mustParseQuery(t, "mailbox:to1@example.com"),
			mustParseQuery(t, "from:unknown@example.com"),
		}, false},
		{"match-one-second", []interface{}{
			mustParseQuery(t, "mailbox:unknown@example.com"),
			mustParseQuery(t, "from:sender@example.com"),
		}, false},
		{"match-none", []interface{}{
			mustParseQuery(t, "mailbox:unknown@example.com"),
			mustParseQuery(t, "from:unknown@example.com"),
		}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Logf("Running test %s", test.name)
			actual := multipart.MatchAll(test.input)
			if actual != test.expectedValue {
				t.Errorf("expected %v, got %v", test.expectedValue, actual)
			}
		})
	}

}

func mustParseQuery(t *testing.T, query string) interface{} {
	matchers, err := matcher.ParseQuery(query)
	if err != nil {
		t.Fatalf("Error parsing query: %v", err)
	}
	if len(matchers) != 1 {
		t.Fatalf("Expected 1 matcher, got %v", len(matchers))
	}
	return matchers[0]
}
