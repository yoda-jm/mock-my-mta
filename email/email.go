package email

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"strings"

	"github.com/google/uuid"

	"mock-my-mta/log"
)

type Attachment struct {
	id        uuid.UUID
	mediaType string
	filename  string
	content   []byte
}

func (a Attachment) GetID() uuid.UUID     { return a.id }
func (a Attachment) GetMediaType() string { return a.mediaType }
func (a Attachment) GetFilename() string  { return a.filename }
func (a Attachment) GetContent() []byte   { return a.content }

// Email represents an email message.
type Email struct {
	rawMessage []byte // just used for serialization

	header        mail.Header
	bodyTxt       []byte
	bodyHtml      []byte
	bodyWatchHtml []byte
	versions      []EmailVersionTypeEnum
	attachments   map[uuid.UUID]*Attachment
}

// NewEmail creates a new Email instance with the given subject, body, sender, and recipients.
func Parse(rawMessage []byte) (*Email, error) {
	email := &Email{}
	if err := email.reset(rawMessage); err != nil {
		return nil, err
	}
	return email, nil
}

// GetSubject returns the subject of the email.
func (e *Email) GetSubject() string {
	return e.header.Get("Subject")
}

// SortFieldEnum represents the available fields for sorting.
type EmailVersionTypeEnum int

// Enum values for SortFieldEnum.
const (
	EmailVersionRaw EmailVersionTypeEnum = iota
	EmailVersionTxt
	EmailVersionHtml
	EmailVersionWatchHtml
)

func ParseEmailVersionTypeEnum(str string) (EmailVersionTypeEnum, error) {
	switch str {
	case "raw":
		return EmailVersionRaw, nil
	case "txt":
		return EmailVersionTxt, nil
	case "html":
		return EmailVersionHtml, nil
	case "watch-html":
		return EmailVersionWatchHtml, nil
	default:
		return EmailVersionRaw, fmt.Errorf("cannot parse email version type %q", str)
	}
}

func (evt EmailVersionTypeEnum) String() string {
	switch evt {
	case EmailVersionRaw:
		return "raw"
	case EmailVersionTxt:
		return "txt"
	case EmailVersionHtml:
		return "html"
	case EmailVersionWatchHtml:
		return "watch-html"
	}
	panic("unsupported email version type")
}

func (e *Email) GetVersions() []EmailVersionTypeEnum {
	return e.versions
}

// GetBody returns the body of the email.
func (e *Email) GetBody(emailVersion EmailVersionTypeEnum) (string, error) {
	switch emailVersion {
	case EmailVersionRaw:
		return string(e.rawMessage), nil
	case EmailVersionTxt:
		return string(e.bodyTxt), nil
	case EmailVersionHtml:
		return string(e.bodyHtml), nil
	case EmailVersionWatchHtml:
		return string(e.bodyWatchHtml), nil
	}
	panic("unsupported email version type")
}

func (e *Email) GetAttachments() []uuid.UUID {
	var uuids []uuid.UUID
	for id := range e.attachments {
		uuids = append(uuids, id)
	}
	return uuids
}

func (e *Email) GetAttachment(id uuid.UUID) (*Attachment, bool) {
	attachment, ok := e.attachments[id]
	return attachment, ok
}

// GetSender returns the sender of the email.
func (e *Email) GetSender() string {
	fromAddresses, err := e.header.AddressList("From")
	if err != nil {
		return ""
	}
	if len(fromAddresses) > 0 {
		return fromAddresses[0].Address
	}
	return ""
}

// GetRecipients returns the recipients of the email.
func (e *Email) GetRecipients() []string {
	toAddresses, err := e.header.AddressList("To")
	if err != nil {
		return nil
	}

	var tos []string
	for _, toAddress := range toAddresses {
		tos = append(tos, toAddress.Address)
	}
	return tos
}

type emailJSON struct {
	RawMessage string `json:"raw_message"`
}

