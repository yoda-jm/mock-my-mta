package storage

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"mock-my-mta/email"
	"mock-my-mta/log"
)

type Engine struct {
	storages []PhysicalLayer
}

// check that the Engine implements the Storage interface
var _ Storage = &Engine{}

// unimplementedMethodInLayer type
type unimplementedMethodInLayer struct{}

func (unimplementedMethodInLayer) Error() string {
	return "unimplemented method in layer"
}

type EngineLayerConfig struct {
	Type       string
	Parameters map[string]string
}

type EngineConfig struct {
	Storages []EngineLayerConfig
}

func NewEngine(config EngineConfig) (*Engine, error) {
	engine := &Engine{}
	// construct storages
	for _, storage := range config.Storages {
		switch storage.Type {
		case "MEMORY":
			physical, err := newMemoryStorage()
			if err != nil {
				return nil, err
			}
			engine.storages = append(engine.storages, physical)
		case "MMM":
			physical, err := newMMMStorage()
			if err != nil {
				return nil, err
			}
			engine.storages = append(engine.storages, physical)
		default:
			return nil, fmt.Errorf("unknown storage type")
		}
	}
	// load the storages in reverse order
	rootLayer := engine.storages[len(engine.storages)-1]
	for i := len(engine.storages) - 1; i >= 0; i-- {
		storage := engine.storages[i]
		parameters := config.Storages[i].Parameters
		if i == len(engine.storages)-1 {
			// the root layer is the last one
			if err := storage.Populate(nil, parameters); err != nil {
				return nil, err
			}
		} else {
			if err := storage.Populate(rootLayer, parameters); err != nil {
				return nil, err
			}
		}
	}
	return engine, nil
}

// Get implements Storage.
func (e *Engine) Get(id uuid.UUID) (*EmailData, error) {
	for _, storage := range e.storages {
		email, err := storage.Read(id)
		if err != nil {
			// skip if unimplemented in the layer
			if _, ok := err.(unimplementedMethodInLayer); ok {
				continue
			}
			return nil, err
		}
		return email, nil
	}
	return nil, fmt.Errorf("no storage layer implemented Get")
}

// Set implements Storage.
func (e *Engine) Set(message []byte) error {
	email, err := email.Parse(message)
	if err != nil {
		return err
	}
	emailData := EmailData{
		ID:           uuid.New(),
		ReceivedTime: time.Now(),
		Email:        email,
	}

	log.Logf(log.INFO, "writting email %v", emailData.ID)
	for _, storage := range e.storages {
		if err := storage.Write(&emailData); err != nil {
			// skip if unimplemented in the layer
			if _, ok := err.(unimplementedMethodInLayer); ok {
				continue
			}
			return err
		}
	}
	return nil
}

// Find implements Storage.
func (e *Engine) Find(matchOptions email.MatchOption, sortOptions SortOption, value string) ([]uuid.UUID, error) {
	for _, storage := range e.storages {
		ids, err := storage.Find(matchOptions, sortOptions, value)
		if err != nil {
			// skip if unimplemented in the layer
			if _, ok := err.(unimplementedMethodInLayer); ok {
				continue
			}
			return nil, err
		}
		return ids, nil
	}
	return nil, fmt.Errorf("no storage layer implemented Find")
}

// Delete implements Storage.
func (e *Engine) Delete(id uuid.UUID) error {
	log.Logf(log.INFO, "deleting email %v", id)
	for _, storage := range e.storages {
		if err := storage.Delete(id); err != nil {
			// skip if unimplemented in the layer
			if _, ok := err.(unimplementedMethodInLayer); ok {
				continue
			}
			return err
		}
	}
	return nil
}
