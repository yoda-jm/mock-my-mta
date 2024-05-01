package multipart

import (
	"encoding/base64"
	"strings"
)

type LeafNode struct {
	headers map[string][]string
	body    []byte
}

func (l LeafNode) GetHeaders() map[string][]string {
	return l.headers
}

func (l LeafNode) GetBody() []byte {
	return l.body
}

func (l LeafNode) WalfLeaves(fn WalkLeavesFunc) WalkStatus {
	return fn(l)
}

func (l LeafNode) IsAttachment() bool {
	contentDisposition := getHeaderValue(l, "Content-Disposition")
	return strings.HasPrefix(contentDisposition, "attachment")
}

func (l LeafNode) GetAttachmentContentType() string {
	return getHeaderValue(l, "Content-Type")
}

func (l LeafNode) GetAttachmentFilename() string {
	contentDisposition := getHeaderValue(l, "Content-Disposition")
	parts := strings.Split(contentDisposition, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "filename=") {
			return strings.Trim(part[9:], "\"")
		}
	}
	return ""
}

func (l LeafNode) GetDecodedBody() string {
	contentTransferEncoding := getHeaderValue(l, "Content-Transfer-Encoding")
	body := string(l.GetBody())
	if contentTransferEncoding == "base64" {
		// decode the body
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err == nil {
			body = string(decoded)
		}
	}
	// FIXME: read the charset from the Content-Type header and decode the body
	return body
}

func (l LeafNode) IsPlainText() bool {
	return strings.HasPrefix(getContentType(l.GetHeaders()), "text/plain")
}

func (l LeafNode) IsHTML() bool {
	return strings.HasPrefix(getContentType(l.GetHeaders()), "text/html")
}

func (l LeafNode) IsWatchHTML() bool {
	return strings.HasPrefix(getContentType(l.GetHeaders()), "text/watch-html")
}

func (l LeafNode) GetAttachmentSize() int {
	return len(l.body)
}

func (l LeafNode) GetContentTransferEncoding() string {
	return getHeaderValue(l, "Content-Transfer-Encoding")
}
