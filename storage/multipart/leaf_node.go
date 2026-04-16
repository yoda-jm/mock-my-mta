package multipart

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/quotedprintable"
	"strings"

	"golang.org/x/text/encoding/ianaindex"
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

	// Step 1: decode Content-Transfer-Encoding
	var decoded []byte
	switch strings.ToLower(encoding) {
	case "base64":
		d, err := base64.StdEncoding.DecodeString(string(bodyBytes))
		if err == nil {
			decoded = d
		} else {
			decoded = bodyBytes
		}
	case "quoted-printable":
		qpr := quotedprintable.NewReader(bytes.NewReader(bodyBytes))
		d, err := io.ReadAll(qpr)
		if err == nil {
			decoded = d
		} else {
			decoded = bodyBytes
		}
	default:
		decoded = bodyBytes
	}

	// Step 2: convert charset to UTF-8
	charset := l.getCharset()
	if charset != "" && !isUTF8Charset(charset) {
		enc, err := ianaindex.IANA.Encoding(charset)
		if err == nil && enc != nil {
			utf8Bytes, err := enc.NewDecoder().Bytes(decoded)
			if err == nil {
				return string(utf8Bytes)
			}
		}
	}

	return string(decoded)
}

// getCharset extracts the charset parameter from the Content-Type header.
func (l leafNode) getCharset() string {
	ct := getContentType(l.getHeaders())
	if ct == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return ""
	}
	return params["charset"]
}

func isUTF8Charset(charset string) bool {
	c := strings.ToLower(strings.TrimSpace(charset))
	return c == "utf-8" || c == "utf8" || c == "us-ascii" || c == "ascii"
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
