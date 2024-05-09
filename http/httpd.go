package http

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"mock-my-mta/log"
	"mock-my-mta/storage"
)

type Server struct {
	server *http.Server
	addr   string

	store storage.Storage
}

// embed static directory
//
//go:embed static
var static embed.FS
var staticDir = "./http/static"

// logf is logging with a request ID and a log level
func logf(requestID string, r *http.Request, level log.LogLevel, format string, args ...interface{}) {
	logFormat := fmt.Sprintf("[request=%v,remote=%v]: %v", requestID, r.RemoteAddr, format)
	log.Logf(level, logFormat, args...)
}

func NewServer(addr string, debug bool, store storage.Storage) *Server {
	s := &Server{
		addr:  addr,
		store: store,
	}

	// Create a new Gorilla Mux router
	r := mux.NewRouter()

	// Create a sub-router for the /api endpoint
	apiRouter := r.PathPrefix("/api").Subrouter()

	// Define the API routes

	// Mailboxes
	apiRouter.HandleFunc(("/mailboxes"), s.getMailboxes).Methods("GET")
	// Emails
	apiRouter.HandleFunc("/emails/", s.getEmails).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}", s.getEmailByID).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}", s.deleteEmailByID).Methods("DELETE")
	apiRouter.HandleFunc("/emails/{email_id}/body/{body_version}", s.getBodyVersion).Methods("GET")
	// Attachments
	apiRouter.HandleFunc("/emails/{email_id}/attachments/", s.getAttachments).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/attachments/{attachment_id}/content", s.getAttachmentContent).Methods("GET")
	// return error if the requested route is not found
	apiRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logf(generateRequestID(), r, log.INFO, "Not Found: %v", r.URL.Path)
		http.Error(w, "Not Found", http.StatusNotFound)
	})

	// Create GUI router
	// Serve static files from the "static" directory
	filesystem, httpFileSystem := getHttpFileSystem(staticDir, debug)

	fileServer := http.FileServer(httpFileSystem)
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fileExists(filesystem, r.URL.Path) {
			fileServer.ServeHTTP(w, r)
			return
		}
		// server index.html for all other requests
		content, err := fs.ReadFile(filesystem, "index.html")
		if err != nil {
			message := fmt.Sprintf("cannot read index.html from embedded filesystem: %v", err)
			logf(generateRequestID(), r, log.ERROR, "error: %v", message)
			http.Error(w, message, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(content)
	})

	s.server = &http.Server{
		Addr:    addr,
		Handler: r,
	}
	return s
}

func fileExists(filesystem fs.FS, filepath string) bool {
	stat, err := fs.Stat(filesystem, filepath)
	return err == nil && !stat.IsDir()
}

func getHttpFileSystem(staticDir string, debug bool) (fs.FS, http.FileSystem) {
	if debug {
		log.Logf(log.INFO, "serving static files from directory: %v", staticDir)
		// check if static directory exists
		if _, err := os.Stat(staticDir); os.IsNotExist(err) {
			// static directory does not exist, revert to embedded filesystem
			log.Logf(log.WARNING, "static directory %v does not exist, serving static files from embedded filesystem", staticDir)
			staticContentFS, _ := fs.Sub(static, "static")
			return staticContentFS, http.FS(static)
		}
		return os.DirFS(staticDir), http.Dir(staticDir)
	} else {
		log.Logf(log.INFO, "serving static files from embedded filesystem")
		staticContentFS, _ := fs.Sub(static, "static")
		return staticContentFS, http.FS(static)
	}
}

