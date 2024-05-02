package multipart

import (
	"encoding/base64"
	"strings"
)

type leafNode struct {
	headers map[string][]string
	body    []byte
}

// leaf node implements the node interface
var _ node = leafNode{}

func (l leafNode) getHeaders() map[string][]string {
	return l.headers
}

func (l leafNode) walfLeaves(fn walkLeavesFunc) walkStatus {
	return fn(l)
}

func (l leafNode) getBody() []byte {
	return l.body
}

func (l leafNode) isAttachment() bool {
	contentDisposition := getHeaderValue(l, "Content-Disposition")
	return strings.HasPrefix(contentDisposition, "attachment")
}

func (l leafNode) GetDecodedBody() string {
	body := string(l.getBody())
	if l.getContentTransferEncoding() == "base64" {
		// decode the body
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err == nil {
			body = string(decoded)
		}
	}
	// FIXME: read the charset from the Content-Type header and decode the body
	return body
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
