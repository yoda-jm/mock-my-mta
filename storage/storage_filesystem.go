package storage

import (
	"bufio"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"sort"

	"mock-my-mta/log"
	"mock-my-mta/storage/matcher"
	"mock-my-mta/storage/multipart"
)

type filesystemType string

const (
	FileStorageTypeEML     filesystemType = "eml"
	FileStorageTypeMailhog filesystemType = "mailhog"
)

func parseFilesystemType(filesystemType string) (filesystemType, error) {
	switch filesystemType {
	case "eml":
		return FileStorageTypeEML, nil
	case "mailhog":
		return FileStorageTypeMailhog, nil
	default:
		return "", fmt.Errorf("unknown filesystem type: %v", filesystemType)
	}
}

func (t filesystemType) GetFileSuffix() string {
	switch t {
	case FileStorageTypeEML:
		return ".eml"
	case FileStorageTypeMailhog:
		return "@mailhog.example"
	}
	panic("unknown filesystem type")
}

// FilesystemStorage is a storage engine that stores emails on the filesystem.
type filesystemStorage struct {
	folder         string
	filesystemType filesystemType
}

// FilesystemStorage implements the Storage interface
var _ Storage = &filesystemStorage{}

func newFilesystemStorage(folder string, filesystemTypeStr string) (*filesystemStorage, error) {
	log.Logf(log.INFO, "using storage in folder %v (type=%v)", folder, filesystemTypeStr)
	filesystemType, err := parseFilesystemType(filesystemTypeStr)
	if err != nil {
		return nil, err
	}
	return &filesystemStorage{folder: folder, filesystemType: filesystemType}, nil
}

// DeleteAllEmails implements Storage.
func (s *filesystemStorage) DeleteAllEmails() error {
	// list all files in the folder
	emailIDs, err := s.getAllEmailIDs()
	if err != nil {
		return err
	}
	// delete all emails
	var errors []error
	for _, emailID := range emailIDs {
		err := s.DeleteEmailByID(emailID)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("errors: %v", errors)
	}
	return nil
}

// DeleteEmailByID implements Storage.
func (s *filesystemStorage) DeleteEmailByID(emailID string) error {
	filePath := s.getEmailFilename(emailID)
	log.Logf(log.DEBUG, "deleting file %v", filePath)
	return os.Remove(filePath)
}

// getEmailFilename returns the filename of the email.
func (s *filesystemStorage) getEmailFilename(emailID string) string {
	return filepath.Join(s.folder, emailID+s.filesystemType.GetFileSuffix())
}

// GetAttachment implements Storage.
func (s *filesystemStorage) GetAttachment(emailID string, attachmentID string) (Attachment, error) {
	mp, err := s.loadEmailFromID(emailID)
	if err != nil {
		return Attachment{}, err
	}
	log.Logf(log.DEBUG, "searching for attachment %v", attachmentID)
	attachmentNode, found := mp.GetAttachment(attachmentID)
	if !found {
		return Attachment{}, fmt.Errorf("attachment not found: %v", attachmentID)
	}
	attachment := Attachment{
		AttachmentHeader: AttachmentHeader{
			ID:          attachmentID,
			ContentType: attachmentNode.GetContentType(),
			Filename:    attachmentNode.GetFilename(),
			Size:        attachmentNode.GetSize(),
		},
		Data: []byte(attachmentNode.GetDecodedBody()),
	}
	log.Logf(log.DEBUG, "found attachment %v", attachment)
	return attachment, nil
}

// GetAttachments implements Storage.
func (s *filesystemStorage) GetAttachments(emailID string) ([]AttachmentHeader, error) {
	mp, err := s.loadEmailFromID(emailID)
	if err != nil {
		return nil, err
	}
	var attachmentHeaders []AttachmentHeader
	for attachmentID, leaf := range mp.GetAttachments() {
		attachmentHeaders = append(attachmentHeaders, AttachmentHeader{
			ID:          attachmentID,
			ContentType: leaf.GetContentType(),
			Filename:    leaf.GetFilename(),
			Size:        leaf.GetSize(),
		})
	}
	return attachmentHeaders, nil
}