func (s *Server) ListenAndServe() error {
	log.Logf(log.INFO, "starting http server on %v", s.addr)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown() error {
	log.Logf(log.INFO, "stopping http server...", s.addr)
	return s.server.Shutdown(context.TODO())
}

func (s *Server) getMailboxes(w http.ResponseWriter, r *http.Request) {
	// Get all mailboxes
	logf(generateRequestID(), r, log.DEBUG, "getting mailboxes")
	mailboxes, err := s.store.GetMailboxes()
	if err != nil {
		message := fmt.Sprintf("Internal Server Error: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	// Write the response
	writeJSONResponse(w, mailboxes)
}

func (s *Server) getEmailByID(w http.ResponseWriter, r *http.Request) {
	// Get the email ID from the URL
	vars := mux.Vars(r)
	emailID := vars["email_id"]
	logf(generateRequestID(), r, log.DEBUG, "getting email by ID: %v", emailID)

	// Get the email by ID
	email, err := s.store.GetEmailByID(emailID)
	if err != nil {
		message := fmt.Sprintf("Internal Server Error: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	// Write the response
	writeJSONResponse(w, email)
}

func (s *Server) deleteEmailByID(w http.ResponseWriter, r *http.Request) {
	// Get the email ID from the URL
	vars := mux.Vars(r)
	emailID := vars["email_id"]
	logf(generateRequestID(), r, log.DEBUG, "deleting email by ID: %v", emailID)

	// Delete the email by ID
	err := s.store.DeleteEmailByID(emailID)
	if err != nil {
		message := fmt.Sprintf("Internal Server Error: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	// Write the response
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getBodyVersion(w http.ResponseWriter, r *http.Request) {
	// Get the email ID and body version from the URL
	vars := mux.Vars(r)
	emailID := vars["email_id"]
	versionString := vars["body_version"]
	logf(generateRequestID(), r, log.DEBUG, "getting body version by ID: %v, version: %v", emailID, versionString)
	version, err := storage.ParseEmailVersionType(versionString)
	if err != nil {
		message := fmt.Sprintf("Bad Request: invalid body version: %v", versionString)
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	// Get the body version by ID
	body, err := s.store.GetBodyVersion(emailID, version)
	if err != nil {
		message := fmt.Sprintf("Internal Server Error: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	// Write the response
	writeJSONResponse(w, body)
}

func (s *Server) getAttachments(w http.ResponseWriter, r *http.Request) {
	// Get the email ID from the URL
	vars := mux.Vars(r)
	emailID := vars["email_id"]
	logf(generateRequestID(), r, log.DEBUG, "getting attachments by email ID: %v", emailID)

	// Get all attachments for the specified email
	attachments, err := s.store.GetAttachments(emailID)
	if err != nil {
		message := fmt.Sprintf("Internal Server Error: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	// Write the response
	writeJSONResponse(w, attachments)
}

func (s *Server) getAttachmentContent(w http.ResponseWriter, r *http.Request) {
	// Get the email ID and attachment ID from the URL
	vars := mux.Vars(r)
	emailID := vars["email_id"]
	attachmentID := vars["attachment_id"]
	logf(generateRequestID(), r, log.DEBUG, "getting attachment content by email ID: %v, attachment ID: %v", emailID, attachmentID)

	// Get the attachment content by ID
	attachment, err := s.store.GetAttachment(emailID, attachmentID)
	if err != nil {
		message := fmt.Sprintf("Internal Server Error: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	// Write the response
	w.Header().Set("Content-Disposition", "attachment; filename="+attachment.Filename)
	w.Header().Set("Content-Type", attachment.ContentType)
	w.Write(attachment.Data)
}

type PaginnationResponse struct {
	CurrentPage  int  `json:"current_page"`
	IsFirstPage  bool `json:"is_first_page"`
	IsLastPage   bool `json:"is_last_page"`
	TotalPages   int  `json:"total_pages"`
	TotalMatches int  `json:"total_matches"`
}

type SearchEmailsResponse struct {
	Emails []storage.EmailHeader `json:"emails"`
	// FIXME: add a nice pagination result
	Paginnation PaginnationResponse `json:"pagination"`
}

func (s *Server) getEmails(w http.ResponseWriter, r *http.Request) {
	// Get the query parameter from the URL
	query := r.URL.Query().Get("query")
	if query != "" {
		logf(generateRequestID(), r, log.DEBUG, "searching emails with query: %q", query)
	} else {
		logf(generateRequestID(), r, log.DEBUG, "getting all emails")
	}

	// Parse the page parameters
	page, pageSize, err := parsePageParameters(r)
	if err != nil {
		message := fmt.Sprintf("Bad Request: %v", err)
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	// Perform the search
	emailHeaders, totalMatches, err := s.store.SearchEmails(query, page, pageSize)
	if err != nil {
		message := fmt.Sprintf("Internal Server Error: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}
	isFirstPage := page == 1
	isLastPage := (page * pageSize) >= totalMatches
	totalPages := (totalMatches + pageSize - 1) / pageSize

	// Create the response
	searchResponse := SearchEmailsResponse{
		Emails: emailHeaders,
		Paginnation: PaginnationResponse{
			CurrentPage:  page,
			IsFirstPage:  isFirstPage,
			IsLastPage:   isLastPage,
			TotalPages:   totalPages,
			TotalMatches: totalMatches,
		},
	}

	// Write the response
	writeJSONResponse(w, searchResponse)
}

func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		message := fmt.Sprintf("Internal Server Error: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
	}
}

func parsePageParameters(r *http.Request) (int, int, error) {
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return 0, 0, fmt.Errorf("invalid page number")
		}
	}
	pageSize := 20

	return page, pageSize, nil
}

// generateRequestID generates a unique request ID for each incoming request.
func generateRequestID() string {
	return uuid.New().String()
}
