package main

import (
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strings"

	"mock-my-mta/storage/multipart" // Full module path
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <eml_file_path1> [<eml_file_path2> ...]\n", os.Args[0])
		os.Exit(1)
	}

	for _, filePath := range os.Args[1:] {
		// Construct path relative to /app, as that's where testdata is.
		// The analyzer will be run from /tmp, but needs to reference files in /app/testdata
		var absPath string
		if filepath.IsAbs(filePath) {
			absPath = filePath
		} else {
			// Assume filePath is like "testdata/related_html_image.eml"
			// and current working directory for the binary will be /app
			// when the bash session executes it.
			// However, the binary is in /tmp. So, need to adjust.
			// The paths passed to /tmp/analyzer will be like "testdata/..."
			// and os.ReadFile will resolve them from the CWD of the binary.
			// So, if we run `cd /app && /tmp/analyzer testdata/file.eml`, it works.

			// Let's assume the user of the script passes a path that works from the execution CWD.
			// For the bash session, we will `cd /app` then run `/tmp/analyzer testdata/file.eml`
			var err error
			absPath, err = filepath.Abs(filePath) // This will be relative to CWD of /tmp/analyzer
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting absolute path for %s: %v\n", filePath, err)
				continue
			}
		}

		emailData, err := os.ReadFile(filePath) // filePath is now used directly
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading email file %s (resolved to %s): %v\n", filePath, absPath, err)
			continue
		}

		reader := strings.NewReader(string(emailData))
		msg, err := mail.ReadMessage(reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing email from file %s: %v\n", filePath, err)
			continue
		}

		mp, err := multipart.New(msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Multipart from email %s: %v\n", filePath, err)
			continue
		}

		fmt.Printf("--- Analysis for EML: %s ---\n", filePath)
		fmt.Printf("Subject: %s\n", mp.GetSubject())

		htmlBody, errHtml := mp.GetBody("html")
		if errHtml != nil {
			fmt.Printf("Error getting HTML body: %v\n", errHtml)
		} else if htmlBody != "" {
			fmt.Printf("HTML Body: Present (approx. %d chars)\n", len(htmlBody))
			if strings.Contains(strings.ToLower(htmlBody), "<img") {
				fmt.Println("HTML Body Content Hint: Contains <img> tag.")
			}
		} else {
			fmt.Println("HTML Body: Not present or empty.")
		}

		plainBody, errPlain := mp.GetBody("plain-text")
		if errPlain != nil {
			fmt.Printf("Error getting Plain Text body: %v\n", errPlain)
		} else if plainBody != "" {
			fmt.Printf("Plain Text Body: Present (approx. %d chars)\n", len(plainBody))
		} else {
			fmt.Println("Plain Text Body: Not present or empty.")
		}

		fmt.Printf("Preview: %s\n", mp.GetPreview())
		bodyVersions := mp.GetBodyVersions()
		fmt.Printf("Body Versions: %v (Count: %d)\n", bodyVersions, len(bodyVersions))
		fmt.Printf("Has Attachments: %v\n", mp.HasAttachments())

		attachments := mp.GetAttachments()
		fmt.Printf("Number of Attachments found by GetAttachments(): %d\n", len(attachments))
		if len(attachments) > 0 {
			i := 0
			for _, attachment := range attachments {
				fmt.Printf("  Attachment %d (from GetAttachments):\n", i)
				fmt.Printf("    Filename: %s\n", attachment.GetFilename())
				fmt.Printf("    Content-Type: %s\n", attachment.GetContentType())
				// AttachmentNode.GetContentID() is not yet implemented.
				// Accessing headers directly is also not possible via public API of AttachmentNode.
				// For this analysis, we will state that Content-ID cannot be directly retrieved.
				fmt.Printf("    Content-ID: (Cannot be directly retrieved with current AttachmentNode API)\n")

				// Manually note expected CIDs for the report based on EML filename
				// This is for reporting clarity, not something the code can currently do.
				var expectedCIDManuallyKnown string
				if strings.HasSuffix(filePath, "related_html_image.eml") && attachment.GetFilename() == "image.png" {
					expectedCIDManuallyKnown = "image1@example.com"
				} else if strings.HasSuffix(filePath, "related_with_start_param.eml") && attachment.GetFilename() == "image.gif" {
					expectedCIDManuallyKnown = "image_part@example.com"
				} else if strings.HasSuffix(filePath, "related_nested_mixed.eml") {
					if attachment.GetFilename() == "photo.jpg" {
						expectedCIDManuallyKnown = "nested_image@example.com"
					}
				}
				if expectedCIDManuallyKnown != "" {
					fmt.Printf("    (Manually known Content-ID from EML for this part: %s)\n", expectedCIDManuallyKnown)
				}
				i++
			}
		}
		fmt.Println("--- End of Analysis ---")
		fmt.Println()
	}
}
