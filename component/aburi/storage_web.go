//go:build js

package aburi

import (
	"fmt"
	"log"

	"gopkg.in/yaml.v3"
)

// WebStorage implements ParticleStorage for Web (WASM) environment
type WebStorage struct{}

// NewStorage creates a new storage instance for web
func NewStorage() ParticleStorage {
	return &WebStorage{}
}

func (s *WebStorage) Save(path string, config *ParticleConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// In Web environment, we can't write to file system directly.
	// For now, we'll log the YAML to console so user can copy it.
	log.Printf("--- SAVE YAML (%s) ---\n%s\n------------------------", path, string(data))

	return nil
}

func (s *WebStorage) Load(path string) (*ParticleConfig, error) {
	// In Web environment, we can't read arbitrary files.
	return nil, fmt.Errorf("loading from file path not supported in web version yet")
}

func (s *WebStorage) List(pattern string) ([]string, error) {
	// Cannot glob in web environment easily without a file server index
	return []string{}, nil
}
