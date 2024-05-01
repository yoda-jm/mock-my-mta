package multipart

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"time"
)

// WalkLeavesFunc is a function that is called for each leaf node
// in a multipart email.
// If the function returns false, the walk is stopped.
type WalkLeavesFunc func(leaf LeafNode) WalkStatus

// enum telling if the walk should continue or stop
type WalkStatus int

const (
	ContinueWalk = iota
	StopWalk
)

type node interface {
	GetHeaders() map[string][]string
	WalfLeaves(fn WalkLeavesFunc) WalkStatus
}

type Multipart struct {
	node
}

func New(message *mail.Message) (*Multipart, error) {
	node, err := parseMail(message.Header, message.Body)
	if err != nil {
		return nil, err
	}
	return &Multipart{node: node}, nil
}

func parseMail(headers map[string][]string, bodyReader io.Reader) (node, error) {
	// find media type and boundary
	contentType := getContentType(headers)
	mediaType := "text/plain"
	params := map[string]string{}
	if contentType != "" {
		var err error
		mediaType, params, err = mime.ParseMediaType(contentType)
		if err != nil {
			return nil, err
		}
	}
	if !strings.HasPrefix(mediaType, "multipart/") {
		// this is a leaf node (no parts)
		// read the body
		body, err := io.ReadAll(bodyReader)
		if err != nil {
			return nil, err
		}
		// create a leaf node
		return LeafNode{
			headers: headers,
			body:    body,
		}, nil
	}
	// this is a multipart email, create a multipart node
	var parts []node
	mr := multipart.NewReader(bodyReader, params["boundary"])
	// Iterate through the parts
	for {
		// Read the next part
		p, err := mr.NextPart()
		if err == io.EOF {
			// End of multipart message
			break
		}
		if err != nil {
			// unexpected error
			return nil, err
		}
		// Parse the part
		node, err := parseMail(p.Header, p)
		if err != nil {
			return nil, err
		}
		parts = append(parts, node)
		// Close the part
		err = p.Close()
		if err != nil {
			return nil, err
		}
	}
	return multipartNode{
		headers: headers,
		parts:   parts,
	}, nil
}

func (mp Multipart) String() string {
	return stringIndent(mp.node, "")
}

func (mp Multipart) GetFrom() mail.Address {
	return parseAddress(decodeHeader(getHeaderValue(mp, "From")))
}

func (mp Multipart) GetTos() []mail.Address {
	return parseAddresses(decodeHeader(getHeaderValue(mp, "To")))
}

func (mp Multipart) GetCCs() []mail.Address {
	return parseAddresses(decodeHeader(getHeaderValue(mp, "Cc")))
}

func (mp Multipart) GetRecipients() []mail.Address {
	recipients := append(mp.GetTos(), mp.GetCCs()...)
	return recipients
}

func (mp Multipart) GetSubject() string {
	return decodeHeader(getHeaderValue(mp, "Subject"))
}

func (mp Multipart) GetDate() time.Time {
	dateStr := getHeaderValue(mp, "Date")
	date, err := mail.ParseDate(dateStr)
	if err != nil {
		return time.Time{}
	}
	return date
}

func (mp Multipart) HasAttachments() bool {
	var hasAttachments bool
	mp.WalfLeaves(func(leaf LeafNode) WalkStatus {
		if strings.HasPrefix(getHeaderValue(leaf, "Content-Disposition"), "attachment") {
			hasAttachments = true
			// stop walking
			return StopWalk
		}
		// continue walking
		return ContinueWalk
	})
	return hasAttachments
}

func (mp Multipart) GetPreview() string {
	var preview string
	if leaf, ok := mp.node.(LeafNode); ok {
		if leaf.IsAttachment() {
			// return empty preview for pure attachment emails
			return ""
		}
		preview = leaf.GetDecodedBody()
	} else {
		mp.WalfLeaves(func(leaf LeafNode) WalkStatus {
			if leaf.IsAttachment() {
				// skip attachments and continue walking
				return ContinueWalk
			}
			if leaf.IsPlainText() {
				preview = leaf.GetDecodedBody()
				preview = string(leaf.body)
				return StopWalk
			}
			// continue walking
			return ContinueWalk
		})
		if preview == "" {
			// no plain text body found, use the html body
			mp.WalfLeaves(func(leaf LeafNode) WalkStatus {
				if leaf.IsAttachment() {
					// skip attachments and continue walking
					return ContinueWalk
				}
				if leaf.IsHTML() {
					preview = leaf.GetDecodedBody()
					return StopWalk
				}
				// continue walking
				return ContinueWalk
			})
		}
	}
	// limit preview to 100 characters
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}
	// remove \r and \n
	preview = strings.ReplaceAll(preview, "\r", "")
	preview = strings.ReplaceAll(preview, "\n", "")
	return preview
}

