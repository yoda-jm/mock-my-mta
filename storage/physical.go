package storage

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Physical interface {
	List() ([]uuid.UUID, error)
	Read(uuid.UUID) (*EmailData, error)
	Write(*EmailData) error
	Delete(uuid.UUID) error
}

func newPhysical(physicalStr string) (Physical, error) {
	if strings.HasPrefix(physicalStr, "filesystem://") {
		return newPhysicalFilesystem(strings.TrimPrefix(physicalStr, "filesystem://"))
	} else {
		return nil, fmt.Errorf("unknow physical storage backend: %q", physicalStr)
	}
}
