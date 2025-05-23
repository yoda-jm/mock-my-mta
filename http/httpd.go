package http

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/pprof"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"mock-my-mta/log"
	"mock-my-mta/smtp"
	"mock-my-mta/storage"
	"mock-my-mta/storage/multipart"
)

type Server struct {
	server *http.Server
	addr   string

	relayConfigurations smtp.RelayConfigurations

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

func NewServer(config Configuration, relayConfigurations smtp.RelayConfigurations, store storage.Storage) *Server {
	s := &Server{
		addr:                config.Addr,
		relayConfigurations: relayConfigurations,
		store:               store,
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
	apiRouter.HandleFunc("/emails/", s.deleteEmails).Methods("DELETE")
	apiRouter.HandleFunc("/emails/{email_id}", s.getEmailByID).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}", s.deleteEmailByID).Methods("DELETE")
	apiRouter.HandleFunc("/emails/{email_id}/body/{body_version}", s.getBodyVersion).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/relay", s.getRelayData).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/relay", s.relayMessage).Methods("POST")
	// Attachments
	apiRouter.HandleFunc("/emails/{email_id}/attachments/", s.getAttachments).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/attachments/{attachment_id}/content", s.getAttachmentContent).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/cid/{cid}", s.getPartByCID).Methods("GET")
	// return error if the requested route is not found
	apiRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeErrorResponse(w, http.StatusNotFound, "Not Found: %v", r.URL.Path)
	})

	// serve pprof routes
	pprofRouter := r.PathPrefix("/debug/pprof").Subrouter()
	AttachProfiler(pprofRouter)

	// Create GUI router
	// Serve static files from the "static" directory or embedded filesystem
	// depending on the debug flag
	filesystem := getFileSystem(staticDir, config.Debug)
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filename, err := locateFile(filesystem, r.URL.Path)
		if err != nil {
			// file not found, serve index.html instead
			log.Logf(log.INFO, "file not found: %v, defaulting to index.html", r.URL.Path)
			filename = "index.html"
		}
		log.Logf(log.DEBUG, "serving file: %v", filename)
		content, err := fs.ReadFile(filesystem, filename)
		if err != nil {
			fileSystemType := "embedded"
			if config.Debug {
				fileSystemType = "local"
			}
			writeErrorResponse(w, http.StatusInternalServerError, "cannot read %q from %v filesystem: %v", filename, fileSystemType, err)
			return
		}
		// determine content type based on file extension
		fileExtension := strings.ToLower(filename[strings.LastIndex(filename, ".")+1:])
		switch fileExtension {
		case "html":
			w.Header().Set("Content-Type", "text/html")
		case "css":
			w.Header().Set("Content-Type", "text/css")
		case "js":
			w.Header().Set("Content-Type", "application/javascript")
		case "json":
			w.Header().Set("Content-Type", "application/json")
		default:
			w.Header().Set("Content-Type", http.DetectContentType(content))
		}
		w.Write(content)
	})

	s.server = &http.Server{
		Addr:    config.Addr,
		Handler: r,
	}
	return s
}

