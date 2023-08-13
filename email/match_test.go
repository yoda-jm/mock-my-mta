package email

import (
	"testing"
)

const email = `From: sender@example.com
To: recipient1@example.com, recipient2@example.com
Subject: Subject

Body`

func TestEmailMatch(t *testing.T) {
	email, err := Parse([]byte(email))
	if err != nil {
		t.Fatalf("cannot create email: %v", err)
	}

	tests := []struct {
		name     string
		option   MatchOption
		value    string
		expected bool
	}{
		{
			name:     "Exact match for subject",
			option:   MatchOption{Field: MatchSubjectField, Type: ExactMatch, CaseSensitive: true},
			value:    "Subject",
			expected: true,
		},
		{
			name:     "Exact match for body",
			option:   MatchOption{Field: MatchBodyField, Type: ExactMatch, CaseSensitive: true},
			value:    "Body",
			expected: true,
		},
		{
			name:     "Exact match for sender",
			option:   MatchOption{Field: MatchSenderField, Type: ExactMatch, CaseSensitive: true},
			value:    "sender@example.com",
			expected: true,
		},
		{
			name:     "Exact match for recipient",
			option:   MatchOption{Field: MatchRecipientField, Type: ExactMatch, CaseSensitive: true},
			value:    "recipient1@example.com",
			expected: true,
		},
		{
			name:     "Contains match for body",
			option:   MatchOption{Field: MatchBodyField, Type: ContainsMatch, CaseSensitive: true},
			value:    "dy",
			expected: true,
		},
		{
			name:     "Exact match for subject (case-insensitive)",
			option:   MatchOption{Field: MatchSubjectField, Type: ExactMatch, CaseSensitive: false},
			value:    "subject",
			expected: true,
		},
		{
			name:     "Exact match for recipient (case-insensitive)",
			option:   MatchOption{Field: MatchRecipientField, Type: ExactMatch, CaseSensitive: false},
			value:    "RECIPIENT1@example.com",
			expected: true,
		},
		{
			name:     "Exact match for recipient with incorrect case",
			option:   MatchOption{Field: MatchRecipientField, Type: ExactMatch, CaseSensitive: true},
			value:    "RECIPIENT1@example.com",
			expected: false,
		},
		{
			name:     "No match and no attachment",
			option:   MatchOption{Field: MatchNoField, Type: ExactMatch, CaseSensitive: true, HasAttachment: true},
			value:    "",
			expected: false,
		},
		{
			name:     "Exact match for invalid field",
			option:   MatchOption{Field: MatchFieldEnum(999), Type: ExactMatch, CaseSensitive: true},
			value:    "Value",
			expected: false,
		},
		{
			name:     "Match all",
			option:   MatchOption{Type: AllMatch},
			value:    "",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := email.Match(test.option, test.value)
			if result != test.expected {
				t.Errorf("Expected match result %v, but got %v (test=%q)", test.expected, result, test.name)
			}
		})
	}
}
