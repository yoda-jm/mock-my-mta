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

// FilesystemStorage is a storage engine that stores emails on the filesystem.
type filesystemStorage struct {
	folder string
}

// FilesystemStorage implements the Storage interface
var _ Storage = &filesystemStorage{}

func newFilesystemStorage(folder string) (*filesystemStorage, error) {
	log.Logf(log.INFO, "using storage in folder %v", folder)
	return &filesystemStorage{folder: folder}, nil
}

// DeleteEmailByID implements Storage.
func (s *filesystemStorage) DeleteEmailByID(emailID string) error {
	filePath := filepath.Join(s.folder, emailID+".eml")
	log.Logf(log.DEBUG, "deleting file %v", filePath)
	return os.Remove(filePath)
}

// GetAttachment implements Storage.
func (s *filesystemStorage) GetAttachment(emailID string, attachmentID string) (Attachment, error) {
	mp, err := loadEmailFromID(s.folder, emailID)
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
	mp, err := loadEmailFromID(s.folder, emailID)
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
		rawBody, err := getRawBody(s.folder, emailID)
		if err != nil {
			return "", err
		}
		return string(rawBody), nil
	}
	mp, err := loadEmailFromID(s.folder, emailID)
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
	multipart, err := loadEmailFromID(s.folder, emailID)
	if err != nil {
		return EmailHeader{}, err
	}
	// create the email header
	return newEmailHeaderFromMultiPart(emailID, multipart), nil
}

// GetMailboxes implements Storage.
func (s *filesystemStorage) GetMailboxes() ([]Mailbox, error) {
	// list all files in the folder
	emailIDs, err := getAllEmailIDs(s.folder)
	if err != nil {
		return nil, err
	}
	// extract the recipients from the emails
	recipents := make(map[string]bool)
	for _, emailID := range emailIDs {
		multipart, err := loadEmailFromID(s.folder, emailID)
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
	emailIDs, err := getAllEmailIDs(s.folder)
	if err != nil {
		return nil, 0, err
	}
	var emailHeaders []EmailHeader
	for _, emailID := range emailIDs {
		multipart, err := loadEmailFromID(s.folder, emailID)
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
	file, err := os.Create(filepath.Join(s.folder, emailID+".eml"))
	if err != nil {
		return err
	}
	defer file.Close()

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
	_, err = loadEmailFromID(s.folder, emailID)
	if err != nil {
		// delete the file
		os.Remove(filepath.Join(s.folder, emailID+".eml"))
		return fmt.Errorf("cannot parse email %v: %v", emailID, err)
	}
	return nil
}

func getRawBody(folder, emailID string) ([]byte, error) {
	file, err := os.Open(filepath.Join(folder, emailID+".eml"))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	// read the complete file
	return io.ReadAll(file)
}

func loadEmailFromID(folder, emailID string) (*multipart.Multipart, error) {
	file, err := os.Open(filepath.Join(folder, emailID+".eml"))
	if err != nil {
		return nil, err
	}
	defer file.Close()
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

func getAllEmailIDs(folder string) ([]string, error) {
	// list all files in the folder
	filenames, err := filepath.Glob(filepath.Join(folder, "*.eml"))
	if err != nil {
		return nil, err
	}
	// extract the email IDs
	emailIDs := make([]string, 0, len(filenames))
	for _, filename := range filenames {
		// remove the folder and the extension
		emailID := filepath.Base(filename)
		emailID = emailID[:len(emailID)-4]
		emailIDs = append(emailIDs, emailID)
	}
	return emailIDs, nil
}
