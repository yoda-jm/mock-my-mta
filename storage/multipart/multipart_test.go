package multipart

import (
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var simpleEmail = `From: from@example.com
To: to1@example.com
Subject: Test email

This is the body of the email.`

var messageSimpleAttachment = `From: from@example.com
To: to1@example.com, to2@example.com
Subject: Test email
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="boundary"

--boundary
Content-Type: text/plain
Content-Disposition: attachment; filename="file.txt"

This is the attachment.

--boundary
Content-Type: text/plain

This is the body of the email.


--boundary--
`

var messagePlainHTML = `From: from@example.com
To: to1@example.com, to2@example.com
Subject: Test email
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="boundary"

--boundary
Content-Type: text/plain

This is the body of the email.

--boundary
Content-Type: text/html

<html>
<body>
<p>This is the body of the email.</p>
</body>
</html>

--boundary--`

var messagePlainHTMLAttachment = `From: from@example.com
To: to1@example.com, to2@example.com
Subject: Test email
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="boundary"

--boundary
Content-Type: multipart/alternative; boundary="boundary2"

--boundary2
Content-Type: text/plain

This is the body of the email.

--boundary2
Content-Type: text/html

<html>
<body>
<p>This is the body of the email.</p>
</body>
</html>

--boundary2--
--boundary
Content-Type: text/plain
Content-Disposition: attachment; filename="file.txt"

This is the attachment of a an email with plain and html body.

--boundary
Content-Type: text/plain
Content-Disposition: attachment; filename="file2.txt"

This is another attachment of a an email with plain and html body.

--boundary--`

func TestParseMultipartEmail(t *testing.T) {
	// create a new email
	messages := map[string]string{
		"simpleEmail":         simpleEmail,
		"simpleAttachment":    messageSimpleAttachment,
		"plainHTML":           messagePlainHTML,
		"plainHTMLAttachment": messagePlainHTMLAttachment,
	}
	for name, message := range messages {
		t.Logf("Parsing email: %v", name)
		// read the email
		email, err := mail.ReadMessage(strings.NewReader(message))
		if err != nil {
			t.Fatal(err)
		}
		// parse the email
		multipart, err := New(email)
		if err != nil {
			t.Fatal(err)
		}
		// print the email
		t.Logf("Printing email:\n%v", multipart)
	}
}

func TestParseMultipartEmailHasAttachments(t *testing.T) {
	// create a new email
	messages := map[string]string{
		"simpleAttachment":    messageSimpleAttachment,
		"plainHTMLAttachment": messagePlainHTMLAttachment,
	}
	for name, message := range messages {
		t.Logf("Parsing email: %v", name)
		// read the email
		email, err := mail.ReadMessage(strings.NewReader(message))
		if err != nil {
			t.Fatal(err)
		}
		// parse the email
		multipart, err := New(email)
		if err != nil {
			t.Fatal(err)
		}
		// check if the email has attachments
		if !multipart.HasAttachments() {
			t.Fatalf("Email %v should have attachments", name)
		}
	}
}

func TestParseMultipartEmailGetPreview(t *testing.T) {
	// create a new email
	messages := map[string]string{
		"simpleEmail":         simpleEmail,
		"simpleAttachment":    messageSimpleAttachment,
		"plainHTML":           messagePlainHTML,
		"plainHTMLAttachment": messagePlainHTMLAttachment,
	}
	for name, message := range messages {
		t.Logf("Parsing email: %v", name)
		// read the email
		email, err := mail.ReadMessage(strings.NewReader(message))
		if err != nil {
			t.Fatal(err)
		}
		// parse the email
		multipart, err := New(email)
		if err != nil {
			t.Fatal(err)
		}
		// print the email
		t.Logf("printing email preview:\n%v", multipart.GetPreview())
	}
}

func TestParseMultipartEmailGetAttachments(t *testing.T) {
	// create a new email
	messages := map[string]string{
		"simpleAttachment":    messageSimpleAttachment,
		"plainHTMLAttachment": messagePlainHTMLAttachment,
	}
	for name, message := range messages {
		t.Logf("Parsing email: %v", name)
		// read the email
		email, err := mail.ReadMessage(strings.NewReader(message))
		if err != nil {
			t.Fatal(err)
		}
		// parse the email
		multipart, err := New(email)
		if err != nil {
			t.Fatal(err)
		}
		// print the email
		// Iterate over attachments using GetAttachments()
		attachments := multipart.GetAttachments()
		for id, att := range attachments {
			t.Logf("Attachment [%s]: %v %v %v", id, att.GetContentType(), att.GetFilename(), att.GetSize())
			t.Logf("Attachment body: %v", string(att.GetBody())) // Use getBody (lowercase)
		}
	}
}

func loadTestEmail(t *testing.T, filePath string) *Multipart {
	t.Helper()
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read email file %s: %v", filePath, err)
	}
	email, err := mail.ReadMessage(strings.NewReader(string(content)))
	if err != nil {
		t.Fatalf("Failed to parse email from file %s: %v", filePath, err)
	}
	mp, err := New(email)
	if err != nil {
		t.Fatalf("Failed to create Multipart from email %s: %v", filePath, err)
	}
	return mp
}

