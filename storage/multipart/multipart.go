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
type walkLeavesFunc func(leaf leafNode) walkStatus

// enum telling if the walk should continue or stop
type walkStatus int

const (
	continueWalk = iota
	stopWalk
)

type node interface {
	getHeaders() map[string][]string
	walfLeaves(fn walkLeavesFunc) walkStatus
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
		return leafNode{
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
	mp.walfLeaves(func(leaf leafNode) walkStatus {
		if strings.HasPrefix(getHeaderValue(leaf, "Content-Disposition"), "attachment") {
			hasAttachments = true
			// stop walking
			return stopWalk
		}
		// continue walking
		return continueWalk
	})
	return hasAttachments
}

func (mp Multipart) GetAttachments() map[string]AttachmentNode {
	attachments := make(map[string]AttachmentNode)
	i := 0
	mp.walfLeaves(func(leaf leafNode) walkStatus {
		if !leaf.isAttachment() {
			// skip non-attachment leaves
			return continueWalk
		}
		attachments[fmt.Sprintf("%v", i)] = AttachmentNode{leafNode: leaf}
		i++
		// continue walking
		return continueWalk
	})
	return attachments
}

func (mp Multipart) GetAttachment(attachmentID string) (AttachmentNode, bool) {
	var attachment AttachmentNode
	id := 0
	found := false
	mp.walfLeaves(func(leaf leafNode) walkStatus {
		if !leaf.isAttachment() {
			// skip non-attachment nodes and continue
			return continueWalk
		}
		idStr := fmt.Sprintf("%v", id)
		if idStr != attachmentID {
			// increment the ID and continue walking
			id++
			return continueWalk
		}
		// found the attachment
		found = true
		attachment = AttachmentNode{leafNode: leaf}
		// stop walking
		return stopWalk
	})
	return attachment, found
}

func (mp Multipart) GetPreview() string {
	var preview string
	if leaf, ok := mp.node.(leafNode); ok {
		if leaf.isAttachment() {
			// return empty preview for pure attachment emails
			return ""
		}
		preview = leaf.GetDecodedBody()
	} else {
		mp.walfLeaves(func(leaf leafNode) walkStatus {
			if leaf.isAttachment() {
				// skip attachments and continue walking
				return continueWalk
			}
			if leaf.isPlainText() {
				preview = leaf.GetDecodedBody()
				preview = string(leaf.body)
				return stopWalk
			}
			// continue walking
			return continueWalk
		})
		if preview == "" {
			// no plain text body found, use the html body
			mp.walfLeaves(func(leaf leafNode) walkStatus {
				if leaf.isAttachment() {
					// skip attachments and continue walking
					return continueWalk
				}
				if leaf.isHTML() {
					preview = leaf.GetDecodedBody()
					return stopWalk
				}
				// continue walking
				return continueWalk
			})
		}
	}
	// limit preview to 100 characters
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}
	// remove \r and \n
	preview = strings.ReplaceAll(preview, "\r", "")
	preview = strings.ReplaceAll(preview, "\n", " ")
	return preview
}

func (mp Multipart) GetBody(bodyVersion string) (string, error) {
	if leaf, ok := mp.node.(leafNode); ok {
		if leaf.isAttachment() {
			// return empty body for pure attachment emails
			return "", nil
		}
		return leaf.GetDecodedBody(), nil
	}
	var body strings.Builder
	mp.walfLeaves(func(leaf leafNode) walkStatus {
		if bodyVersion == "plain-text" && leaf.isPlainText() {
			body.WriteString(leaf.GetDecodedBody())
			return stopWalk
		}
		if bodyVersion == "html" && leaf.isHTML() {
			body.WriteString(leaf.GetDecodedBody())
			return stopWalk
		}
		if bodyVersion == "watch-html" && leaf.isWatchHTML() {
			body.WriteString(leaf.GetDecodedBody())
			return stopWalk
		}
		// continue walking
		return continueWalk
	})
	return body.String(), nil
}

func (mp Multipart) GetBodyVersions() []string {
	// if root is a leaf
	if leaf, ok := mp.node.(leafNode); ok {
		if leaf.isPlainText() {
			return []string{"plain-text"}
		} else if leaf.isHTML() {
			return []string{"html"}
		} else if leaf.isWatchHTML() {
			return []string{"watch-html"}
		}
		if leaf.isAttachment() {
			// FIXME: return text-plain version that will be empty
			return []string{}
		}
		return []string{"plain-text"}
	}
	var bodyVersions []string
	mp.walfLeaves(func(leaf leafNode) walkStatus {
		if leaf.isPlainText() {
			bodyVersions = append(bodyVersions, "plain-text")
		} else if leaf.isHTML() {
			bodyVersions = append(bodyVersions, "html")
		} else if leaf.isWatchHTML() {
			bodyVersions = append(bodyVersions, "watch-html")
		}
		return continueWalk
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
	case leafNode:
		if node.isAttachment() {
			attachmentNode := AttachmentNode{leafNode: node}
			return fmt.Sprintf("%vattachment node (Content-Type=%v, filename=%v, size=%v)\n", indent, getContentType(node.getHeaders()), attachmentNode.GetFilename(), attachmentNode.GetSize())
		}
		return fmt.Sprintf("%vleaf node (Content-Type=%v)\n", indent, getContentType(node.getHeaders()))
	case multipartNode:
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%vmultipart node (Content-Type=%v)\n", indent, getContentType(node.getHeaders())))
		for _, part := range node.parts {
			sb.WriteString(stringIndent(part, indent+"  "))
		}
		return sb.String()
	}
	return ""
}

func getHeaderValues(n node, key string) []string {
	values, _ := n.getHeaders()[key]
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
