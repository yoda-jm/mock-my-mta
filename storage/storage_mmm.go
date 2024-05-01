package storage

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"mock-my-mta/log"
	"mock-my-mta/storage/multipart"
)

// MMMStorage is a storage engine that stores emails on the filesystem.
type MMMStorage struct {
	folder string
}

// MMMStorage implements the Storage interface
var _ Storage = &MMMStorage{}

func newMMMStorage(folder string) (*MMMStorage, error) {
	log.Logf(log.INFO, "using storage in folder %v", folder)
	return &MMMStorage{folder: folder}, nil
}

// DeleteEmailByID implements Storage.
func (s *MMMStorage) DeleteEmailByID(emailID string) error {
	filePath := filepath.Join(s.folder, emailID+".eml")
	log.Logf(log.DEBUG, "deleting file %v", filePath)
	return os.Remove(filePath)
}

// GetAttachment implements Storage.
func (s *MMMStorage) GetAttachment(emailID string, attachmentID string) (Attachment, error) {
	mp, err := loadEmailFromID(s.folder, emailID)
	if err != nil {
		return Attachment{}, err
	}
	var attachment Attachment
	id := 0
	found := false
	var attachmentLeaf multipart.LeafNode
	log.Logf(log.DEBUG, "searching for attachment %v", attachmentID)
	mp.WalfLeaves(func(leaf multipart.LeafNode) multipart.WalkStatus {
		if !leaf.IsAttachment() {
			// skip non-attachment nodes and continue
			return multipart.ContinueWalk
		}
		idStr := fmt.Sprintf("%v", id)
		if idStr != attachmentID {
			// increment the ID and continue walking
			id++
			return multipart.ContinueWalk
		}
		found = true
		attachmentLeaf = leaf
		// stop walking
		return multipart.StopWalk

	})
	if !found {
		return Attachment{}, fmt.Errorf("attachment not found: %v", attachmentID)
	}

	data := attachmentLeaf.GetBody()
	if attachmentLeaf.GetContentTransferEncoding() == "base64" {
		// decode the base64 data
		decodedData := make([]byte, 2*len(data))
		n, err := base64.StdEncoding.Decode(decodedData, data)
		if err != nil {
			return Attachment{}, err
		}
		data = decodedData[:n]
	}
	attachment = Attachment{
		AttachmentHeader: AttachmentHeader{
			ID:          attachmentID,
			ContentType: attachmentLeaf.GetAttachmentContentType(),
			Filename:    attachmentLeaf.GetAttachmentFilename(),
			Size:        attachmentLeaf.GetAttachmentSize(),
		},
		Data: data,
	}
	log.Logf(log.DEBUG, "found attachment %v", attachment)
	return attachment, nil
}

// GetAttachments implements Storage.
func (s *MMMStorage) GetAttachments(emailID string) ([]AttachmentHeader, error) {
	mp, err := loadEmailFromID(s.folder, emailID)
	if err != nil {
		return nil, err
	}
	var attachmentHeaders []AttachmentHeader
	id := 0
	mp.WalfLeaves(func(leaf multipart.LeafNode) multipart.WalkStatus {
		if !leaf.IsAttachment() {
			// skip non-attachment nodes and continue
			return multipart.ContinueWalk
		}
		attachmentHeaders = append(attachmentHeaders, AttachmentHeader{
			ID:          fmt.Sprintf("%v", id),
			ContentType: leaf.GetAttachmentContentType(),
			Filename:    leaf.GetAttachmentFilename(),
			Size:        leaf.GetAttachmentSize(),
		})
		// increment the ID and continue walking
		id++
		return multipart.ContinueWalk
	})
	return attachmentHeaders, nil
}

