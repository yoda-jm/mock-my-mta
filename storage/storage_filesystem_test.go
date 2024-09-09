package storage

import (
	"net/mail"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// test that we can create a new storage layer
func TestCreateStorage(t *testing.T) {
	// new temporary test folder
	testCases := []struct {
		storageType string
		isError     bool
	}{
		{"eml", false},
		{"mailhog", false},
		{"invalid", true},
	}
	for _, data := range testCases {
		t.Run(data.storageType, func(t *testing.T) {
			tmpFolder := t.TempDir()
			storage, err := newFilesystemStorage(tmpFolder, data.storageType)
			// check if we got the expected error
			if data.isError && err == nil {
				t.Errorf("expected error, got nil (storage type %s)", data.storageType)
			}
			if !data.isError && err != nil {
				t.Errorf("expected no error, got %v (storage type %s)", err, data.storageType)
			}
			// check that storage is not nil when there is no error
			if !data.isError && storage == nil {
				t.Errorf("expected storage, got nil (storage type %s)", data.storageType)
			}
		})
	}
}

func TestLoadCreatesFolder(t *testing.T) {
	tmpFolder := t.TempDir()
	storageFolder := filepath.Join(tmpFolder, "eml")
	storage, err := newFilesystemStorage(storageFolder, "eml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// load the storage
	err = storage.load(nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// check that storageFolder exists and is a folder
	if stat, err := os.Stat(storageFolder); err != nil || !stat.IsDir() {
		t.Errorf("expected folder to be created, got %v", err)
	}
}

var simpleEmail = `From: from@example.com
To: to1@example.com
Subject: Test email

This is the body of the email.`

func TestEMLStorage(t *testing.T) {
	tmpFolder := t.TempDir()
	storageFolder := filepath.Join(tmpFolder, "eml")
	storage, err := newFilesystemStorage(storageFolder, "eml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// load the storage
	err = storage.load(nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// create a new email
	message, err := mail.ReadMessage(strings.NewReader(simpleEmail))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// set the email
	const emailID = "simple-email"
	err = storage.setWithID(emailID, message)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// check that the email file exists
	emailFile := filepath.Join(storageFolder, emailID+".eml")
	if _, err := os.Stat(emailFile); err != nil {
		t.Errorf("expected email file to exist, got %v", err)
	}
	// check that the ID is in the list
	ids, err := storage.getAllEmailIDs()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(ids) != 1 || ids[0] != emailID {
		t.Errorf("expected email ID in list, got %v", ids)
	}
	// check that the retrieval of the email works and is the same
	retrievedMessage, err := storage.getRawBody(emailID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// check that split/sort the email lines returns the same arrays
	// this is required because the mail parser is reordering the headers
	sortedLinesExpected := sort.StringSlice(strings.Split(simpleEmail, "\n"))
	sortedLinesRetrieved := sort.StringSlice(strings.Split(string(retrievedMessage), "\n"))
	if len(sortedLinesExpected) != len(sortedLinesRetrieved) {
		t.Errorf("expected same number of lines, got %d and %d", len(sortedLinesExpected), len(sortedLinesRetrieved))
	}
	for i, line := range sortedLinesExpected {
		if line != sortedLinesRetrieved[i] {
			t.Errorf("expected same line, got %s and %s", line, sortedLinesRetrieved[i])
		}
	}
}

func TestMailhogStorage(t *testing.T) {
	tmpFolder := t.TempDir()
	storageFolder := filepath.Join(tmpFolder, "mailhog")
	storage, err := newFilesystemStorage(storageFolder, "mailhog")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// load the storage
	err = storage.load(nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// create a new email
	message, err := mail.ReadMessage(strings.NewReader(simpleEmail))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// set the email
	const emailID = "simple-email"
	err = storage.setWithID(emailID, message)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// check that the email file exists
	emailFile := filepath.Join(storageFolder, emailID+"@mailhog.example")
	if _, err := os.Stat(emailFile); err != nil {
		t.Errorf("expected email file to exist, got %v", err)
	}
	// check that the ID is in the list
	ids, err := storage.getAllEmailIDs()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(ids) != 1 || ids[0] != emailID {
		t.Errorf("expected email ID in list, got %v", ids)
	}
	// FIXME: check that the email content starts with a mailhog header
	// check that the retrieval of the email works and is the same
	retrievedMessage, err := storage.getRawBody(emailID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// check that split/sort the email lines returns the same arrays
	// this is required because the mail parser is reordering the headers
	sortedLinesExpected := sort.StringSlice(strings.Split(simpleEmail, "\n"))
	sortedLinesRetrieved := sort.StringSlice(strings.Split(string(retrievedMessage), "\n"))
	if len(sortedLinesExpected) != len(sortedLinesRetrieved) {
		t.Errorf("expected same number of lines, got %d and %d", len(sortedLinesExpected), len(sortedLinesRetrieved))
	}
	for i, line := range sortedLinesExpected {
		if line != sortedLinesRetrieved[i] {
			t.Errorf("expected same line, got %s and %s", line, sortedLinesRetrieved[i])
		}
	}
}
