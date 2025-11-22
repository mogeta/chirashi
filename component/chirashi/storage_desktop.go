//go:build !js

package chirashi

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileStorage implements ParticleStorage using the local file system
type FileStorage struct{}

// NewStorage creates a new storage instance for desktop
func NewStorage() ParticleStorage {
	return &FileStorage{}
}

func (s *FileStorage) Save(path string, config *ParticleConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", path, err)
	}

	return nil
}

func (s *FileStorage) Load(path string) (*ParticleConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config ParticleConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config %s: %w", path, err)
	}

	return &config, nil
}

func (s *FileStorage) List(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
