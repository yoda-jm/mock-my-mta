package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"mock-my-mta/email"
	"mock-my-mta/log"
	"mock-my-mta/storage"
)

type httpServer struct {
	server *http.Server
	addr   string

	store storage.Storage
}

func newHttpServer(addr string, store storage.Storage) *httpServer {
	s := &httpServer{
		addr:  addr,
		store: store,
	}

	// Create a new Gorilla Mux router
	r := mux.NewRouter()

	// Create a sub-router for the /api endpoint
	apiRouter := r.PathPrefix("/api").Subrouter()

	// Define the API routes
	apiRouter.HandleFunc("/emails", s.getEmails).Methods("GET")
	apiRouter.HandleFunc("/emails/{id}", s.getEmailByID).Methods("GET")
	apiRouter.HandleFunc("/emails/{id}", s.deleteEmailByID).Methods("DELETE")
	apiRouter.HandleFunc("/emails/{email_id}/body/{body_version}", s.getBodyVersion).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/attachments", s.getAttachments).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/attachments/{attachment_id}", s.getAttachmentByID).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/attachments/{attachment_id}/content", s.getAttachmentContent).Methods("GET")

	// Create GUI router
	// Serve static files from the "static" directory
	staticDir := "./static"
	fileServer := http.FileServer(http.Dir(staticDir))
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the file exists before serving
		filePath := filepath.Join(staticDir, r.URL.Path)
		if _, err := os.Stat(filePath); err == nil {
			fileServer.ServeHTTP(w, r)
		} else {
			// Redirect to index.html if the file is not found
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
		}
	})

	s.server = &http.Server{
		Addr:    addr,
		Handler: r,
	}
	return s
}

func (s *httpServer) Start() error {
	log.Logf(log.INFO, "starting http server on %v", s.addr)
	return s.server.ListenAndServe()
}

func (s *httpServer) Stop() error {
	log.Logf(log.INFO, "stopping http server...", s.addr)
	return s.server.Shutdown(context.TODO())
}

func newJsonEncoder(w http.ResponseWriter) *json.Encoder {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder
}

type EmailJson struct {
	ID            uuid.UUID `json:"id"`
	Sender        string    `json:"sender"`
	Recipents     []string  `json:"recipients"`
	Subject       string    `json:"subject"`
	ReceivedTime  time.Time `json:"received_time"`
	BodyVersions  []string  `json:"body_versions"`
	HasAttachment bool      `json:"has_attachment"`
}

func NewEmailJson(emailData *storage.EmailData) EmailJson {
	var bodyVersions []string
	for _, version := range emailData.Email.GetVersions() {
		bodyVersions = append(bodyVersions, version.String())
	}
	return EmailJson{
		ID:            emailData.ID,
		Sender:        emailData.Email.GetSender(),
		Recipents:     emailData.Email.GetRecipients(),
		Subject:       emailData.Email.GetSubject(),
		ReceivedTime:  emailData.ReceivedTime,
		BodyVersions:  bodyVersions,
		HasAttachment: len(emailData.Email.GetAttachments()) > 0,
	}
}

func (s *httpServer) getEmails(w http.ResponseWriter, r *http.Request) {
	sort, err := storage.ParseSortFieldEnum(r.URL.Query().Get("sort"))
	if err != nil {
		sort = storage.SortDateField
	}
	order, err := storage.ParseSortType(r.URL.Query().Get("order"))
	if err != nil {
		if sort == storage.SortDateField {
			order = storage.Descending
		} else {
			order = storage.Ascending
		}
	}

	so := storage.SortOption{
		Field:     sort,
		Direction: order,
	}
	mo := storage.MatchOption{
		Field:         storage.MatchNoField,
		Type:          storage.ExactMatch,
		CaseSensitive: true,
		HasAttachment: false,
	}
	searchPattern := ""
	uuids, err := s.store.Find(mo, so, searchPattern)
	if err != nil {
		// return 500
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		newJsonEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("internal server error: %v", err)})
		return

	}
	var emails []EmailJson
	for _, uuid := range uuids {
		emailData, err := s.store.Get(uuid)
		if err != nil {
			// return 500
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			newJsonEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("internal server error: %v", err)})
			return
		}
		emails = append(emails, NewEmailJson(emailData))
	}
	w.Header().Set("Content-Type", "application/json")
	newJsonEncoder(w).Encode(emails)
}

func (s *httpServer) getEmailByID(w http.ResponseWriter, r *http.Request) {
	// Get the email ID from the URL path
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	// Retrieve the email data from the data store
	emailData, err := s.store.Get(id)
	if err != nil {
		http.Error(w, "Email not found", http.StatusBadRequest)
		return
	}

	// Write the email to the response as JSON
	w.Header().Set("Content-Type", "application/json")
	newJsonEncoder(w).Encode(NewEmailJson(emailData))
}