func (e Email) MarshalJSON() ([]byte, error) {
	email := emailJSON{
		RawMessage: string(e.rawMessage),
	}
	return json.Marshal(email)
}

func (e *Email) UnmarshalJSON(data []byte) error {
	var email emailJSON
	if err := json.Unmarshal(data, &email); err != nil {
		return err
	}
	return e.reset([]byte(email.RawMessage))
}

func (e *Email) reset(rawMessage []byte) error {
	*e = Email{
		rawMessage:  rawMessage,
		attachments: make(map[uuid.UUID]*Attachment),
	}
	message, err := mail.ReadMessage(bytes.NewReader(e.rawMessage))
	if err != nil {
		return err
	}
	e.header = message.Header
	body, err := io.ReadAll(message.Body)
	if err != nil {
		return err
	}
	e.versions = []EmailVersionTypeEnum{
		EmailVersionRaw,
	}

	// check for multipart
	err = parseMultipart(0, body, e.header, e)
	if err != nil {
		log.Logf(log.WARNING, "cannot read multipart: %v", err)
	}
	return nil
}

type header interface {
	Get(string) string
}

func parseMultipart(depth uint64, body []byte, header header, email *Email) error {
	contentType := header.Get("Content-Type")
	if contentType == "" {
		// default content type should be plain text
		contentType = "text/plain"
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return err
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		log.Logf(log.DEBUG, "detected multipart (depth=%v, type=%v, boundary=%v)", depth, mediaType, boundary)
		parts, err := readMultipart(body, boundary)
		if err != nil {
			return err
		}
		for _, part := range parts {
			log.Logf(log.DEBUG, "found part (%v bytes)", len(string(part.body)))
			err = parseMultipart(depth+1, part.body, part.header, email)
			if err != nil {
				return err
			}
		}
	} else if mediaType == "text/plain" {
		log.Logf(log.DEBUG, "detected email text part (depth=%v, type=%v, length=%v)", depth, mediaType, len(body))
		email.bodyTxt = body
		email.versions = append(email.versions, EmailVersionTxt)
	} else if mediaType == "text/html" {
		log.Logf(log.DEBUG, "detected email html part (depth=%v, type=%v, length=%v)", depth, mediaType, len(body))
		email.bodyHtml = body
		email.versions = append(email.versions, EmailVersionHtml)
	} else if mediaType == "text/watch-html" {
		log.Logf(log.DEBUG, "detected email watcg html part (depth=%v, type=%v, length=%v)", depth, mediaType, len(body))
		email.bodyWatchHtml = body
		email.versions = append(email.versions, EmailVersionWatchHtml)
	} else {
		attachment := &Attachment{
			id:        uuid.New(),
			mediaType: mediaType,
			content:   body,
		}
		contentDisposition := header.Get("Content-Disposition")
		if contentDisposition == "" {
			contentDisposition = "attachment"
		}
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err != nil {
			return err
		}
		if filename, found := params["filename"]; found {
			attachment.filename = filename
		}
		if header.Get("Content-Transfer-Encoding") == "base64" {
			// the content is base64 encoded, decode it into the content
			dst := make([]byte, base64.StdEncoding.DecodedLen(len(attachment.content)))
			n, err := base64.StdEncoding.Decode(dst, attachment.content)
			if err != nil {
				return err
			}
			attachment.content = dst[:n]
		}
		log.Logf(log.DEBUG, "detected attachment part (depth=%v, id=%v, type=%v, filename=%q, length=%v)", depth, attachment.id, attachment.mediaType, attachment.filename, len(attachment.content))
		email.attachments[attachment.id] = attachment
	}

	return nil
}

type multipartPart struct {
	header textproto.MIMEHeader
	body   []byte
}

func readMultipart(body []byte, boundary string) ([]multipartPart, error) {
	mr := multipart.NewReader(bytes.NewReader(body), boundary)
	var parts []multipartPart
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			return parts, nil
		}
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(p)
		if err != nil {
			return nil, err
		}
		parts = append(parts, multipartPart{header: p.Header, body: body})
	}
}
