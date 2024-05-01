package storage

type StorageLayerConfiguration struct {
	Type       string            `json:"type"`
	Parameters map[string]string `json:"parameters"`
}
