package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mock-my-mta/smtp"
	"mock-my-mta/storage"
)

// mockStorage implements storage.Storage for testing HTTP handlers.
type mockStorage struct {
	emails      map[string]storage.EmailHeader
	rawEmails   map[string][]byte
	bodies      map[string]map[storage.EmailVersionType]string
	attachments map[string][]storage.AttachmentHeader
	attachment  map[string]storage.Attachment
	mailboxes   []storage.Mailbox
	err         error // if set, all methods return this error
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		emails:      make(map[string]storage.EmailHeader),
		rawEmails:   make(map[string][]byte),
		bodies:      make(map[string]map[storage.EmailVersionType]string),
		attachments: make(map[string][]storage.AttachmentHeader),
		attachment:  make(map[string]storage.Attachment),
	}
}

func (m *mockStorage) GetMailboxes() ([]storage.Mailbox, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.mailboxes, nil
}

func (m *mockStorage) GetEmailByID(emailID string) (storage.EmailHeader, error) {
	if m.err != nil {
		return storage.EmailHeader{}, m.err
	}
	email, ok := m.emails[emailID]
	if !ok {
		return storage.EmailHeader{}, fmt.Errorf("email not found: %s", emailID)
	}
	return email, nil
}

func (m *mockStorage) DeleteAllEmails() error {
	return m.err
}

func (m *mockStorage) DeleteEmailByID(emailID string) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.emails[emailID]; !ok {
		return fmt.Errorf("email not found: %s", emailID)
	}
	delete(m.emails, emailID)
	return nil
}

func (m *mockStorage) GetBodyVersion(emailID string, version storage.EmailVersionType) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	versions, ok := m.bodies[emailID]
	if !ok {
		return "", fmt.Errorf("email not found: %s", emailID)
	}
	body, ok := versions[version]
	if !ok {
		return "", nil
	}
	return body, nil
}

func (m *mockStorage) GetAttachments(emailID string) ([]storage.AttachmentHeader, error) {
	if m.err != nil {
		return nil, m.err
	}
	atts, ok := m.attachments[emailID]
	if !ok {
		return nil, fmt.Errorf("email not found: %s", emailID)
	}
	return atts, nil
}

func (m *mockStorage) GetAttachment(emailID string, attachmentID string) (storage.Attachment, error) {
	if m.err != nil {
		return storage.Attachment{}, m.err
	}
	key := emailID + "/" + attachmentID
	att, ok := m.attachment[key]
	if !ok {
		return storage.Attachment{}, fmt.Errorf("attachment not found: %s/%s", emailID, attachmentID)
	}
	return att, nil
}

func (m *mockStorage) SearchEmails(query string, page, pageSize int) ([]storage.EmailHeader, int, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	var results []storage.EmailHeader
	for _, e := range m.emails {
		results = append(results, e)
	}
	return results, len(results), nil
}

func (m *mockStorage) GetRawEmail(emailID string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	raw, ok := m.rawEmails[emailID]
	if !ok {
		return nil, fmt.Errorf("email not found: %s", emailID)
	}
	return raw, nil
}

func newTestServer(store storage.Storage) *Server {
	config := Configuration{Addr: ":0", Debug: false}
	relays := smtp.RelayConfigurations{}
	return NewServer(config, relays, store)
}

// --- Tests ---

func TestGetEmailByID_NotFound(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/nonexistent-id", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "not found") {
		t.Errorf("expected error message containing 'not found', got %q", rr.Body.String())
	}
}

func TestGetEmailByID_Success(t *testing.T) {
	store := newMockStorage()
	store.emails["test-123"] = storage.EmailHeader{
		ID:      "test-123",
		Subject: "Test Email",
		From:    storage.EmailAddress{Address: "from@test.com"},
		Date:    time.Now(),
	}
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/test-123", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var email storage.EmailHeader
	if err := json.Unmarshal(rr.Body.Bytes(), &email); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	if email.Subject != "Test Email" {
		t.Errorf("expected subject 'Test Email', got %q", email.Subject)
	}
}

