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
		// expectedHTMLPreview is used implicitly by the HasPrefix check
		preview := mp.GetPreview()
		if !strings.HasPrefix(preview, "<html><body><p>This is the <b>HTML</b> part only.") { // Check prefix due to potential truncation
			t.Errorf("GetPreview() got %q, want prefix %q", preview, "<html><body><p>This is the <b>HTML</b> part only.")
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
