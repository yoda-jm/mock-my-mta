package main_test // Changed package to main_test to avoid conflicts when importing "mock-my-mta/cmd/server"

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	// Import for the configuration package
	appconfig "mock-my-mta/cmd/server/configtypes" 
	// Import for the http package where HandleFilterSuggestions now resides
	mtaHttp "mock-my-mta/http"       
)

var sampleSyntax = []appconfig.FilterSyntaxEntry{ // Use appconfig.FilterSyntaxEntry
	{Command: "has", Suggestion: "has:attachment", Description: "Has attachment"},
	{Command: "from", Suggestion: "from:<email>", Description: "From email"},
	{Command: "subject", Suggestion: "subject:<text>", Description: "Subject text"},
}

func TestHandleFilterSuggestions(t *testing.T) {
	// Test Case 1: Get all suggestions (for help popup - no term)
	t.Run("GetAllSuggestions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/filters/suggestions", nil)
		rr := httptest.NewRecorder()
		// Call the function from the mtaHttp package
		mtaHttp.HandleFilterSuggestions(rr, req, sampleSyntax) 

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		expectedResponse, _ := json.Marshal(sampleSyntax)
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(string(expectedResponse)) {
			t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(expectedResponse))
		}
		
		// More detailed check for content
		var actualResponse []appconfig.FilterSyntaxEntry // Use appconfig.FilterSyntaxEntry
		if err := json.Unmarshal(rr.Body.Bytes(), &actualResponse); err != nil {
			t.Fatalf("could not unmarshal response body: %v", err)
		}
		if !reflect.DeepEqual(actualResponse, sampleSyntax) {
			t.Errorf("handler returned unexpected body content: got %+v want %+v", actualResponse, sampleSyntax)
		}
	})

	// Test Case 2: Get filtered suggestions (for live typing)
	t.Run("GetFilteredSuggestionsMatch", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/filters/suggestions?term=h", nil)
		rr := httptest.NewRecorder()
		// Call the function from the mtaHttp package
		mtaHttp.HandleFilterSuggestions(rr, req, sampleSyntax) 

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
		// Call the function from the mtaHttp package
		mtaHttp.HandleFilterSuggestions(rr, req, sampleSyntax) 

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
		// Call the function from the mtaHttp package
		mtaHttp.HandleFilterSuggestions(rr, req, sampleSyntax) 

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code for empty term: got %v want %v", status, http.StatusOK)
		}

		expectedResponse, _ := json.Marshal(sampleSyntax)
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(string(expectedResponse)) {
			t.Errorf("handler returned unexpected body for empty term: got %v want %v", rr.Body.String(), string(expectedResponse))
		}
		
		var actualResponse []appconfig.FilterSyntaxEntry // Use appconfig.FilterSyntaxEntry
		if err := json.Unmarshal(rr.Body.Bytes(), &actualResponse); err != nil {
			t.Fatalf("could not unmarshal response body for empty term: %v", err)
		}
		if !reflect.DeepEqual(actualResponse, sampleSyntax) {
			t.Errorf("handler returned unexpected body content for empty term: got %+v want %+v", actualResponse, sampleSyntax)
		}
	})

	// Test Case 5: Empty filterSyntax input
	t.Run("EmptyFilterSyntax", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/filters/suggestions?term=h", nil)
		rr := httptest.NewRecorder()
		// Call the function from the mtaHttp package, pass empty slice of the correct type
		mtaHttp.HandleFilterSuggestions(rr, req, []appconfig.FilterSyntaxEntry{}) 

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code for empty syntax list: got %v want %v", status, http.StatusOK)
		}
		if strings.TrimSpace(rr.Body.String()) != "[]" {
			t.Errorf("handler returned unexpected body for empty syntax list: got %v want %v", rr.Body.String(), "[]")
		}
	})

	// Test Case 6: Empty filterSyntax input and no term
    t.Run("EmptyFilterSyntaxNoTerm", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/filters/suggestions", nil)
        rr := httptest.NewRecorder()
				// Call the function from the mtaHttp package, pass empty slice of the correct type
        mtaHttp.HandleFilterSuggestions(rr, req, []appconfig.FilterSyntaxEntry{}) 

        if status := rr.Code; status != http.StatusOK {
            t.Errorf("handler returned wrong status code for empty syntax list (no term): got %v want %v", status, http.StatusOK)
        }
        if strings.TrimSpace(rr.Body.String()) != "[]" {
            t.Errorf("handler returned unexpected body for empty syntax list (no term): got %v want %v", rr.Body.String(), "[]")
        }
    })
}
