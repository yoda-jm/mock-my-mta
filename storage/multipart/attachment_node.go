package multipart

import (
	"mime"
	"strings"
)

type AttachmentNode struct {
	leafNode
}

func (l AttachmentNode) GetContentType() string {
	return getHeaderValue(l, "Content-Type")
}

func (l AttachmentNode) GetFilename() string {
	contentDisposition := getHeaderValue(l, "Content-Disposition")
	if contentDisposition == "" {
		return ""
	}

	// mime.ParseMediaType handles both RFC 2047 and RFC 2231 parameter encoding
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		// Fallback to manual parsing for malformed headers
		return parseFilenameManual(contentDisposition)
	}

	if filename, ok := params["filename"]; ok {
		return filename
	}
	return ""
}

// parseFilenameManual is a fallback for malformed Content-Disposition headers.
func parseFilenameManual(contentDisposition string) string {
	parts := strings.Split(contentDisposition, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "filename=") {
			return strings.Trim(part[9:], "\"")
		}
	}
	return ""
}

func (l AttachmentNode) GetSize() int {
	return len(l.body)
}
