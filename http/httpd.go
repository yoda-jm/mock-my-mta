package http

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/microcosm-cc/bluemonday"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"mock-my-mta/log"
	"mock-my-mta/smtp"
	"mock-my-mta/storage"
	"mock-my-mta/storage/multipart"
)

type Server struct {
	server    *http.Server
	addr      string
	startTime time.Time

	relayConfigurations smtp.RelayConfigurations

	store    storage.Storage
	readEmails sync.Map // tracks which email IDs have been read
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
		startTime:           time.Now(),
		relayConfigurations: relayConfigurations,
		store:               store,
	}

	// Create a new Gorilla Mux router
	r := mux.NewRouter()

	// Create a sub-router for the /api endpoint
	apiRouter := r.PathPrefix("/api").Subrouter()

	// Define the API routes

	// Mailboxes
	apiRouter.HandleFunc("/mailboxes", s.getMailboxes).Methods("GET")
	// Emails
	apiRouter.HandleFunc("/emails/wait", s.waitForEmail).Methods("GET")
	apiRouter.HandleFunc("/emails/", s.getEmails).Methods("GET")
	apiRouter.HandleFunc("/emails/", s.deleteEmails).Methods("DELETE")
	apiRouter.HandleFunc("/emails/bulk-delete", s.bulkDeleteEmails).Methods("POST")
	apiRouter.HandleFunc("/emails/bulk-relay", s.bulkRelayEmails).Methods("POST")
	apiRouter.HandleFunc("/emails/bulk-mark-read", s.bulkMarkRead).Methods("POST")
	apiRouter.HandleFunc("/emails/bulk-mark-unread", s.bulkMarkUnread).Methods("POST")
	apiRouter.HandleFunc("/emails/{email_id}", s.getEmailByID).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}", s.deleteEmailByID).Methods("DELETE")
	apiRouter.HandleFunc("/emails/{email_id}/body/{body_version}", s.getBodyVersion).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/headers", s.getEmailHeaders).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/download", s.downloadEmail).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/mime-tree", s.getMimeTree).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/relay", s.getRelayData).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/relay", s.relayMessage).Methods("POST")
	// Attachments
	apiRouter.HandleFunc("/emails/{email_id}/attachments/", s.getAttachments).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/attachments/{attachment_id}/content", s.getAttachmentContent).Methods("GET")
	apiRouter.HandleFunc("/emails/{email_id}/cid/{cid}", s.getPartByCID).Methods("GET")
	// Filter suggestions
	apiRouter.HandleFunc("/filters/suggestions", getFilterSuggestions).Methods("GET")
	// Health, stats, and settings
	apiRouter.HandleFunc("/health", s.getHealth).Methods("GET")
	apiRouter.HandleFunc("/stats", s.getStats).Methods("GET")
	apiRouter.HandleFunc("/settings", handleGetSettings).Methods("GET")
	apiRouter.HandleFunc("/settings", handlePutSettings).Methods("PUT")
	apiRouter.HandleFunc("/read-status", s.resetReadStatus).Methods("DELETE")
	// WebSocket for real-time notifications
	apiRouter.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r)
	})
	// return error if the requested route is not found
	apiRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeErrorResponse(w, http.StatusNotFound, "Not Found: %v", r.URL.Path)
	})

	// serve pprof routes (only in debug mode)
	if config.Debug {
		log.Logf(log.INFO, "debug mode enabled: pprof endpoints available at /debug/pprof/")
		pprofRouter := r.PathPrefix("/debug/pprof").Subrouter()
		AttachProfiler(pprofRouter)
	}

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

func (s *Server) Shutdown(ctx context.Context) error {
	log.Logf(log.INFO, "stopping http server on %v...", s.addr)
	return s.server.Shutdown(ctx)
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

	// Mark as read
	s.readEmails.Store(emailID, true)
	email.IsRead = true

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
	BroadcastEvent("delete_email", map[string]string{"id": emailID})
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

	// If the body is HTML, sanitize and replace cid: links
	if version == storage.EmailVersionHtml || version == storage.EmailVersionWatchHtml {
		// Sanitize: remove script tags and event handler attributes
		body = sanitizeHTML(body)
		// Replace cid: links with API endpoints
		cidRegex := regexp.MustCompile(`src=["']cid:([^"']+)["']`)
		replacementPattern := fmt.Sprintf("src=\"/api/emails/%s/cid/$1\"", emailID)
		body = cidRegex.ReplaceAllString(body, replacementPattern)
	}

	// Write the response
	writeJSONResponse(w, body)
}

