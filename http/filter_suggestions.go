package http

import (
	"encoding/json"
	"net/http"
	"strings"
)

// getFilterSuggestions provides search filter command suggestions.
// - If 'term' query parameter is provided and not empty, it filters suggestions by command and returns an array of suggestion strings.
// - If 'term' is not provided or is empty, it returns the full list of FilterSyntaxEntry objects.
func getFilterSuggestions(w http.ResponseWriter, r *http.Request) {
	term := strings.ToLower(r.URL.Query().Get("term"))

	w.Header().Set("Content-Type", "application/json")

	if len(filterSyntaxEntries) == 0 {
		// If there's no filter syntax configured, return empty list regardless of term
		w.Write([]byte("[]")) // Empty JSON array
		return
	}

	if term != "" {
		// Filter by term and return only suggestion strings
		var suggestions []string
		for _, entry := range filterSyntaxEntries {
			if strings.HasPrefix(strings.ToLower(entry.Command), term) {
				suggestions = append(suggestions, entry.Suggestion)
			}
		}
		if suggestions == nil { // Ensure empty array, not null
			suggestions = []string{}
		}
		js, err := json.Marshal(suggestions)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(js)
	} else {
		// No term, return full filter syntax entries
		js, err := json.Marshal(filterSyntaxEntries)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(js)
	}
}

// FilterSyntaxEntry defines the structure for each filter command syntax help.
type FilterSyntaxEntry struct {
	Command     string `json:"command"`
	Suggestion  string `json:"suggestion"`
	Description string `json:"description"`
}

// filterSyntaxEntries contains the list of filter command syntax entries.
var filterSyntaxEntries = []FilterSyntaxEntry{
	{
		Command:     "mailbox",
		Suggestion:  "mailbox:<name>",
		Description: "Search for emails in a specific mailbox.",
	},
	{
		Command:     "has",
		Suggestion:  "has:attachment",
		Description: "Search for emails that have attachments.",
	},
	{
		Command:     "before",
		Suggestion:  "before:<YYYY-MM-DD>",
		Description: "Search for emails received before a specific date.",
	},
	{
		Command:     "after",
		Suggestion:  "after:<YYYY-MM-DD>",
		Description: "Search for emails received after a specific date.",
	},
	{
		Command:     "from",
		Suggestion:  "from:<email_address>",
		Description: "Search for emails from a specific sender.",
	},
	{
		Command:     "subject",
		Suggestion:  "subject:<text>",
		Description: "Search for emails with specific text in the subject.",
	},
	{
		Command:     "older_than",
		Suggestion:  "older_than:<duration>",
		Description: "Search for emails older than a duration (e.g., 7d, 2w, 1m, 1y). Supports d, w, month, y and Go's time.ParseDuration format (e.g. 24h, 720h).",
	},
	{
		Command:     "newer_than",
		Suggestion:  "newer_than:<duration>",
		Description: "Search for emails newer than a duration (e.g., 7d, 2w, 1m, 1y). Supports d, w, month, y and Go's time.ParseDuration format (e.g. 24h, 720h).",
	},
}
