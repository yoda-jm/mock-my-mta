package multipart

import "strings"

type AttachmentNode struct {
	leafNode
}

func (l AttachmentNode) GetContentType() string {
	return getHeaderValue(l, "Content-Type")
}

func (l AttachmentNode) GetFilename() string {
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

func (l AttachmentNode) GetSize() int {
	return len(l.body)
}
