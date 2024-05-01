package multipart

import (
	"net/mail"
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
		multipart.WalfLeaves(func(leaf LeafNode) WalkStatus {
			if leaf.IsAttachment() {
				t.Logf("Attachment: %v %v %v", leaf.GetAttachmentContentType(), leaf.GetAttachmentFilename(), leaf.GetAttachmentSize())
				t.Logf("Attachment body: %v", string(leaf.GetBody()))
			}
			// continue walking
			return ContinueWalk
		})
	}
}