func (s *Server) getEmailHeaders(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	emailID := vars["email_id"]
	logf(generateRequestID(), r, log.DEBUG, "getting headers for email: %v", emailID)

	rawEmail, err := s.store.GetRawEmail(emailID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "cannot get raw email (id=%v): %v", emailID, err)
		return
	}

	parsedMail, err := multipart.ParseEmailFromBytes(rawEmail)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "cannot parse email (id=%v): %v", emailID, err)
		return
	}

	writeJSONResponse(w, parsedMail.GetAllHeaders())
}

func (s *Server) downloadEmail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	emailID := vars["email_id"]
	logf(generateRequestID(), r, log.DEBUG, "downloading email: %v", emailID)

	rawEmail, err := s.store.GetRawEmail(emailID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "cannot get raw email (id=%v): %v", emailID, err)
		return
	}

	w.Header().Set("Content-Type", "message/rfc822")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.eml\"", emailID))
	w.Write(rawEmail)
}

func (s *Server) getMimeTree(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	emailID := vars["email_id"]
	logf(generateRequestID(), r, log.DEBUG, "getting MIME tree for email: %v", emailID)

	rawEmail, err := s.store.GetRawEmail(emailID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "cannot get raw email (id=%v): %v", emailID, err)
		return
	}

	parsedMail, err := multipart.ParseEmailFromBytes(rawEmail)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "cannot parse email (id=%v): %v", emailID, err)
		return
	}

	writeJSONResponse(w, parsedMail.GetMimeTree())
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
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, attachment.Filename))
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

type PaginationResponse struct {
	CurrentPage  int  `json:"current_page"`
	IsFirstPage  bool `json:"is_first_page"`
	IsLastPage   bool `json:"is_last_page"`
	TotalPages   int  `json:"total_pages"`
	TotalMatches int  `json:"total_matches"`
}

type SearchEmailsResponse struct {
	Emails []storage.EmailHeader `json:"emails"`
	Pagination PaginationResponse `json:"pagination"`
}

func (s *Server) deleteEmails(w http.ResponseWriter, r *http.Request) {
	err := s.store.DeleteAllEmails()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "cannot delete all emails: %v", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	BroadcastEvent("delete_all", nil)
}

type BulkDeleteRequest struct {
	IDs []string `json:"ids"`
}

type BulkResult struct {
	Succeeded []string `json:"succeeded"`
	Failed    []string `json:"failed"`
}

func (s *Server) bulkDeleteEmails(w http.ResponseWriter, r *http.Request) {
	var request BulkDeleteRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "cannot parse request body: %v", err)
		return
	}

	result := BulkResult{}
	for _, id := range request.IDs {
		if err := s.store.DeleteEmailByID(id); err != nil {
			result.Failed = append(result.Failed, id)
		} else {
			result.Succeeded = append(result.Succeeded, id)
		}
	}
	writeJSONResponse(w, result)
}

type BulkRelayRequest struct {
	IDs        []string `json:"ids"`
	RelayName  string   `json:"relay_name"`
	Sender     string   `json:"sender"`
	Recipients []string `json:"recipients"`
}

func (s *Server) bulkRelayEmails(w http.ResponseWriter, r *http.Request) {
	var request BulkRelayRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "cannot parse request body: %v", err)
		return
	}

	relayConfig, found := s.relayConfigurations.Get(request.RelayName)
	if !found {
		writeErrorResponse(w, http.StatusBadRequest, "relay %q not found", request.RelayName)
		return
	}

	result := BulkResult{}
	for _, id := range request.IDs {
		rawData, err := s.store.GetBodyVersion(id, storage.EmailVersionRaw)
		if err != nil {
			result.Failed = append(result.Failed, id)
			continue
		}
		envelope := smtp.Envelope{
			Sender:     request.Sender,
			Recipients: request.Recipients,
			Data:       []byte(rawData),
		}
		if err := smtp.RelayMessage(relayConfig, id, envelope); err != nil {
			result.Failed = append(result.Failed, id)
		} else {
			result.Succeeded = append(result.Succeeded, id)
		}
	}
	writeJSONResponse(w, result)
}

type BulkReadStatusRequest struct {
	IDs []string `json:"ids"`
}

func (s *Server) bulkMarkRead(w http.ResponseWriter, r *http.Request) {
	var request BulkReadStatusRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "cannot parse request body: %v", err)
		return
	}

	result := BulkResult{}
	for _, id := range request.IDs {
		s.readEmails.Store(id, true)
		result.Succeeded = append(result.Succeeded, id)
	}
	writeJSONResponse(w, result)
}