func (s *httpServer) deleteEmailByID(w http.ResponseWriter, r *http.Request) {
	// Get the email ID from the URL path
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	// Retrieve the email data from the data store
	err = s.store.Delete(id)
	if err != nil {
		http.Error(w, "Email not found", http.StatusBadRequest)
		return
	}
}

type AttachmentJson struct {
	ID        uuid.UUID `json:"id"`
	MediaType string    `json:"media_type"`
	Filename  string    `json:"filename"`
}

func NewAttachmentJson(attachment *email.Attachment) AttachmentJson {
	return AttachmentJson{
		ID:        attachment.GetID(),
		MediaType: attachment.GetMediaType(),
		Filename:  attachment.GetFilename(),
	}
}

func getAttachment(store storage.Storage, emailID, attachmentID uuid.UUID) (*email.Attachment, error) {
	emailData, err := store.Get(emailID)
	if err != nil {
		return nil, fmt.Errorf("cannot find email %q: %v", emailID, err)
	}
	attachment, found := emailData.Email.GetAttachment(attachmentID)
	if !found {
		return nil, fmt.Errorf("cannot find attachment %q in email %q", attachmentID, emailID)
	}
	return attachment, nil
}

func getBody(store storage.Storage, emailID uuid.UUID, bodyVersion email.EmailVersionTypeEnum) (string, error) {
	emailData, err := store.Get(emailID)
	if err != nil {
		return "", fmt.Errorf("cannot find email %q: %v", emailID, err)
	}
	return emailData.Email.GetBody(bodyVersion)
}

func (s *httpServer) getBodyVersion(w http.ResponseWriter, r *http.Request) {
	// Get the email ID from the URL path
	vars := mux.Vars(r)
	emailID, err := uuid.Parse(vars["email_id"])
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}
	bodyVersion, err := email.ParseEmailVersionTypeEnum(vars["body_version"])
	if err != nil {
		http.Error(w, "Invalid body version", http.StatusBadRequest)
		return
	}

	// Retrieve the email data from the data store
	body, err := getBody(s.store, emailID, bodyVersion)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Write the attachment to the response as JSON
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(body))
}

func (s *httpServer) getAttachments(w http.ResponseWriter, r *http.Request) {
	// Get the email ID from the URL path
	vars := mux.Vars(r)
	emailID, err := uuid.Parse(vars["email_id"])
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}
	emailData, err := s.store.Get(emailID)
	if err != nil {
		http.Error(w, "cannot find email", http.StatusBadRequest)
	}
	attachmentIDs := emailData.Email.GetAttachments()
	var attachments []AttachmentJson
	for _, attachmentID := range attachmentIDs {
		attachment, found := emailData.Email.GetAttachment(attachmentID)
		if !found {
			continue
		}
		attachments = append(attachments, NewAttachmentJson(attachment))
	}
	w.Header().Set("Content-Type", "application/json")
	newJsonEncoder(w).Encode(attachments)
}

func (s *httpServer) getAttachmentByID(w http.ResponseWriter, r *http.Request) {
	// Get the email ID from the URL path
	vars := mux.Vars(r)
	emailID, err := uuid.Parse(vars["email_id"])
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}
	attachmentID, err := uuid.Parse(vars["attachment_id"])
	if err != nil {
		http.Error(w, "Invalid attachment ID", http.StatusBadRequest)
		return
	}
	// Retrieve the email data from the data store
	attachment, err := getAttachment(s.store, emailID, attachmentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Write the attachment to the response as JSON
	w.Header().Set("Content-Type", "application/json")
	newJsonEncoder(w).Encode(NewAttachmentJson(attachment))
}

func (s *httpServer) getAttachmentContent(w http.ResponseWriter, r *http.Request) {
	// Get the email ID from the URL path
	vars := mux.Vars(r)
	emailID, err := uuid.Parse(vars["email_id"])
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}
	attachmentID, err := uuid.Parse(vars["attachment_id"])
	if err != nil {
		http.Error(w, "Invalid attachment ID", http.StatusBadRequest)
		return
	}
	// Retrieve the email data from the data store
	attachment, err := getAttachment(s.store, emailID, attachmentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Write the attachment to the response as JSON
	w.Header().Set("Content-Disposition", "attachment; filename="+attachment.GetFilename())
	w.Header().Set("Content-Type", attachment.GetMediaType())
	w.Write(attachment.GetContent())
}
