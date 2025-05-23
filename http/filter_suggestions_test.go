package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestHandleFilterSuggestions(t *testing.T) {
	// Test Case 1: Get all suggestions (for help popup - no term)
	t.Run("GetAllSuggestions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/filters/suggestions", nil)
		rr := httptest.NewRecorder()
		getFilterSuggestions(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedResponse, _ := json.Marshal(filterSyntaxEntries)
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(string(expectedResponse)) {
			t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(expectedResponse))
		}

		// More detailed check for content
		var actualResponse []FilterSyntaxEntry
		if err := json.Unmarshal(rr.Body.Bytes(), &actualResponse); err != nil {
			t.Fatalf("could not unmarshal response body: %v", err)
		}
		if !reflect.DeepEqual(actualResponse, filterSyntaxEntries) {
			t.Errorf("handler returned unexpected body content: got %+v want %+v", actualResponse, filterSyntaxEntries)
		}
	})

	// Test Case 2: Get filtered suggestions (for live typing)
	t.Run("GetFilteredSuggestionsMatch", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/filters/suggestions?term=h", nil)
		rr := httptest.NewRecorder()
		getFilterSuggestions(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedSuggestions := []string{"has:attachment"}
		expectedResponse, _ := json.Marshal(expectedSuggestions)
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(string(expectedResponse)) {
			t.Errorf("handler returned unexpected body for term 'h': got %v want %v", rr.Body.String(), string(expectedResponse))
		}

		var actualSuggestions []string
		if err := json.Unmarshal(rr.Body.Bytes(), &actualSuggestions); err != nil {
			t.Fatalf("could not unmarshal response body: %v", err)
		}
		if !reflect.DeepEqual(actualSuggestions, expectedSuggestions) {
			t.Errorf("handler returned unexpected suggestions: got %v want %v", actualSuggestions, expectedSuggestions)
		}
	})

	// Test Case 3: Get filtered suggestions (no match)
	t.Run("GetFilteredSuggestionsNoMatch", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/filters/suggestions?term=xyz", nil)
		rr := httptest.NewRecorder()
		getFilterSuggestions(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedSuggestions := []string{} // Empty slice
		expectedResponse, _ := json.Marshal(expectedSuggestions)
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(string(expectedResponse)) {
			t.Errorf("handler returned unexpected body for term 'xyz': got %v want %v", rr.Body.String(), string(expectedResponse))
		}

		var actualSuggestions []string
		if err := json.Unmarshal(rr.Body.Bytes(), &actualSuggestions); err != nil {
			t.Fatalf("could not unmarshal response body: %v", err)
		}
		if len(actualSuggestions) != 0 {
			t.Errorf("expected empty suggestions array, got %v", actualSuggestions)
		}
	})

	// Test Case 4: Get filtered suggestions (empty term - should return all for help)
	// This is similar to Test Case 1 based on current logic where empty term returns full FilterSyntaxEntry list
	t.Run("GetSuggestionsEmptyTerm", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/filters/suggestions?term=", nil)
		rr := httptest.NewRecorder()
		getFilterSuggestions(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code for empty term: got %v want %v", status, http.StatusOK)
		}

		expectedResponse, _ := json.Marshal(filterSyntaxEntries)
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(string(expectedResponse)) {
			t.Errorf("handler returned unexpected body for empty term: got %v want %v", rr.Body.String(), string(expectedResponse))
		}

		var actualResponse []FilterSyntaxEntry
		if err := json.Unmarshal(rr.Body.Bytes(), &actualResponse); err != nil {
			t.Fatalf("could not unmarshal response body for empty term: %v", err)
		}
		if !reflect.DeepEqual(actualResponse, filterSyntaxEntries) {
			t.Errorf("handler returned unexpected body content for empty term: got %+v want %+v", actualResponse, filterSyntaxEntries)
		}
	})
}
