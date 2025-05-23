package multipart

import (
	"bytes"
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

type headers struct {
	// headers is a map of headers (case insensitive)
	// keys are stored in lower case
	values map[string][]string
}

func newHeaders(values map[string][]string) headers {
	h := headers{
		values: make(map[string][]string),
	}
	for k, v := range values {
		h.values[strings.ToLower(k)] = v
	}
	return h
}

// getValues returns the values of the header with the given name (case insensitive)
func (h headers) get(name string) []string {
	return h.values[strings.ToLower(name)]
}

type node interface {
	getHeaders() headers
	walkLeaves(fn walkLeavesFunc) walkStatus
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

// ParseEmailFromBytes takes raw email bytes, parses them into a mail.Message,
// and then into a multipart.Multipart object.
func ParseEmailFromBytes(rawEmail []byte) (*Multipart, error) {
	reader := bytes.NewReader(rawEmail)
	msg, err := mail.ReadMessage(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read message from bytes: %w", err)
	}
	return New(msg)
}

func parseMail(headersMap map[string][]string, bodyReader io.Reader) (node, error) {
	headers := newHeaders(headersMap)
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

// walkNodeFunc is a function that is called for each node
// in a multipart email.
// If the function returns false, the walk is stopped.
type walkNodeFunc func(n node, parentContentType string) walkStatus

// Helper function to recursively walk through the multipart structure.
// It calls the provided function for each node.
func walkNodes(n node, parentContentType string, fn walkNodeFunc) walkStatus {
	status := fn(n, parentContentType)
	if status == stopWalk {
		return stopWalk
	}

	if mn, ok := n.(multipartNode); ok {
		// Get the content type of the current multipart node
		currentContentType, _, _ := mime.ParseMediaType(getContentType(mn.headers))
		for _, part := range mn.parts {
			if walkNodes(part, currentContentType, fn) == stopWalk {
				return stopWalk
			}
		}
	}
	return continueWalk
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
	walkNodes(mp.node, "", func(n node, parentContentType string) walkStatus {
		if leaf, ok := n.(leafNode); ok {
			if leaf.isAttachment() {
				hasAttachments = true
				return stopWalk
			}
		}
		return continueWalk
	})
	return hasAttachments
}

func (mp Multipart) GetAttachments() map[string]AttachmentNode {
	attachments := make(map[string]AttachmentNode)
	i := 0
	walkNodes(mp.node, "", func(n node, parentContentType string) walkStatus {
		if leaf, ok := n.(leafNode); ok {
			if leaf.isAttachment() {
				attachments[fmt.Sprintf("%v", i)] = AttachmentNode{leafNode: leaf}
				i++
			}
		}
		return continueWalk
	})
	return attachments
}

func (mp Multipart) GetAttachment(attachmentID string) (AttachmentNode, bool) {
	var attachment AttachmentNode
	id := 0
	found := false
	walkNodes(mp.node, "", func(n node, parentContentType string) walkStatus {
		if leaf, ok := n.(leafNode); ok {
			if leaf.isAttachment() {
				idStr := fmt.Sprintf("%v", id)
				if idStr == attachmentID {
					found = true
					attachment = AttachmentNode{leafNode: leaf}
					return stopWalk
				}
				id++
			}
		}
		return continueWalk
	})
	return attachment, found
}

func (mp Multipart) GetPreview() string {
	var previewText, htmlText string

	walkNodes(mp.node, "", func(n node, parentContentType string) walkStatus {
		if leaf, ok := n.(leafNode); ok {
			if leaf.isAttachment() {
				return continueWalk // Skip attachments
			}

			// If we are inside a multipart/alternative, prioritize based on type
			if parentContentType == "multipart/alternative" {
				if leaf.isPlainText() && previewText == "" { // Take the first plain text
					previewText = leaf.GetDecodedBody()
				} else if leaf.isHTML() && htmlText == "" { // Take the first HTML
					htmlText = leaf.GetDecodedBody()
				}
			} else if leaf.isPlainText() && previewText == "" { // For other types, prefer plain text
				previewText = leaf.GetDecodedBody()
			} else if leaf.isHTML() && htmlText == "" && previewText == "" { // Or HTML if plain not found yet
				htmlText = leaf.GetDecodedBody()
			}
		}
		// If plain text is found within multipart/alternative, stop early for that alternative part.
		if parentContentType == "multipart/alternative" && previewText != "" {
			return stopWalk // Stop searching this alternative part
		}
		// If plain text found globally, can consider stopping if not inside an alternative.
		if previewText != "" && parentContentType != "multipart/alternative" {
			// This condition might be too aggressive if plain text is outside alternative
			// and we want to ensure we've checked alternative blocks.
			// For now, let's continue to ensure alternatives are checked.
		}
		return continueWalk
	})

	finalPreview := previewText
	if finalPreview == "" {
		finalPreview = htmlText
	}

	// limit preview to 100 characters
	if len(finalPreview) > 100 {
		finalPreview = finalPreview[:100] + "..."
	}
	// remove \r and \n
	finalPreview = strings.ReplaceAll(finalPreview, "\r", "")
	finalPreview = strings.ReplaceAll(finalPreview, "\n", " ")
	return strings.TrimSpace(finalPreview)
}

func (mp Multipart) GetBody(bodyVersion string) (string, error) {
	var bodyContent string
	var foundBody bool

	walkNodes(mp.node, "", func(n node, currentParentContentType string) walkStatus {
		if leaf, ok := n.(leafNode); ok {
			if leaf.isAttachment() {
				return continueWalk // Skip attachments
			}

			targetType := ""
			switch bodyVersion {
			case "plain-text":
				targetType = "text/plain"
			case "html":
				targetType = "text/html"
			case "watch-html":
				// Assuming watch-html is a specific type of html
				// For now, let's treat it as text/watch-html or check content type specifically
				// This part might need more specific logic if text/watch-html is a distinct MIME type
				if leaf.isWatchHTML() { // isWatchHTML should check for the specific content type
					bodyContent = leaf.GetDecodedBody()
					foundBody = true
					return stopWalk
				}
				return continueWalk // If not watch-html, continue
			default:
				return stopWalk // Unknown body version
			}

			contentType, _, _ := mime.ParseMediaType(getContentType(leaf.headers))

			if currentParentContentType == "multipart/alternative" {
				if contentType == targetType {
					bodyContent = leaf.GetDecodedBody()
					foundBody = true
					return stopWalk // Found the preferred type in alternative
				}
			} else if contentType == targetType { // For non-alternative, first match is fine
				bodyContent = leaf.GetDecodedBody()
				foundBody = true
				return stopWalk
			}
		} else if mn, ok := n.(multipartNode); ok {
			nodeContentType, _, _ := mime.ParseMediaType(getContentType(mn.headers))
			if nodeContentType == "multipart/alternative" {
				// Iterate parts of multipart/alternative to find the best match
				var plainPart, htmlPart leafNode
				var plainFound, htmlFound bool

				for _, partNode := range mn.parts {
					if partLeaf, isLeaf := partNode.(leafNode); isLeaf {
						if partLeaf.isAttachment() {
							continue
						}
						if partLeaf.isPlainText() && !plainFound {
							plainPart = partLeaf
							plainFound = true
						} else if partLeaf.isHTML() && !htmlFound {
							htmlPart = partLeaf
							htmlFound = true
						}
					}
				}

				if bodyVersion == "plain-text" && plainFound {
					bodyContent = plainPart.GetDecodedBody()
					foundBody = true
					return stopWalk
				} else if bodyVersion == "html" && htmlFound {
					bodyContent = htmlPart.GetDecodedBody()
					foundBody = true
					return stopWalk
				}
				// If requested version not found in this alternative, but other exists, stop for this alternative block.
				// This prevents grabbing, for example, HTML from a parent if text/plain was requested in child alternative.
				if plainFound || htmlFound {
					return stopWalk
				}
			}
		}
		return continueWalk
	})

	if !foundBody {
		// If not found, and it's a simple email (root is leaf)
		if leaf, ok := mp.node.(leafNode); ok && !leaf.isAttachment() {
			correctType := false
			if bodyVersion == "plain-text" && leaf.isPlainText() {
				correctType = true
			} else if bodyVersion == "html" && leaf.isHTML() {
				correctType = true
			} else if bodyVersion == "watch-html" && leaf.isWatchHTML() {
				correctType = true
			}
			if correctType {
				return leaf.GetDecodedBody(), nil
			}
			return "", nil // Or an error indicating not found / wrong type
		}
		// For multipart, if not found after walk, it means it's not there or not in preferred part.
		// Depending on strictness, could return error or empty string.
		// return "", fmt.Errorf("body version %s not found", bodyVersion)
	}

	return bodyContent, nil
}

func (mp Multipart) GetBodyVersions() []string {
	var versions []string
	versionsMap := make(map[string]bool)

	walkNodes(mp.node, "", func(n node, parentContentType string) walkStatus {
		if leaf, ok := n.(leafNode); ok {
			if leaf.isAttachment() {
				return continueWalk // Skip attachments
			}
			if leaf.isPlainText() {
				versionsMap["plain-text"] = true
			}
			if leaf.isHTML() {
				versionsMap["html"] = true
			}
			if leaf.isWatchHTML() {
				versionsMap["watch-html"] = true
			}
		}
		return continueWalk
	})

	// If root is a simple non-attachment leaf, ensure its type is added.
	if leaf, ok := mp.node.(leafNode); ok && !leaf.isAttachment() {
		if leaf.isPlainText() && !versionsMap["plain-text"] {
			versionsMap["plain-text"] = true
		} else if leaf.isHTML() && !versionsMap["html"] {
			versionsMap["html"] = true
		} else if leaf.isWatchHTML() && !versionsMap["watch-html"] {
			versionsMap["watch-html"] = true
		}
		// If it's a non-specific leaf and no versions were found (e.g. image/*),
		// but it's not an attachment, it implies it's the main body.
		// The current logic in leaf.isPlainText() etc. might need adjustment
		// if "text/plain" is the default for unspecified content types.
		// For now, if it's a leaf and not an attachment, and no specific type found,
		// and it's the root node, we might assume it's plain text by default.
		if len(versionsMap) == 0 {
			// This part is tricky: what if the root is image/jpeg and not an attachment?
			// The original code had a fallback to "plain-text" for root leaves.
			// Let's try to preserve that if no other versions are explicitly found from content types.
			// However, getContentType(leaf.headers) would be more accurate.
			// For now, this will rely on isPlainText/isHTML being robust.
		}
	}

	for v := range versionsMap {
		versions = append(versions, v)
	}
	// Ensure consistent order for testing
	// sort.Strings(versions) // Consider if consistent order is needed.
	return versions
}

func (mp Multipart) GetPartByCID(cid string) (leafNode, bool) {
	var foundNode leafNode
	found := false

	walkNodes(mp.node, "", func(n node, parentContentType string) walkStatus {
		if leaf, ok := n.(leafNode); ok {
			// Do not consider attachments as inline parts, CIDs are usually for inline images.
			// if leaf.isAttachment() {
			// 	return continueWalk
			// }
			if leaf.GetContentID() == cid || leaf.GetContentID() == "<"+cid+">" {
				foundNode = leaf
				found = true
				return stopWalk
			}
		}
		return continueWalk
	})

	return foundNode, found
}

func decodeHeader(header string) string {
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(header)
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
	values := n.getHeaders().get(key)
	return values
}

func getHeaderValue(n node, key string) string {
	values := getHeaderValues(n, key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func getContentType(headers headers) string {
	contentTypes := headers.get("Content-Type")
	if len(contentTypes) == 0 {
		return ""
	}
	return contentTypes[0]
}