func TestDecodeHeader(t *testing.T) {
	tests := []struct {
		name     string
		encoded  string
		expected string
	}{
		{
			name:     "UTF-8 Q-encoding",
			encoded:  "=?UTF-8?Q?This_is_a_test?= <test@example.com>",
			expected: "This is a test <test@example.com>",
		},
		{
			name:     "UTF-8 Q-encoding with special characters",
			encoded:  "=?UTF-8?Q?=E2=98=9D_Point_here_?= <test@example.com>",
			expected: "☝ Point here  <test@example.com>",
		},
		{
			name:     "ISO-8859-1 B-encoding",
			encoded:  "=?ISO-8859-1?B?SWYgeW91IGNhbiByZWFkIHRoaXMgeW8=?= =?ISO-8859-1?B?dSB1bmRlcnN0YW5kIHRoZSBleGFtcGxlLg==?=",
			expected: "If you can read this you understand the example.",
		},
		{
			name:     "KOI8-R Q-encoding",
			encoded:  "=?KOI8-R?Q?=C3=CF=D7=C5=D4=2C_=D7=CF=D2=CF=C4=CE=C9=CA=21?= <test@example.com>",
			expected: "=?KOI8-R?Q?=C3=CF=D7=C5=D4=2C_=D7=CF=D2=CF=C4=CE=C9=CA=21?= <test@example.com>",
		},
		{
			name:     "Mixed Q-encoded and B-encoded",
			encoded:  "=?UTF-8?Q?First_part?= =?UTF-8?B?U2Vjb25kIHBhcnQ=?=",
			expected: "First partSecond part",
		},
		{
			name:     "Encoded word at the beginning",
			encoded:  "=?UTF-8?Q?Beginning?= rest of the string",
			expected: "Beginning rest of the string",
		},
		{
			name:     "Encoded word in the middle",
			encoded:  "Start =?UTF-8?Q?Middle?= End",
			expected: "Start Middle End",
		},
		{
			name:     "Encoded word at the end",
			encoded:  "Start of the string =?UTF-8?Q?End?=",
			expected: "Start of the string End",
		},
		{
			name:     "Multiple encoded words",
			encoded:  "=?UTF-8?Q?First?= =?UTF-8?Q?Second?= =?UTF-8?Q?Third?=",
			expected: "FirstSecondThird",
		},
		{
			name:     "Not encoded",
			encoded:  "This is a simple string",
			expected: "This is a simple string",
		},
		{
			name:     "Empty string",
			encoded:  "",
			expected: "",
		},
		{
			name:     "Malformed header - invalid encoding",
			encoded:  "=?INVALID_ENCODING?Q?Test?=",
			expected: "=?INVALID_ENCODING?Q?Test?=",
		},
		{
			name:     "Malformed header - invalid Q-encoding",
			encoded:  "=?UTF-8?Q?Test%=?=",
			expected: "=?UTF-8?Q?Test%=?=",
		},
		{
			name:     "From header with display name ISO-8859-1 B-encoding",
			encoded:  "=?ISO-8859-1?B?R2lvdmFubmkgR2FsbGk=?= <test@example.com>",
			expected: "Giovanni Galli <test@example.com>",
		},
		{
			name:     "Subject with UTF-8 chars Q-encoding",
			encoded:  "=?UTF-8?Q?Test_subject_with_special_chars_?= =?UTF-8?Q?=C3=A9=C3=A0=C3=B2?= <test@example.com>",
			expected: "Test subject with special chars éàò <test@example.com>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded := decodeHeader(tt.encoded)
			if decoded != tt.expected {
				t.Errorf("decodeHeader(%q) got %q, want %q", tt.encoded, decoded, tt.expected)
			}
		})
	}
}

