package multipart

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime/quotedprintable"
	"strings"
)

type leafNode struct {
	headers headers
	body    []byte
}

// leaf node implements the node interface
var _ node = leafNode{}

func (l leafNode) getHeaders() headers {
	return l.headers
}

func (l leafNode) GetHeader(name string) []string {
	return l.headers.get(name)
}

func (l leafNode) walkLeaves(fn walkLeavesFunc) walkStatus {
	return fn(l)
}

func (l leafNode) GetBody() []byte {
	return l.body
}

func (l leafNode) isAttachment() bool {
	contentDisposition := getHeaderValue(l, "Content-Disposition")
	return strings.HasPrefix(contentDisposition, "attachment")
}

func (l leafNode) GetDecodedBody() string {
	bodyBytes := l.GetBody()
	encoding := l.getContentTransferEncoding()

	switch strings.ToLower(encoding) {
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(string(bodyBytes))
		if err == nil {
			return string(decoded)
		}
		// Fallback to original body if base64 decoding fails
		return string(bodyBytes)
	case "quoted-printable":
		qpr := quotedprintable.NewReader(bytes.NewReader(bodyBytes))
		decodedBytes, err := io.ReadAll(qpr)
		if err == nil {
			return string(decodedBytes)
		}
		// Fallback to original body if QP decoding fails
		return string(bodyBytes)
	default:
		// Includes "7bit", "8bit", binary, or no encoding specified
		return string(bodyBytes)
	}
	// FIXME: read the charset from the Content-Type header and decode the body
}

func (l leafNode) isPlainText() bool {
	return strings.HasPrefix(getContentType(l.getHeaders()), "text/plain")
}

func (l leafNode) isHTML() bool {
	return strings.HasPrefix(getContentType(l.getHeaders()), "text/html")
}

func (l leafNode) isWatchHTML() bool {
	return strings.HasPrefix(getContentType(l.getHeaders()), "text/watch-html")
}

func (l leafNode) getContentTransferEncoding() string {
	return getHeaderValue(l, "Content-Transfer-Encoding")
}

func (l leafNode) GetContentID() string {
	cid := getHeaderValue(l, "Content-ID")
	// Trim angle brackets, if present
	cid = strings.TrimPrefix(cid, "<")
	cid = strings.TrimSuffix(cid, ">")
	return cid
}