// GetBodyVersion implements Storage.
func (s *MMMStorage) GetBodyVersion(emailID string, version EmailVersionType) (string, error) {
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
func (s *MMMStorage) GetEmailByID(emailID string) (EmailHeader, error) {
	multipart, err := loadEmailFromID(s.folder, emailID)
	if err != nil {
		return EmailHeader{}, err
	}
	// create the email header
	return newEmailHeaderFromMultiPart(emailID, multipart), nil
}

// GetMailboxes implements Storage.
func (s *MMMStorage) GetMailboxes() ([]Mailbox, error) {
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

// parseQuery parses the input string into a slice of key-value pairs and plain text elements.
func parseQuery(query string) ([]map[string]string, []string) {
	var keyValuePairs []map[string]string
	var plainTexts []string

	// Regex pattern to extract key:value pairs and quoted/non-quoted text
	pattern := `(\w+:[^\s"]+|"[^"]*"|\S+)`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllString(query, -1)

	for _, match := range matches {
		if strings.Contains(match, ":") && !strings.HasPrefix(match, "\"") {
			// Split the first occurrence of ':' to separate key and value
			split := strings.SplitN(match, ":", 2)
			key := split[0]
			value := split[1]

			// Check if the value is quoted and remove quotes if needed
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = strings.Trim(value, "\"")
			}

			keyValuePair := make(map[string]string)
			keyValuePair[key] = value
			keyValuePairs = append(keyValuePairs, keyValuePair)
		} else if strings.HasPrefix(match, "\"") && strings.HasSuffix(match, "\"") {
			// Remove the quotes for plain text matches
			plainTexts = append(plainTexts, strings.Trim(match, "\""))
		} else {
			plainTexts = append(plainTexts, match)
		}
	}

	return keyValuePairs, plainTexts
}

// SearchEmails implements Storage.
func (s *MMMStorage) SearchEmails(query string, page int, pageSize int) ([]EmailHeader, int, error) {
	// Parse the query string
	matchers, err := extractMatchers(query)
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
		matchAll := true
		for _, matcher := range matchers {
			if !matcher.Match(multipart) {
				matchAll = false
				break
			}
		}
		if matchAll {
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

// extract matchers from the query
func extractMatchers(query string) ([]multipart.MultipartMatcher, error) {
	const LAYOUT_DATE = "2006-01-02"
	keyValuePairs, plainTexts := parseQuery(query)

	matchers := make([]multipart.MultipartMatcher, 0)
	for _, keyValue := range keyValuePairs {
		for key, value := range keyValue {
			switch key {
			case "mailbox":
				// Search for emails in the specified mailbox
				log.Logf(log.DEBUG, "searching for mailbox %v", value)
				matchers = append(matchers, multipart.NewMailboxMatch(value))
			case "has":
				switch value {
				case "attachment":
					// Search for emails that have the specified attribute
					log.Logf(log.DEBUG, "searching for emails with attachments")
					matchers = append(matchers, multipart.NewAttachmentMatch())
				default:
					return nil, fmt.Errorf("unknown search attribute for 'has': %v", value)
				}
			case "before":
				// search for emails with date before
				valueDate, err := time.Parse(LAYOUT_DATE, value)
				if err != nil {
					return nil, fmt.Errorf("invalid date format: %v", value)
				}
				log.Logf(log.DEBUG, "searching for emails before %v", value)
				matchers = append(matchers, multipart.NewBeforeMatch(valueDate))
			case "after":
				// search for emails with date after
				valueDate, err := time.Parse(LAYOUT_DATE, value)
				if err != nil {
					return nil, fmt.Errorf("invalid date format: %v", value)
				}
				log.Logf(log.DEBUG, "searching for emails after %v", value)
				matchers = append(matchers, multipart.NewAfterMatch(valueDate))
			case "from":
				// search for emails from the specified address
				log.Logf(log.DEBUG, "searching for emails from %v", value)
				matchers = append(matchers, multipart.NewFromMatch(value))
			case "older_than":
				// search for emails older than the specified duration
				duration, err := time.ParseDuration(value)
				if err != nil {
					return nil, fmt.Errorf("invalid duration format: %v", value)
				}
				log.Logf(log.DEBUG, "searching for emails older than %v", duration)
				matchers = append(matchers, multipart.NewOlderThanMatch(duration))
			case "newer_than":
				// search for emails newer than the specified duration
				duration, err := time.ParseDuration(value)
				if err != nil {
					return nil, fmt.Errorf("invalid duration format: %v", value)
				}
				log.Logf(log.DEBUG, "searching for emails newer than %v", duration)
				matchers = append(matchers, multipart.NewNewerThanMatch(duration))
			case "subject":
				// search for emails with the specified word in the subject
				log.Logf(log.DEBUG, "searching for emails with subject %v", value)
				matchers = append(matchers, multipart.NewSubjectMatch(value))
			default:
				return nil, fmt.Errorf("unknown search key: %v", key)
			}
		}
	}

	for _, plainText := range plainTexts {
		if plainText == "" {
			continue
		}
		// Search for emails that contain the plain text
		log.Logf(log.DEBUG, "searching for plain text %q", plainText)
		matchers = append(matchers, multipart.NewPlainTextMatch(plainText))
	}
	return matchers, nil
}

// Load loads the storage based on the root storage
func (s *MMMStorage) load(rootStorage Storage) error {
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
func (s *MMMStorage) setWithID(emailID string, message *mail.Message) error {
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
		From:           NewEmailAddressFromAddress(multipart.GetFrom()),
		Tos:            NewEmailAddressesFromAddresses(multipart.GetTos()),
		CCs:            NewEmailAddressesFromAddresses(multipart.GetCCs()),
		Subject:        multipart.GetSubject(),
		Date:           multipart.GetDate(),
		HasAttachments: multipart.HasAttachments(),
		Preview:        multipart.GetPreview(),
		BodyVersions:   multipart.GetBodyVersions(),
	}
}

func NewEmailAddressesFromAddresses(addresses []mail.Address) []EmailAddress {
	emailAddresses := make([]EmailAddress, 0, len(addresses))
	for _, address := range addresses {
		emailAddresses = append(emailAddresses, NewEmailAddressFromAddress(address))
	}
	return emailAddresses
}

func NewEmailAddressFromAddress(address mail.Address) EmailAddress {
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