func TestMultipartAlternative(t *testing.T) {
	basePath := "testdata"

	t.Run("AlternativePlainHTML", func(t *testing.T) {
		mp := loadTestEmail(t, filepath.Join(basePath, "alternative_plain_html.eml"))

		// Test GetPreview
		expectedPreview := "This is the plain text part. It has some special chars like éàò."
		preview := mp.GetPreview()
		if preview != expectedPreview {
			t.Errorf("GetPreview() got %q, want %q", preview, expectedPreview)
		}

		// Test GetBody("plain-text")
		plainBody, err := mp.GetBody("plain-text")
		if err != nil {
			t.Errorf("GetBody(\"plain-text\") returned error: %v", err)
		}
		expectedPlainBodyString := "This is the plain text part.\nIt has some special chars like éàò."
		if !strings.Contains(plainBody, expectedPlainBodyString) {
			t.Errorf("GetBody(\"plain-text\") got %q, want to contain %q", plainBody, expectedPlainBodyString)
		}

		// Test GetBody("html")
		htmlBody, err := mp.GetBody("html")
		if err != nil {
			t.Errorf("GetBody(\"html\") returned error: %v", err)
		}
		if !strings.Contains(htmlBody, "<p>This is the <b>HTML</b> part.</p>") {
			t.Errorf("GetBody(\"html\") got %q, want to contain %q", htmlBody, "<p>This is the <b>HTML</b> part.</p>")
		}

		// Test GetBodyVersions
		versions := mp.GetBodyVersions()
		expectedVersions := map[string]bool{"plain-text": true, "html": true}
		if len(versions) != len(expectedVersions) {
			t.Errorf("GetBodyVersions() got %v, want %v", versions, expectedVersions)
		}
		for _, v := range versions {
			if !expectedVersions[v] {
				t.Errorf("GetBodyVersions() got unexpected version %q in %v", v, versions)
			}
		}
	})

	t.Run("AlternativeOnlyHTML", func(t *testing.T) {
		mp := loadTestEmail(t, filepath.Join(basePath, "alternative_only_html.eml"))
		preview := mp.GetPreview()
		// Preview should NOT contain HTML tags — they must be stripped
		if strings.Contains(preview, "<") || strings.Contains(preview, ">") {
			t.Errorf("GetPreview() should not contain HTML tags, got %q", preview)
		}
		if !strings.Contains(preview, "This is the") {
			t.Errorf("GetPreview() should contain readable text, got %q", preview)
		}

		plainBody, err := mp.GetBody("plain-text")
		if err != nil {
			t.Errorf("GetBody(\"plain-text\") returned error: %v", err)
		}
		if plainBody != "" {
			t.Errorf("GetBody(\"plain-text\") got %q, want \"\"", plainBody)
		}

		htmlBody, err := mp.GetBody("html")
		if err != nil {
			t.Errorf("GetBody(\"html\") returned error: %v", err)
		}
		if !strings.Contains(htmlBody, "This is the <b>HTML</b> part only.") {
			t.Errorf("GetBody(\"html\") got %q, want to contain %q", htmlBody, "This is the <b>HTML</b> part only.")
		}
		versions := mp.GetBodyVersions()
		if !(len(versions) == 1 && versions[0] == "html") {
			t.Errorf("GetBodyVersions() got %v, want [\"html\"]", versions)
		}
	})

	t.Run("AlternativeOnlyPlain", func(t *testing.T) {
		mp := loadTestEmail(t, filepath.Join(basePath, "alternative_only_plain.eml"))
		expectedPlainPreview := "This is the plain text part only."
		preview := mp.GetPreview()
		if preview != expectedPlainPreview {
			t.Errorf("GetPreview() got %q, want %q", preview, expectedPlainPreview)
		}

		plainBody, err := mp.GetBody("plain-text")
		if err != nil {
			t.Errorf("GetBody(\"plain-text\") returned error: %v", err)
		}
		if !strings.Contains(plainBody, "This is the plain text part only.") {
			t.Errorf("GetBody(\"plain-text\") got %q, want to contain %q", plainBody, "This is the plain text part only.")
		}
		htmlBody, err := mp.GetBody("html")
		if err != nil {
			t.Errorf("GetBody(\"html\") returned error: %v", err)
		}
		if htmlBody != "" {
			t.Errorf("GetBody(\"html\") got %q, want \"\"", htmlBody)
		}
		versions := mp.GetBodyVersions()
		if !(len(versions) == 1 && versions[0] == "plain-text") {
			t.Errorf("GetBodyVersions() got %v, want [\"plain-text\"]", versions)
		}
	})

	t.Run("MixedAlternative", func(t *testing.T) {
		mp := loadTestEmail(t, filepath.Join(basePath, "mixed_alternative.eml"))

		// Preview should be from the plain text part of the alternative
		expectedPreview := "This is the plain text part of the alternative."
		preview := mp.GetPreview()
		if preview != expectedPreview {
			t.Errorf("GetPreview() got %q, want %q", preview, expectedPreview)
		}

		// GetBody plain text from alternative
		plainBody, err := mp.GetBody("plain-text")
		if err != nil {
			t.Errorf("GetBody(\"plain-text\") returned error: %v", err)
		}
		expectedPlainBodyMixed := "This is the plain text part of the alternative."
		if !strings.Contains(plainBody, expectedPlainBodyMixed) {
			t.Errorf("GetBody(\"plain-text\") got %q, want to contain %q", plainBody, expectedPlainBodyMixed)
		}

		// GetBody HTML from alternative
		htmlBody, err := mp.GetBody("html")
		if err != nil {
			t.Errorf("GetBody(\"html\") returned error: %v", err)
		}
		if !strings.Contains(htmlBody, "<p>This is the <b>HTML</b> part of the alternative.</p>") {
			t.Errorf("GetBody(\"html\") got %q, want to contain %q", htmlBody, "<p>This is the <b>HTML</b> part of the alternative.</p>")
		}

		// Check for attachments
		if !mp.HasAttachments() {
			t.Errorf("HasAttachments() returned false, want true")
		}
		attachments := mp.GetAttachments()
		if len(attachments) != 1 {
			t.Errorf("GetAttachments() returned %d attachments, want 1", len(attachments))
		}
		// Note: Attachment iteration order is not guaranteed by map.
		// For a more robust test, iterate and check filenames or content.
		var foundAttachment bool
		for _, att := range attachments {
			if att.GetFilename() == "attachment.txt" {
				foundAttachment = true
				if !strings.Contains(string(att.GetBody()), "This is an attachment.") {
					t.Errorf("Attachment content incorrect: got %s", string(att.GetBody()))
				}
			}
		}
		if !foundAttachment {
			t.Errorf("Attachment with filename 'attachment.txt' not found")
		}

		// GetBodyVersions should include both from alternative
		versions := mp.GetBodyVersions()
		expectedVersions := map[string]bool{"plain-text": true, "html": true}
		if len(versions) != len(expectedVersions) {
			t.Errorf("GetBodyVersions() got %v, want versions for %v", versions, expectedVersions)
		}
		for _, v := range versions {
			if !expectedVersions[v] {
				t.Errorf("GetBodyVersions() got unexpected version %q in %v", v, versions)
			}
		}
	})

	// --- Bug fix tests (expected to FAIL until fixed) ---

	t.Run("HTMLOnlyNoMultipart_PreviewStripsHTML", func(t *testing.T) {
		mp := loadTestEmail(t, filepath.Join(basePath, "html_only_no_multipart.eml"))

		preview := mp.GetPreview()
		// Preview must NOT contain HTML tags
		if strings.Contains(preview, "<") || strings.Contains(preview, ">") {
			t.Errorf("GetPreview() should strip HTML tags, got %q", preview)
		}
		// Preview should contain the readable text
		if !strings.Contains(preview, "Welcome") {
			t.Errorf("GetPreview() should contain 'Welcome', got %q", preview)
		}
		if !strings.Contains(preview, "pure HTML email") {
			t.Errorf("GetPreview() should contain 'pure HTML email', got %q", preview)
		}
	})

	t.Run("NoContentType_GetBodyPlainText", func(t *testing.T) {
		mp := loadTestEmail(t, filepath.Join(basePath, "no_content_type.eml"))

		// Email with no Content-Type should default to text/plain
		body, err := mp.GetBody("plain-text")
		if err != nil {
			t.Errorf("GetBody(\"plain-text\") returned error: %v", err)
		}
		if body == "" {
			t.Errorf("GetBody(\"plain-text\") returned empty string, want body content")
		}
		if !strings.Contains(body, "no Content-Type header") {
			t.Errorf("GetBody(\"plain-text\") got %q, want to contain 'no Content-Type header'", body)
		}

		// Preview should also work
		preview := mp.GetPreview()
		if preview == "" {
			t.Errorf("GetPreview() returned empty for email without Content-Type")
		}

		// Body versions should include plain-text
		versions := mp.GetBodyVersions()
		found := false
		for _, v := range versions {
			if v == "plain-text" {
				found = true
			}
		}
		if !found {
			t.Errorf("GetBodyVersions() = %v, want to include 'plain-text'", versions)
		}
	})

	t.Run("ISO8859Body_CharsetDecoded", func(t *testing.T) {
		mp := loadTestEmail(t, filepath.Join(basePath, "iso8859_body.eml"))

		body, err := mp.GetBody("plain-text")
		if err != nil {
			t.Errorf("GetBody(\"plain-text\") returned error: %v", err)
		}

		// After proper charset decoding, ISO-8859-1 bytes should be UTF-8
		// E9=é, E8=è, FB=û, E7=ç, EA=ê
		if !strings.Contains(body, "Café") {
			t.Errorf("GetBody(\"plain-text\") should contain 'Café' (decoded from ISO-8859-1), got %q", body)
		}
		if !strings.Contains(body, "crème") {
			t.Errorf("GetBody(\"plain-text\") should contain 'crème' (decoded from ISO-8859-1), got %q", body)
		}
		if !strings.Contains(body, "brûlée") {
			t.Errorf("GetBody(\"plain-text\") should contain 'brûlée' (decoded from ISO-8859-1), got %q", body)
		}
	})

	t.Run("RFC2231Filename_Decoded", func(t *testing.T) {
		mp := loadTestEmail(t, filepath.Join(basePath, "rfc2231_filename.eml"))

		if !mp.HasAttachments() {
			t.Fatal("HasAttachments() returned false, want true")
		}

		attachments := mp.GetAttachments()
		var foundFilename string
		for _, att := range attachments {
			foundFilename = att.GetFilename()
		}

		// RFC 2231 encoded filename*=UTF-8''t%C3%A9st%20r%C3%A9sum%C3%A9.pdf
		// should decode to: tést résumé.pdf
		expected := "tést résumé.pdf"
		if foundFilename != expected {
			t.Errorf("GetFilename() = %q, want %q (RFC 2231 decoded)", foundFilename, expected)
		}
	})

	t.Run("WatchHTML_BodyReturnsContent", func(t *testing.T) {
		mp := loadTestEmail(t, filepath.Join(basePath, "alternative_with_watch_html.eml"))

		// watch-html should be in body versions
		versions := mp.GetBodyVersions()
		found := false
		for _, v := range versions {
			if v == "watch-html" {
				found = true
			}
		}
		if !found {
			t.Errorf("GetBodyVersions() = %v, want to include 'watch-html'", versions)
		}

		// GetBody("watch-html") should return the watch-html content
		watchBody, err := mp.GetBody("watch-html")
		if err != nil {
			t.Errorf("GetBody(\"watch-html\") returned error: %v", err)
		}
		if watchBody == "" {
			t.Error("GetBody(\"watch-html\") returned empty string")
		}
		if !strings.Contains(watchBody, "Watch HTML part") {
			t.Errorf("GetBody(\"watch-html\") = %q, want to contain 'Watch HTML part'", watchBody)
		}

		// Other body versions should also work
		htmlBody, err := mp.GetBody("html")
		if err != nil {
			t.Errorf("GetBody(\"html\") returned error: %v", err)
		}
		if !strings.Contains(htmlBody, "standard HTML part") {
			t.Errorf("GetBody(\"html\") = %q, want to contain 'standard HTML part'", htmlBody)
		}

		plainBody, err := mp.GetBody("plain-text")
		if err != nil {
			t.Errorf("GetBody(\"plain-text\") returned error: %v", err)
		}
		if !strings.Contains(plainBody, "plain text part") {
			t.Errorf("GetBody(\"plain-text\") = %q, want to contain 'plain text part'", plainBody)
		}
	})

	// --- New feature tests ---

	t.Run("GetAllHeaders_ReturnsDecodedHeaders", func(t *testing.T) {
		mp := loadTestEmail(t, filepath.Join(basePath, "alternative_plain_html.eml"))

		headers := mp.GetAllHeaders()
		if len(headers) == 0 {
			t.Fatal("GetAllHeaders() returned empty map")
		}

		// Check that common headers are present (keys are lowercase)
		if _, ok := headers["from"]; !ok {
			t.Error("GetAllHeaders() missing 'from' header")
		}
		if _, ok := headers["to"]; !ok {
			t.Error("GetAllHeaders() missing 'to' header")
		}
		if _, ok := headers["subject"]; !ok {
			t.Error("GetAllHeaders() missing 'subject' header")
		}
		if _, ok := headers["content-type"]; !ok {
			t.Error("GetAllHeaders() missing 'content-type' header")
		}
	})

	t.Run("GetMimeTree_SimpleEmail", func(t *testing.T) {
		email, err := mail.ReadMessage(strings.NewReader(simpleEmail))
		if err != nil {
			t.Fatal(err)
		}
		mp, err := New(email)
		if err != nil {
			t.Fatal(err)
		}

		tree := mp.GetMimeTree()
		if tree == nil {
			t.Fatal("GetMimeTree() returned nil")
		}
		// Simple email is a leaf — should have text/plain content type, no children
		if !strings.HasPrefix(tree.ContentType, "text/plain") {
			t.Errorf("GetMimeTree().ContentType = %q, want prefix 'text/plain'", tree.ContentType)
		}
		if len(tree.Children) > 0 {
			t.Errorf("GetMimeTree() simple email should have no children, got %d", len(tree.Children))
		}
		if tree.Size == 0 {
			t.Error("GetMimeTree().Size should be > 0 for a leaf node")
		}
	})

	t.Run("GetMimeTree_MultipartEmail", func(t *testing.T) {
		mp := loadTestEmail(t, filepath.Join(basePath, "mixed_alternative.eml"))

		tree := mp.GetMimeTree()
		if tree == nil {
			t.Fatal("GetMimeTree() returned nil")
		}
		// mixed_alternative.eml: multipart/mixed > [attachment.txt, multipart/alternative > [plain, html]]
		if !strings.HasPrefix(tree.ContentType, "multipart/mixed") {
			t.Errorf("GetMimeTree().ContentType = %q, want 'multipart/mixed'", tree.ContentType)
		}
		if len(tree.Children) != 2 {
			t.Fatalf("GetMimeTree() should have 2 children, got %d", len(tree.Children))
		}

		// First child is the attachment
		att := tree.Children[0]
		if att.Filename != "attachment.txt" {
			t.Errorf("First child Filename = %q, want 'attachment.txt'", att.Filename)
		}
		if att.Disposition == "" {
			t.Error("Attachment node should have a disposition")
		}

		// Second child is multipart/alternative with plain + html
		alt := tree.Children[1]
		if !strings.HasPrefix(alt.ContentType, "multipart/alternative") {
			t.Errorf("Second child ContentType = %q, want 'multipart/alternative'", alt.ContentType)
		}
		if len(alt.Children) != 2 {
			t.Errorf("Alternative node should have 2 children (plain + html), got %d", len(alt.Children))
		}
	})

	t.Run("AlternativeNestedMixed", func(t *testing.T) {
		mp := loadTestEmail(t, filepath.Join(basePath, "alternative_nested_mixed.eml"))
		// Preview should be from the plain text part of the alternative
		expectedPreview := "This is the plain text part of the alternative."
		preview := mp.GetPreview()
		if preview != expectedPreview {
			t.Errorf("GetPreview() got %q, want %q", preview, expectedPreview)
		}

		// GetBody plain text from alternative
		plainBody, err := mp.GetBody("plain-text")
		if err != nil {
			t.Errorf("GetBody(\"plain-text\") returned error: %v", err)
		}
		expectedPlainBodyNested := "This is the plain text part of the alternative."
		if !strings.Contains(plainBody, expectedPlainBodyNested) {
			t.Errorf("GetBody(\"plain-text\") got %q, want to contain %q", plainBody, expectedPlainBodyNested)
		}

		// GetBody HTML from alternative
		htmlBody, err := mp.GetBody("html")
		if err != nil {
			t.Errorf("GetBody(\"html\") returned error: %v", err)
		}
		if !strings.Contains(htmlBody, "<p>This is the <b>HTML</b> part of the alternative.</p>") {
			t.Errorf("GetBody(\"html\") got %q, want to contain %q", htmlBody, "<p>This is the <b>HTML</b> part of the alternative.</p>")
		}

		// Check for attachments
		if !mp.HasAttachments() {
			t.Errorf("HasAttachments() returned false, want true")
		}
		attachments := mp.GetAttachments()
		if len(attachments) != 1 {
			// This might fail if GetAttachments is not correctly skipping non-attachment parts within the alternative
			t.Fatalf("GetAttachments() returned %d attachments, want 1. Attachments: %v", len(attachments), attachments)
		}
		var foundAttachment bool
		for _, att := range attachments {
			if att.GetFilename() == "attachment.txt" {
				foundAttachment = true
				if !strings.Contains(string(att.GetBody()), "This is an attachment.") {
					t.Errorf("Attachment content incorrect: got %s", string(att.GetBody()))
				}
			}
		}
		if !foundAttachment {
			t.Errorf("Attachment with filename 'attachment.txt' not found")
		}

		// GetBodyVersions should include both from alternative
		versions := mp.GetBodyVersions()
		expectedVersions := map[string]bool{"plain-text": true, "html": true}
		if len(versions) != len(expectedVersions) {
			t.Errorf("GetBodyVersions() got %v, want versions for %v", versions, expectedVersions)
		}
		for _, v := range versions {
			if !expectedVersions[v] {
				t.Errorf("GetBodyVersions() got unexpected version %q in %v", v, versions)
			}
		}
	})
}