// GetBodyVersion implements Storage.
func (s *filesystemStorage) GetBodyVersion(emailID string, version EmailVersionType) (string, error) {
	if version == EmailVersionRaw {
		// return the raw version of the email
		rawBody, err := s.getRawBody(emailID)
		if err != nil {
			return "", err
		}
		return string(rawBody), nil
	}
	mp, err := s.loadEmailFromID(emailID)
	if err != nil {
		return "", err
	}
	switch version {
	case EmailVersionHtml:
		return mp.GetBody("html")
	case EmailVersionPlainText:
		return mp.GetBody("plain-text")
	case EmailVersionWatchHtml:
		return mp.GetBody("watch-html")
	default:
		return "", fmt.Errorf("unknown email version: %v", version)
	}
}

// GetEmailByID implements Storage.
func (s *filesystemStorage) GetEmailByID(emailID string) (EmailHeader, error) {
	multipart, err := s.loadEmailFromID(emailID)
	if err != nil {
		return EmailHeader{}, err
	}
	// create the email header
	return newEmailHeaderFromMultiPart(emailID, multipart), nil
}

// GetMailboxes implements Storage.
func (s *filesystemStorage) GetMailboxes() ([]Mailbox, error) {
	// list all files in the folder
	emailIDs, err := s.getAllEmailIDs()
	if err != nil {
		return nil, err
	}
	// extract the recipients from the emails
	recipents := make(map[string]bool)
	for _, emailID := range emailIDs {
		multipart, err := s.loadEmailFromID(emailID)
		if err != nil {
			return nil, err
		}
		for _, address := range multipart.GetRecipients() {
			recipents[address.Address] = true
		}
	}
	// create the mailboxes
	mailboxes := make([]Mailbox, 0, len(recipents))
	for address := range recipents {
		mailboxes = append(mailboxes, Mailbox{Name: address})
	}
	sort.Slice(mailboxes, func(i, j int) bool {
		return mailboxes[i].Name < mailboxes[j].Name
	})
	return mailboxes, nil
}

// SearchEmails implements Storage.
func (s *filesystemStorage) SearchEmails(query string, page int, pageSize int) ([]EmailHeader, int, error) {
	// Parse the query string
	matchers, err := matcher.ParseQuery(query)
	if err != nil {
		return nil, 0, err
	}

	// list all files in the folder
	emailIDs, err := s.getAllEmailIDs()
	if err != nil {
		return nil, 0, err
	}
	var emailHeaders []EmailHeader
	for _, emailID := range emailIDs {
		multipart, err := s.loadEmailFromID(emailID)
		if err != nil {
			return nil, 0, err
		}
		if multipart.MatchAll(matchers) {
			emailHeaders = append(emailHeaders, newEmailHeaderFromMultiPart(emailID, multipart))
		}
	}

	// sort email headers by date
	sort.Slice(emailHeaders, func(i, j int) bool {
		return emailHeaders[i].Date.After(emailHeaders[j].Date)
	})

	totalMatches := len(emailHeaders)

	// do the pagination
	if page < 1 {
		return nil, 0, fmt.Errorf("invalid page number: %v", page)
	}
	start := (page - 1) * pageSize
	var end int
	if pageSize < 0 {
		// return all emails
		end = len(emailHeaders)
	} else {
		end = start + pageSize
	}
	if start > len(emailHeaders) {
		start = len(emailHeaders)
	}
	if end > len(emailHeaders) {
		end = len(emailHeaders)
	}
	return emailHeaders[start:end], totalMatches, nil
}

// Load loads the storage based on the root storage
func (s *filesystemStorage) load(rootStorage Storage) error {
	// check that the folder exists
	if _, err := os.Stat(s.folder); os.IsNotExist(err) {
		// create the folder
		log.Logf(log.INFO, "creating folder %v", s.folder)
		if err := os.MkdirAll(s.folder, 0755); err != nil {
			return err
		}
	}
	// we don't load anything, all calls are direct to the filesystem
	return nil
}

