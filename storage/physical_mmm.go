package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"mock-my-mta/email"
	"mock-my-mta/log"
)

type mmmPhysicalStorage struct {
	folder string
}

// check that the mmmPhysicalStorage implements the PhysicalLayer interface
var _ PhysicalLayer = &mmmPhysicalStorage{}

func newMMMStorage() (*mmmPhysicalStorage, error) {
	return &mmmPhysicalStorage{}, nil
}

// Delete implements PhysicalLayer.
func (mmm *mmmPhysicalStorage) Delete(id uuid.UUID) error {
	filePath := filepath.Join(mmm.folder, id.String()+".json")
	log.Logf(log.DEBUG, "deleting file %v", filePath)
	return os.Remove(filePath)
}

// Find implements PhysicalLayer.
func (*mmmPhysicalStorage) Find(matchOptions email.MatchOption, sortOptions SortOption, value string) ([]uuid.UUID, error) {
	return nil, unimplementedMethodInLayer{}
}

// List implements PhysicalLayer.
func (mmm *mmmPhysicalStorage) List() ([]uuid.UUID, error) {
	files, err := os.ReadDir(mmm.folder)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(files))
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		idStr := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// Load implements PhysicalLayer.
func (mmm *mmmPhysicalStorage) Populate(underlying PhysicalLayer, parameters map[string]string) error {
	log.Logf(log.INFO, "populating MMM layer")
	folder, ok := parameters["folder"]
	if !ok {
		return fmt.Errorf("missing folder parameter")
	}
	mmm.folder = folder
	if underlying != nil {
		// clear folder directory if it contains any files
		ids, err := mmm.List()
		if err != nil {
			return err
		}
		for _, id := range ids {
			if err := mmm.Delete(id); err != nil {
				return err
			}
		}

		// load data from underlying storage
		underlyingIds, err := mmm.List()
		if err != nil {
			return err
		}
		for _, id := range underlyingIds {
			emailData, err := mmm.Read(id)
			if err != nil {
				return err
			}
			if err := mmm.Write(emailData); err != nil {
				return err
			}
		}
	}
	return nil
}

// Read implements PhysicalLayer.
func (mmm *mmmPhysicalStorage) Read(id uuid.UUID) (*EmailData, error) {
	filePath := filepath.Join(mmm.folder, id.String()+".json")
	log.Logf(log.DEBUG, "loading file %v", filePath)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var emailData EmailData
	err = json.Unmarshal(content, &emailData)
	if err != nil {
		return nil, err
	}

	return &emailData, nil
}

// Write implements PhysicalLayer.
func (mmm *mmmPhysicalStorage) Write(emailData *EmailData) error {
	data, err := json.Marshal(*emailData)
	if err != nil {
		return err
	}

	filePath := filepath.Join(mmm.folder, emailData.ID.String()+".json")
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
