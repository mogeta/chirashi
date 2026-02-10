//go:build js

package chirashi

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// WebStorage implements ParticleStorage for Web (WASM) environment
type WebStorage struct{}

// NewStorage creates a new storage instance for web
func NewStorage() ParticleStorage {
	return &WebStorage{}
}

// Save validates serialization but returns an error because arbitrary file writes are not supported on web.
func (s *WebStorage) Save(path string, config *ParticleConfig) error {
	if _, err := yaml.Marshal(config); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return fmt.Errorf("saving to file path is not supported in web version yet: %s", path)
}

// Load returns an error because arbitrary file reads are not supported on web.
func (s *WebStorage) Load(path string) (*ParticleConfig, error) {
	// In Web environment, we can't read arbitrary files.
	return nil, fmt.Errorf("loading from file path not supported in web version yet")
}

// List returns an empty result on web where globbing local files is not available.
func (s *WebStorage) List(pattern string) ([]string, error) {
	// Cannot glob in web environment easily without a file server index
	return []string{}, nil
}
