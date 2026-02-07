package aburi

import (
	"fmt"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// ConfigLoader manages particle configuration loading and caching
type ConfigLoader struct {
	configs map[string]*ParticleConfig
	storage ParticleStorage
	mutex   sync.RWMutex
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		configs: make(map[string]*ParticleConfig),
		storage: NewStorage(),
	}
}

// LoadConfig loads a particle configuration from a file path
func (l *ConfigLoader) LoadConfig(path string) (*ParticleConfig, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Check cache first
	if config, exists := l.configs[path]; exists {
		return config, nil
	}

	// Load from storage
	config, err := l.storage.Load(path)
	if err != nil {
		return nil, err
	}

	// Validate configuration
	if err := l.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}

	// Cache the configuration
	l.configs[path] = config
	return config, nil
}

// SaveConfig saves a particle configuration to a file path
func (l *ConfigLoader) SaveConfig(path string, config *ParticleConfig) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if err := l.storage.Save(path, config); err != nil {
		return err
	}

	// Update cache
	l.configs[path] = config
	return nil
}

// LoadConfigFromBytes loads a particle configuration from byte data
func (l *ConfigLoader) LoadConfigFromBytes(data []byte, name string) (*ParticleConfig, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Check cache first
	if config, exists := l.configs[name]; exists {
		return config, nil
	}

	var config ParticleConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config %s: %w", name, err)
	}

	// Validate configuration
	if err := l.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", name, err)
	}

	// Cache the configuration
	l.configs[name] = &config
	return &config, nil
}

// LoadFromAssets loads a particle configuration from assets directory
func (l *ConfigLoader) LoadFromAssets(name string) (*ParticleConfig, error) {
	// Check if name already has .yaml extension
	if filepath.Ext(name) == ".yaml" {
		assetsPath := filepath.Join("assets", "particles", "aburi", name)
		return l.LoadConfig(assetsPath)
	}
	assetsPath := filepath.Join("assets", "particles", "aburi", name+".yaml")
	return l.LoadConfig(assetsPath)
}

// GetConfig retrieves a cached configuration by path
func (l *ConfigLoader) GetConfig(path string) *ParticleConfig {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return l.configs[path]
}

// ClearCache clears the configuration cache
func (l *ConfigLoader) ClearCache() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.configs = make(map[string]*ParticleConfig)
}

// ListConfigs returns a list of configuration files matching the pattern
func (l *ConfigLoader) ListConfigs(pattern string) ([]string, error) {
	return l.storage.List(pattern)
}

// validateConfig validates a particle configuration
func (l *ConfigLoader) validateConfig(config *ParticleConfig) error {
	if config.Name == "" {
		return fmt.Errorf("name is required")
	}

	if config.Spawn.MaxParticles <= 0 {
		return fmt.Errorf("max_particles must be greater than 0")
	}

	if config.Spawn.ParticlesPerSpawn <= 0 {
		return fmt.Errorf("particles_per_spawn must be greater than 0")
	}

	if config.Spawn.Interval < 0 {
		return fmt.Errorf("interval must be non-negative")
	}

	if config.Animation.Duration.Value <= 0 {
		return fmt.Errorf("animation.duration.value must be greater than 0")
	}

	return nil
}
