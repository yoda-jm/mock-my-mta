package main

import (
	"os"
	"path/filepath"

	"mock-my-mta/email"
	"mock-my-mta/log"
	"mock-my-mta/storage"
)

func testFind(store storage.Storage) {
	for _, direction := range []storage.SortType{storage.Ascending, storage.Descending} {
		so := storage.SortOption{
			Field:     storage.SortDateField,
			Direction: direction,
		}
		searchPattern := "Email 1"
		log.Logf(log.DEBUG, "searching for emails with subject %q", searchPattern)
		mo := email.MatchOption{Field: email.MatchSubjectField, Type: email.ExactMatch, CaseSensitive: true}
		foundEmails, err := store.Find(mo, so, searchPattern)
		if err != nil {
			log.Logf(log.WARNING, "error while searching for emails: %v", err)
			continue
		}
		log.Logf(log.DEBUG, "found %d emails", len(foundEmails))
		for _, id := range foundEmails {
			log.Logf(log.DEBUG, "found email with ID: %v", id)
		}
	}
}

func testPrintAll(store storage.Storage) {
	for _, direction := range []storage.SortType{storage.Ascending, storage.Descending} {
		so := storage.SortOption{
			Field:     storage.SortDateField,
			Direction: direction,
		}
		so.Direction = direction
		// Get email data by UUID
		log.Logf(log.DEBUG, "getting all emails from storage")
		uuids, err := storage.GetAll(store, so)
		if err != nil {
			log.Logf(log.WARNING, "error while getting all emails: %v", err)
		} else if len(uuids) == 0 {
			log.Logf(log.DEBUG, "no UUID found in storage")
		} else {
			log.Logf(log.DEBUG, "found %d emails", len(uuids))
			for _, uuid := range uuids {
				emailData, err := store.Get(uuid)
				if err != nil {
					log.Logf(log.WARNING, "email data not found for UUID:%v", emailData.ID)
				}

				log.Logf(log.DEBUG, "retrieved email data for UUID:%v, received at %v", emailData.ID, emailData.ReceivedTime)
				log.Logf(log.DEBUG, "subject: %v", emailData.Email.GetSubject())
				versions := emailData.Email.GetVersions()
				log.Logf(log.DEBUG, "body verions count: %v", len(versions))
				for _, version := range versions {
					body, err := emailData.Email.GetBody(version)
					if err != nil {
						log.Logf(log.DEBUG, "cannot get body for version %q", version)
					} else {
						log.Logf(log.DEBUG, "- %q: %v bytes", version, len(body))
					}
				}
				attachmentIDs := emailData.Email.GetAttachments()
				if len(attachmentIDs) > 0 {
					log.Logf(log.DEBUG, "%v attachments", len(attachmentIDs))
					for _, attachmentID := range attachmentIDs {
						attachment, found := emailData.Email.GetAttachment(attachmentID)
						if found {
							log.Logf(log.DEBUG, "- attachment (id=%v, type=%v, filename=%q, length=%v)", attachment.GetID(), attachment.GetMediaType(), attachment.GetFilename(), len(attachment.GetContent()))
						} else {
							log.Logf(log.DEBUG, "email attachment not found for UUID:%v", attachmentID)
						}
					}
				}
			}
		}
	}
}

func loadTestData(store storage.Storage, testDataDir string) error {
	files, err := os.ReadDir(testDataDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".txt" {
			continue
		}

		filePath := filepath.Join(testDataDir, file.Name())
		log.Logf(log.INFO, "loading file %q", filePath)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		err = store.Set(content)
		if err != nil {
			return err
		}
	}

	return nil
}