func TestDeleteEmailByID_NotFound(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	req := httptest.NewRequest("DELETE", "/api/emails/nonexistent-id", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestGetBodyVersion_InvalidVersion(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/test-123/body/invalid-version", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid body version, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetBodyVersion_EmailNotFound(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/nonexistent/body/html", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetAttachments_EmailNotFound(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/nonexistent/attachments/", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestGetAttachmentContent_NotFound(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/test-123/attachments/bad-att/content", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestRelayMessage_MalformedJSON(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	req := httptest.NewRequest("POST", "/api/emails/test-123/relay", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for malformed JSON, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRelayMessage_RelayNotFound(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	body := `{"relay_name":"nonexistent","sender":"a@b.com","recipients":["c@d.com"]}`
	req := httptest.NewRequest("POST", "/api/emails/test-123/relay", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for unknown relay, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetEmails_InvalidPageParameter(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	tests := []struct {
		name string
		url  string
	}{
		{"page=0", "/api/emails/?page=0"},
		{"page=-1", "/api/emails/?page=-1"},
		{"page=abc", "/api/emails/?page=abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			rr := httptest.NewRecorder()
			srv.server.Handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestGetEmails_Success(t *testing.T) {
	store := newMockStorage()
	store.emails["test-1"] = storage.EmailHeader{
		ID:      "test-1",
		Subject: "Email 1",
		Date:    time.Now(),
	}
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp SearchEmailsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	if len(resp.Emails) != 1 {
		t.Errorf("expected 1 email, got %d", len(resp.Emails))
	}
	if resp.Pagination.TotalMatches != 1 {
		t.Errorf("expected total_matches=1, got %d", resp.Pagination.TotalMatches)
	}
}

func TestDeleteAllEmails_StorageError(t *testing.T) {
	store := newMockStorage()
	store.err = fmt.Errorf("storage failure")
	srv := newTestServer(store)

	req := httptest.NewRequest("DELETE", "/api/emails/", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestDownloadEmail_NotFound(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/nonexistent/download", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestDownloadEmail_Success(t *testing.T) {
	store := newMockStorage()
	store.rawEmails["test-123"] = []byte("From: a@b.com\r\nSubject: Test\r\n\r\nBody")
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/test-123/download", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "message/rfc822" {
		t.Errorf("expected Content-Type message/rfc822, got %q", ct)
	}
	if cd := rr.Header().Get("Content-Disposition"); !strings.Contains(cd, ".eml") {
		t.Errorf("expected Content-Disposition with .eml, got %q", cd)
	}
}

func TestGetHeaders_NotFound(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/nonexistent/headers", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestGetMimeTree_NotFound(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/nonexistent/mime-tree", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		absent   string
	}{
		{
			name:     "removes script tags",
			input:    `<p>Hello</p><script>alert('xss')</script><p>World</p>`,
			contains: "<p>Hello</p>",
			absent:   "<script>",
		},
		{
			name:     "removes event handlers",
			input:    `<img src="x" onerror="alert('xss')" alt="pic">`,
			contains: `<img src="x"`,
			absent:   "onerror",
		},
		{
			name:     "preserves normal HTML",
			input:    `<html><body><p>Normal content</p></body></html>`,
			contains: "<p>Normal content</p>",
		},
		{
			name:     "removes onload",
			input:    `<body onload="doEvil()"><p>Content</p></body>`,
			contains: "<p>Content</p>",
			absent:   "onload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeHTML(tt.input)
			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("expected result to contain %q, got %q", tt.contains, result)
			}
			if tt.absent != "" && strings.Contains(result, tt.absent) {
				t.Errorf("expected result NOT to contain %q, got %q", tt.absent, result)
			}
		})
	}
}

func TestBulkDelete_Success(t *testing.T) {
	store := newMockStorage()
	store.emails["email-1"] = storage.EmailHeader{ID: "email-1", Subject: "E1"}
	store.emails["email-2"] = storage.EmailHeader{ID: "email-2", Subject: "E2"}
	store.emails["email-3"] = storage.EmailHeader{ID: "email-3", Subject: "E3"}
	srv := newTestServer(store)

	body := `{"ids":["email-1","email-3"]}`
	req := httptest.NewRequest("POST", "/api/emails/bulk-delete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var result BulkResult
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	if len(result.Succeeded) != 2 {
		t.Errorf("expected 2 succeeded, got %d", len(result.Succeeded))
	}
	// email-2 should still exist
	if _, ok := store.emails["email-2"]; !ok {
		t.Error("email-2 should not have been deleted")
	}
}

func TestBulkDelete_PartialFailure(t *testing.T) {
	store := newMockStorage()
	store.emails["email-1"] = storage.EmailHeader{ID: "email-1", Subject: "E1"}
	// email-2 doesn't exist — should fail
	srv := newTestServer(store)

	body := `{"ids":["email-1","email-2"]}`
	req := httptest.NewRequest("POST", "/api/emails/bulk-delete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var result BulkResult
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	if len(result.Succeeded) != 1 {
		t.Errorf("expected 1 succeeded, got %d", len(result.Succeeded))
	}
	if len(result.Failed) != 1 {
		t.Errorf("expected 1 failed, got %d", len(result.Failed))
	}
}

func TestBulkDelete_MalformedJSON(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	req := httptest.NewRequest("POST", "/api/emails/bulk-delete", strings.NewReader("{bad"))
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestGetStats(t *testing.T) {
	store := newMockStorage()
	store.emails["e1"] = storage.EmailHeader{ID: "e1", Subject: "Test"}
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/stats", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var stats map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
		t.Fatalf("could not unmarshal: %v", err)
	}
	if stats["status"] != "ok" {
		t.Errorf("expected status ok, got %v", stats["status"])
	}
	if stats["email_count"].(float64) != 1 {
		t.Errorf("expected email_count=1, got %v", stats["email_count"])
	}
	if stats["uptime"] == nil {
		t.Error("expected uptime field")
	}
}

func TestWaitForEmail_ImmediateMatch(t *testing.T) {
	store := newMockStorage()
	store.emails["e1"] = storage.EmailHeader{ID: "e1", Subject: "Welcome"}
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/wait?query=Welcome&timeout=2s", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp WaitForEmailResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("could not unmarshal: %v", err)
	}
	if resp.Email.ID != "e1" {
		t.Errorf("expected email e1, got %v", resp.Email.ID)
	}
	if resp.TotalMatches != 1 {
		t.Errorf("expected total_matches=1, got %d", resp.TotalMatches)
	}
	if !strings.Contains(resp.URL, "/#/email/e1") {
		t.Errorf("expected URL containing /#/email/e1, got %q", resp.URL)
	}
}

func TestWaitForEmail_Timeout(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/wait?query=nonexistent&timeout=1s", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestTimeout {
		t.Errorf("expected 408, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWaitForEmail_InvalidTimeout(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/wait?query=test&timeout=invalid", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetPartByCID_EmailNotFound(t *testing.T) {
	store := newMockStorage()
	srv := newTestServer(store)

	req := httptest.NewRequest("GET", "/api/emails/nonexistent/cid/somecid", nil)
	rr := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}