func AttachProfiler(router *mux.Router) {
	router.HandleFunc("/", pprof.Index)
	router.HandleFunc("/cmdline", pprof.Cmdline)
	router.HandleFunc("/profile", pprof.Profile)
	router.HandleFunc("/symbol", pprof.Symbol)
	router.HandleFunc("/trace", pprof.Trace)
	router.HandleFunc("/allocs", pprof.Handler("allocs").ServeHTTP)
	router.HandleFunc("/goroutine", pprof.Handler("goroutine").ServeHTTP)
	router.HandleFunc("/heap", pprof.Handler("heap").ServeHTTP)
	router.HandleFunc("/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	router.HandleFunc("/block", pprof.Handler("block").ServeHTTP)
	router.HandleFunc("/mutex", pprof.Handler("mutex").ServeHTTP)
	router.HandleFunc("/profile", pprof.Handler("profile").ServeHTTP)
}

func locateFile(filesystem fs.FS, filepath string) (string, error) {
	if filepath == "/" {
		// serve index.html when the root path is requested
		return "index.html", nil
	}
	// check if file exists
	if stat, err := fs.Stat(filesystem, filepath); err == nil && !stat.IsDir() {
		return filepath, nil
	}
	// check if file exists when trimming the leading slash
	trimmedFilepath := strings.TrimPrefix(filepath, "/")
	if stat, err := fs.Stat(filesystem, trimmedFilepath); err == nil && !stat.IsDir() {
		return trimmedFilepath, nil
	}
	return "", fmt.Errorf("file not found: %v", filepath)
}

func getFileSystem(staticDir string, debug bool) fs.FS {
	if debug {
		log.Logf(log.INFO, "serving static files from directory: %v", staticDir)
		// check if static directory exists
		if _, err := os.Stat(staticDir); os.IsNotExist(err) {
			// static directory does not exist, revert to embedded filesystem
			log.Logf(log.WARNING, "static directory %v does not exist, serving static files from embedded filesystem", staticDir)
			staticContentFS, _ := fs.Sub(static, "static")
			return staticContentFS
		}
		return os.DirFS(staticDir)
	} else {
		log.Logf(log.INFO, "serving static files from embedded filesystem")
		staticContentFS, _ := fs.Sub(static, "static")
		return staticContentFS
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
		writeErrorResponse(w, http.StatusInternalServerError, "cannot get mailboxes: %v", err)
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
		writeErrorResponse(w, http.StatusInternalServerError, "cannot get email (id=%v): %v", emailID, err)
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
		writeErrorResponse(w, http.StatusInternalServerError, "cannot delete email (id=%v): %v", emailID, err)
		return
	}

	// Write the response
	w.WriteHeader(http.StatusNoContent)
}

type RelayData struct {
	RelayNames []string               `json:"relay_names"`
	Sender     storage.EmailAddress   `json:"sender"`
	Recipients []storage.EmailAddress `json:"recipients"`
}

func (s *Server) getRelayData(w http.ResponseWriter, r *http.Request) {
	// Get the email ID from the URL
	vars := mux.Vars(r)
	emailID := vars["email_id"]
	logf(generateRequestID(), r, log.DEBUG, "getting relay data by ID: %v", emailID)

	// Get the email by ID
	email, err := s.store.GetEmailByID(emailID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "cannot get email (id=%v): %v", emailID, err)
		return
	}

	// Write the response
	relayData := RelayData{
		RelayNames: s.relayConfigurations.Names(),
		Sender:     email.From,
		Recipients: append(email.Tos, email.CCs...),
	}
	writeJSONResponse(w, relayData)
}

type RelayMessageRequest struct {
	RelayName  string   `json:"relay_name"`
	Sender     string   `json:"sender"`
	Recipients []string `json:"recipients"`
}

func (s *Server) relayMessage(w http.ResponseWriter, r *http.Request) {
	// Get the email ID from the URL
	vars := mux.Vars(r)
	emailID := vars["email_id"]
	logf(generateRequestID(), r, log.DEBUG, "relaying message by ID: %v", emailID)

	// Parse the request body
	var request RelayMessageRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "cannot parse request body: %v", err)
		return
	}

	// Find the relay configuration by name
	relayConfig, found := s.relayConfigurations.Get(request.RelayName)
	if !found {
		writeErrorResponse(w, http.StatusBadRequest, "relay %q not found", request.RelayName)
		return
	}

	// Get the email by ID
	rawData, err := s.store.GetBodyVersion(emailID, storage.EmailVersionRaw)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "cannot get email (id=%v): %v", emailID, err)
		return
	}

	// Relay the message
	logf(generateRequestID(), r, log.INFO, "relaying message to %v", relayConfig.Addr)
	envelope := smtp.Envelope{
		Sender:     request.Sender,
		Recipients: request.Recipients,
		Data:       []byte(rawData),
	}

	err = smtp.RelayMessage(relayConfig, emailID, envelope)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "cannot relay message (id=%v): %v", emailID, err)
		return
	}
}