func (s *Server) bulkMarkUnread(w http.ResponseWriter, r *http.Request) {
	var request BulkReadStatusRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "cannot parse request body: %v", err)
		return
	}

	result := BulkResult{}
	for _, id := range request.IDs {
		s.readEmails.Delete(id)
		result.Succeeded = append(result.Succeeded, id)
	}
	writeJSONResponse(w, result)
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
	// Inject read status
	for i := range emailHeaders {
		if _, ok := s.readEmails.Load(emailHeaders[i].ID); ok {
			emailHeaders[i].IsRead = true
		}
	}
	isFirstPage := page == 1
	isLastPage := (page * pageSize) >= totalMatches
	totalPages := (totalMatches + pageSize - 1) / pageSize

	// Create the response
	searchResponse := SearchEmailsResponse{
		Emails: emailHeaders,
		Pagination: PaginationResponse{
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

func (s *Server) getHealth(w http.ResponseWriter, r *http.Request) {
	// Quick health check — verify storage is accessible
	_, _, err := s.store.SearchEmails("", 1, 1)
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "storage unhealthy: %v", err)
		return
	}
	writeJSONResponse(w, map[string]string{"status": "ok"})
}

func (s *Server) resetReadStatus(w http.ResponseWriter, r *http.Request) {
	s.readEmails = sync.Map{}
	BroadcastEvent("read_status_reset", nil)
	writeJSONResponse(w, map[string]string{"status": "ok"})
}

func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	emailCount := 0
	emails, total, err := s.store.SearchEmails("", 1, 1)
	if err == nil {
		emailCount = total
		_ = emails
	}

	stats := map[string]interface{}{
		"status":      "ok",
		"uptime":      time.Since(s.startTime).String(),
		"started_at":  s.startTime.Format(time.RFC3339),
		"email_count": emailCount,
		"http_addr":   s.addr,
	}
	writeJSONResponse(w, stats)
}

// waitForEmail long-polls until an email matching the query arrives or timeout.
// Usage: GET /api/emails/wait?query=from:alice@test.com&timeout=30s
// Returns the first matching email, or 408 Request Timeout.
func (s *Server) waitForEmail(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	timeoutStr := r.URL.Query().Get("timeout")
	if timeoutStr == "" {
		timeoutStr = "30s"
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid timeout: %v", err)
		return
	}
	if timeout > 5*time.Minute {
		timeout = 5 * time.Minute
	}

	requestID := generateRequestID()
	logf(requestID, r, log.DEBUG, "waiting for email matching %q (timeout=%v)", query, timeout)

	deadline := time.After(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	buildResponse := func(r *http.Request) *WaitForEmailResponse {
		return s.findMatchingEmails(r, query)
	}

	// Check immediately before starting the loop
	if resp := buildResponse(r); resp != nil {
		writeJSONResponse(w, resp)
		return
	}

	for {
		select {
		case <-deadline:
			writeErrorResponse(w, http.StatusRequestTimeout, "no email matching %q within %v", query, timeout)
			return
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if resp := buildResponse(r); resp != nil {
				logf(requestID, r, log.DEBUG, "found %d matching email(s)", resp.TotalMatches)
				writeJSONResponse(w, resp)
				return
			}
		}
	}
}

// WaitForEmailResponse is the response from the wait-for-email API.
type WaitForEmailResponse struct {
	Email        storage.EmailHeader `json:"email"`         // first matching email
	TotalMatches int                 `json:"total_matches"` // total number of matches
	URL          string              `json:"url"`           // deep link to view the email
}

func (s *Server) findMatchingEmails(r *http.Request, query string) *WaitForEmailResponse {
	emails, total, err := s.store.SearchEmails(query, 1, 1)
	if err != nil || total == 0 || len(emails) == 0 {
		return nil
	}
	// Build the deep link URL
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	emailURL := fmt.Sprintf("%s://%s/#/email/%s", scheme, host, emails[0].ID)

	return &WaitForEmailResponse{
		Email:        emails[0],
		TotalMatches: total,
		URL:          emailURL,
	}
}

// generateRequestID generates a unique request ID for each incoming request.
func generateRequestID() string {
	return uuid.New().String()
}

// emailHTMLPolicy is a bluemonday policy that allows safe HTML for email display.
// It permits standard formatting, images (for CID), links, and tables
// while stripping scripts, event handlers, and other XSS vectors.
var emailHTMLPolicy = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowImages()
	p.AllowStyling()
	p.AllowAttrs("style").Globally()
	p.AllowAttrs("class").Globally()
	p.AllowAttrs("width", "height", "alt", "title").OnElements("img")
	p.AllowAttrs("bgcolor", "cellpadding", "cellspacing", "border", "align", "valign").OnElements("table", "tr", "td", "th")
	p.AllowAttrs("colspan", "rowspan").OnElements("td", "th")
	p.AllowURLSchemeWithCustomPolicy("cid", func(url *url.URL) bool { return true })
	return p
}()

func sanitizeHTML(html string) string {
	return emailHTMLPolicy.Sanitize(html)
}
