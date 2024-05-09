package storage

import "testing"

func TestParseEmailVersionType(t *testing.T) {
	tests := []struct {
		input    string
		expected EmailVersionType
	}{
		{"raw", EmailVersionRaw},
		{"plain-text", EmailVersionPlainText},
		{"html", EmailVersionHtml},
		{"watch-html", EmailVersionWatchHtml},
	}

	for _, data := range tests {
		t.Run(data.input, func(t *testing.T) {
			result, err := ParseEmailVersionType(data.input)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != data.expected {
				t.Errorf("Expected %v, got %v", data.expected, result)
			}
		})
	}

	// Test invalid input
	_, err := ParseEmailVersionType("invalid")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}
