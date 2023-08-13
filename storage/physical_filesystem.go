package storage

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"mock-my-mta/log"
)

type physicalFilesystem struct {
	folder string
}

func newPhysicalFilesystem(folder string) (Physical, error) {
	return &physicalFilesystem{
		folder: folder,
	}, nil
}

func (pf *physicalFilesystem) List() ([]uuid.UUID, error) {
	files, err := ioutil.ReadDir(pf.folder)
	if err != nil {
		return nil, err
	}

	var uuids []uuid.UUID
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		uuidStr := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		uuid, err := uuid.Parse(uuidStr)
		if err != nil {
			return nil, err
		}
		uuids = append(uuids, uuid)
	}
	return uuids, nil
}

func (pf *physicalFilesystem) Read(uuid uuid.UUID) (*EmailData, error) {
	filePath := filepath.Join(pf.folder, uuid.String()+".json")
	log.Logf(log.DEBUG, "loading file %v", filePath)
	content, err := ioutil.ReadFile(filePath)
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

func (pf *physicalFilesystem) Write(emailData *EmailData) error {
	data, err := json.Marshal(*emailData)
	if err != nil {
		return err
	}

	filePath := filepath.Join(pf.folder, emailData.ID.String()+".json")
	err = ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (pf *physicalFilesystem) Delete(uuid uuid.UUID) error {
	filePath := filepath.Join(pf.folder, uuid.String()+".json")
	log.Logf(log.DEBUG, "deleting file %v", filePath)
	return os.Remove(filePath)
}