// setWithID inserts a new email into the storage.
func (s *filesystemStorage) setWithID(emailID string, message *mail.Message) error {
	log.Logf(log.INFO, "saving email %v", emailID)
	// create the file
	emailFilename := s.getEmailFilename(emailID)
	file, err := os.Create(emailFilename)
	if err != nil {
		return err
	}
	defer file.Close()
	switch s.filesystemType {
	case FileStorageTypeEML:
		// nothing to do
	case FileStorageTypeMailhog:
		// write a fake mailhog header
		var header string
		header += "HELO:<fake-server>\n"
		header += "FROM:<fake-sender@exmaple.com>\n"
		header += "TO:<fake-recipient@example.com\n"
		header += "\n"
		if _, err := file.WriteString(header); err != nil {
			return err
		}
	}
	// write the email to the file
	writer := bufio.NewWriter(file)
	for key, values := range message.Header {
		for _, value := range values {
			if _, err := writer.WriteString(fmt.Sprintf("%v: %v\n", key, value)); err != nil {
				return err
			}
		}
	}
	if _, err = writer.WriteString("\n"); err != nil {
		return err
	}
	if _, err = writer.ReadFrom(message.Body); err != nil {
		return err
	}
	err = writer.Flush()
	if err != nil {
		return err
	}
	// check if we can parse the email
	_, err = s.loadEmailFromID(emailID)
	if err != nil {
		// delete the file
		os.Remove(emailFilename)
		return fmt.Errorf("cannot parse email %v: %v", emailID, err)
	}
	return nil
}

func (s *filesystemStorage) getRawBody(emailID string) ([]byte, error) {
	file, err := os.Open(s.getEmailFilename(emailID))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	switch s.filesystemType {
	case FileStorageTypeEML:
		// nothing to do
	case FileStorageTypeMailhog:
		err = skipMailhogHeader(file)
		if err != nil {
			return nil, err
		}
	}
	// read the complete file
	return io.ReadAll(file)
}

func (s *filesystemStorage) loadEmailFromID(emailID string) (*multipart.Multipart, error) {
	file, err := os.Open(s.getEmailFilename(emailID))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	switch s.filesystemType {
	case FileStorageTypeEML:
		// nothing to do
	case FileStorageTypeMailhog:
		err = skipMailhogHeader(file)
		if err != nil {
			return nil, err
		}
	}
	message, err := mail.ReadMessage(file)
	if err != nil {
		return nil, fmt.Errorf("cannot parse email %v: %v", emailID, err)
	}
	// parse multipart
	multipart, err := multipart.New(message)
	if err != nil {
		return nil, fmt.Errorf("cannot parse email %v: %v", emailID, err)
	}
	return multipart, nil
}

// readMailhogHeader reads the mailhog header from the file
// it reads until the first empty line
func skipMailhogHeader(file *os.File) error {
	reader := bufio.NewReader(file)
	var totalBytesRead int64
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		totalBytesRead += int64(len(line))
		if line == "\n" || line == "\r\n" {
			// set the file pointer to the start of the body
			_, err = file.Seek(totalBytesRead, io.SeekStart)
			if err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func newEmailHeaderFromMultiPart(ID string, multipart *multipart.Multipart) EmailHeader {
	return EmailHeader{
		ID:             ID,
		From:           newEmailAddressFromAddress(multipart.GetFrom()),
		Tos:            newEmailAddressesFromAddresses(multipart.GetTos()),
		CCs:            newEmailAddressesFromAddresses(multipart.GetCCs()),
		Subject:        multipart.GetSubject(),
		Date:           multipart.GetDate(),
		HasAttachments: multipart.HasAttachments(),
		Preview:        multipart.GetPreview(),
		BodyVersions:   append(multipart.GetBodyVersions(), "raw"),
	}
}

func newEmailAddressesFromAddresses(addresses []mail.Address) []EmailAddress {
	emailAddresses := make([]EmailAddress, 0, len(addresses))
	for _, address := range addresses {
		emailAddresses = append(emailAddresses, newEmailAddressFromAddress(address))
	}
	return emailAddresses
}

func newEmailAddressFromAddress(address mail.Address) EmailAddress {
	// parse the email address

	return EmailAddress{
		Name:    address.Name,
		Address: address.Address,
	}
}

func (s *filesystemStorage) getAllEmailIDs() ([]string, error) {
	// list all files in the folder
	fileSuffix := s.filesystemType.GetFileSuffix()
	filenames, err := filepath.Glob(filepath.Join(s.folder, "*"+fileSuffix))
	if err != nil {
		return nil, err
	}
	// extract the email IDs
	emailIDs := make([]string, 0, len(filenames))
	for _, filename := range filenames {
		// remove the folder and the extension
		emailID := filepath.Base(filename)
		emailID = emailID[:len(emailID)-len(fileSuffix)]
		emailIDs = append(emailIDs, emailID)
	}
	return emailIDs, nil
}
