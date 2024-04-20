package email

import (
	"net/textproto"
	"testing"
)

func TestReadMultipart(t *testing.T) {
	tests := []struct {
		name      string
		multipart string
		boundary  string
		isError   bool
		expected  []multipartPart
	}{
		{
			name: "Single part",
			multipart: `From:
				To:
				Subject:
			`,
			boundary: "boundary",
			isError:  false,
			expected: []multipartPart{
				{
					header: textproto.MIMEHeader{
						"From":    []string{""},
						"To":      []string{""},
						"Subject": []string{""},
					},
					body: []byte{},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parts, err := readMultipart([]byte(test.multipart), test.boundary)
			if test.isError && err == nil {
				t.Fatalf("expected error, got nil")
			} else if !test.isError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if len(parts) != len(test.expected) {
				t.Fatalf("expected %d parts, got %d", len(test.expected), len(parts))
			}

			for i, part := range parts {
				if len(part.header) != len(test.expected[i].header) {
					t.Fatalf("expected %d headers, got %d", len(test.expected[i].header), len(part.header))
				}
				for j, header := range part.header {
					if len(header) != len(test.expected[i].header[j]) {
						t.Errorf("expected %d header values, got %d", len(test.expected[i].header[j]), len(part.header[j]))
					}
				}
				if string(part.body) != string(test.expected[i].body) {
					t.Errorf("expected %q, got %q", test.expected[i], string(part.body))
				}
			}
		})
	}
}