func (mp Multipart) GetBody(bodyVersion string) (string, error) {
	if leaf, ok := mp.node.(LeafNode); ok {
		if leaf.IsAttachment() {
			// return empty body for pure attachment emails
			return "", nil
		}
		return leaf.GetDecodedBody(), nil
	}
	var body strings.Builder
	mp.WalfLeaves(func(leaf LeafNode) WalkStatus {
		if bodyVersion == "plain-text" && leaf.IsPlainText() {
			body.WriteString(leaf.GetDecodedBody())
			return StopWalk
		}
		if bodyVersion == "html" && leaf.IsHTML() {
			body.WriteString(leaf.GetDecodedBody())
			return StopWalk
		}
		if bodyVersion == "watch-html" && leaf.IsWatchHTML() {
			body.WriteString(leaf.GetDecodedBody())
			return StopWalk
		}
		// continue walking
		return ContinueWalk
	})
	return body.String(), nil
}

func (mp Multipart) GetBodyVersions() []string {
	// if root is a leaf
	if leaf, ok := mp.node.(LeafNode); ok {
		if leaf.IsPlainText() {
			return []string{"raw", "plain-text"}
		} else if leaf.IsHTML() {
			return []string{"raw", "html"}
		} else if leaf.IsWatchHTML() {
			return []string{"raw", "watch-html"}
		}
		if leaf.IsAttachment() {
			// FIXME: return text-plain version that will be empty
			return []string{"raw"}
		}
		return []string{"raw", "plain-text"}
	}
	var bodyVersions []string
	bodyVersions = append(bodyVersions, "raw")
	mp.WalfLeaves(func(leaf LeafNode) WalkStatus {
		if leaf.IsPlainText() {
			bodyVersions = append(bodyVersions, "plain-text")
		} else if leaf.IsHTML() {
			bodyVersions = append(bodyVersions, "html")
		} else if leaf.IsWatchHTML() {
			bodyVersions = append(bodyVersions, "watch-html")
		}
		return ContinueWalk
	})
	return bodyVersions
}

func decodeHeader(header string) string {
	dec := new(mime.WordDecoder)
	decoded, err := dec.Decode(header)
	if err != nil {
		return header
	}
	return decoded
}

func parseAddresses(addresses string) []mail.Address {
	addrsPtr, err := mail.ParseAddressList(addresses)
	if err != nil {
		return []mail.Address{}
	}
	addrs := make([]mail.Address, len(addrsPtr))
	for i, addr := range addrsPtr {
		addrs[i] = *addr
	}
	return addrs

}

func parseAddress(address string) mail.Address {
	addr, err := mail.ParseAddress(address)
	if err != nil {
		return mail.Address{
			Name:    "",
			Address: address,
		}
	}
	return *addr
}

func stringIndent(n node, indent string) string {
	// switch on the node type
	switch node := n.(type) {
	case LeafNode:
		if node.IsAttachment() {
			return fmt.Sprintf("%vattachment node (Content-Type=%v, filename=%v, size=%v)\n", indent, getContentType(node.GetHeaders()), node.GetAttachmentFilename(), node.GetAttachmentSize())
		}
		return fmt.Sprintf("%vleaf node (Content-Type=%v)\n", indent, getContentType(node.GetHeaders()))
	case multipartNode:
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%vmultipart node (Content-Type=%v)\n", indent, getContentType(node.GetHeaders())))
		for _, part := range node.parts {
			sb.WriteString(stringIndent(part, indent+"  "))
		}
		return sb.String()
	}
	return ""
}

func getHeaderValues(n node, key string) []string {
	values, _ := n.GetHeaders()[key]
	return values
}

func getHeaderValue(n node, key string) string {
	values := getHeaderValues(n, key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func getContentType(headers map[string][]string) string {
	contentTypes, _ := headers["Content-Type"]
	if len(contentTypes) == 0 {
		return ""
	}
	return contentTypes[0]
}