func (s *Server) getBodyVersion(w http.ResponseWriter, r *http.Request) {
	// Get the email ID and body version from the URL
	vars := mux.Vars(r)
	emailID := vars["email_id"]
	versionString := vars["body_version"]
	logf(generateRequestID(), r, log.DEBUG, "getting body version by ID: %v, version: %v", emailID, versionString)
	version, err := storage.ParseEmailVersionType(versionString)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "cannot parse body version %v for email %v: %v", versionString, emailID, err)
		return
	}

	// Get the body version by ID
	body, err := s.store.GetBodyVersion(emailID, version)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "cannot get body version %v for email %v: %v", version, emailID, err)
		return
	}

	// If the body is HTML, replace cid: links
	if version == storage.EmailVersionHtml || version == storage.EmailVersionWatchHtml {
		// Regex to find src="cid:..." or src='cid:...'
		// It captures the content of cid (the actual CID) in group 1
		cidRegex := regexp.MustCompile(`src=["']cid:([^"']+)["']`)
		// Replacement pattern uses $1 to refer to the captured group (the CID)
		replacementPattern := fmt.Sprintf("src=\"/api/emails/%s/cid/$1\"", emailID)
		body = cidRegex.ReplaceAllString(body, replacementPattern)
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
		writeErrorResponse(w, http.StatusInternalServerError, "cannot get attachments for email %v: %v", emailID, err)
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
		writeErrorResponse(w, http.StatusInternalServerError, "cannot get attachment %v for email %v: %v", attachmentID, emailID, err)
		return
	}

	// Write the response
	w.Header().Set("Content-Disposition", "attachment; filename="+attachment.Filename)
	w.Header().Set("Content-Type", attachment.ContentType)
	w.Write(attachment.Data)
}

func (s *Server) getPartByCID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	emailID := vars["email_id"]
	cid := vars["cid"]
	requestID := generateRequestID()
	logf(requestID, r, log.DEBUG, "getting part by email ID: %v, CID: %v", emailID, cid)

	// Get the raw email data first
	rawEmail, err := s.store.GetRawEmail(emailID)
	if err != nil {
		logf(requestID, r, log.ERROR, "cannot get raw email (id=%v): %v", emailID, err)
		if strings.Contains(err.Error(), "not found") { // Basic check, could be more robust
			writeErrorResponse(w, http.StatusNotFound, "email not found: %v", emailID)
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "cannot get raw email (id=%v): %v", emailID, err)
		}
		return
	}

	// Parse the email
	// Note: We need to use the multipart package directly here for ParseEmailFromBytes
	parsedMail, err := multipart.ParseEmailFromBytes(rawEmail)
	if err != nil {
		logf(requestID, r, log.ERROR, "cannot parse email (id=%v): %v", emailID, err)
		writeErrorResponse(w, http.StatusInternalServerError, "cannot parse email (id=%v): %v", emailID, err)
		return
	}

	// Get the part by CID
	part, found := parsedMail.GetPartByCID(cid)
	if !found {
		logf(requestID, r, log.DEBUG, "part with CID %v not found in email %v", cid, emailID)
		writeErrorResponse(w, http.StatusNotFound, "part with CID %s not found", cid)
		return
	}

	// Get part's content type and body
	contentType := part.GetHeader("Content-Type") // This should provide the full content type string
	if len(contentType) == 0 {
		logf(requestID, r, log.WARNING, "part with CID %v in email %v has no Content-Type header", cid, emailID)
		// Default to application/octet-stream if Content-Type is missing
		w.Header().Set("Content-Type", "application/octet-stream")
	} else {
		w.Header().Set("Content-Type", contentType[0])
	}

	body := part.GetDecodedBody()

	// Write the response
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	// Consider adding "Content-Disposition: inline" if appropriate for typical CID uses (like images)
	// w.Header().Set("Content-Disposition", "inline")
	w.Write([]byte(body))
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

func (s *Server) deleteEmails(w http.ResponseWriter, r *http.Request) {
	err := s.store.DeleteAllEmails()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "cannot delete all emails: %v", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
		writeErrorResponse(w, http.StatusBadRequest, "cannot parse page parameters: %v", err)
		return
	}

	// Perform the search
	emailHeaders, totalMatches, err := s.store.SearchEmails(query, page, pageSize)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "cannot search emails: %v", err)
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
		writeErrorResponse(w, http.StatusInternalServerError, "cannot encode JSON: %v", err)
	}
}

func writeErrorResponse(w http.ResponseWriter, status int, messageFormat string, args ...interface{}) {
	message := fmt.Sprintf(messageFormat, args...)
	log.Logf(log.ERROR, "error: %v (status=%v)", message, status)
	http.Error(w, message, status)
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
