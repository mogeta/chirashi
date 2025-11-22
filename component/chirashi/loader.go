package chirashi

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// ConfigLoader manages particle configuration loading and caching
type ConfigLoader struct {
	configs map[string]*ParticleConfig
	mutex   sync.RWMutex
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		configs: make(map[string]*ParticleConfig),
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

	// Load from file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config ParticleConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config %s: %w", path, err)
	}

	// Validate configuration
	if err := l.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}

	// Cache the configuration
	l.configs[path] = &config
	return &config, nil
}

// SaveConfig saves a particle configuration to a file path
func (l *ConfigLoader) SaveConfig(path string, config *ParticleConfig) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", path, err)
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
		assetsPath := filepath.Join("assets", "particles", "effects", name)
		return l.LoadConfig(assetsPath)
	} else {
		assetsPath := filepath.Join("assets", "particles", "effects", name+".yaml")
		return l.LoadConfig(assetsPath)
	}
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

	// Validate movement tweens
	if err := l.validateTweenConfig(&config.Movement.X, "movement.x"); err != nil {
		return err
	}
	if err := l.validateTweenConfig(&config.Movement.Y, "movement.y"); err != nil {
		return err
	}

	// Validate appearance tweens
	if err := l.validateTweenConfig(&config.Appearance.Alpha, "appearance.alpha"); err != nil {
		return err
	}

	return nil
}

// validateTweenConfig validates a tween configuration
func (l *ConfigLoader) validateTweenConfig(tween *TweenConfig, fieldName string) error {
	if tween.Type != "single" && tween.Type != "sequence" {
		return fmt.Errorf("%s.type must be 'single' or 'sequence'", fieldName)
	}

	if len(tween.Steps) == 0 {
		return fmt.Errorf("%s.steps cannot be empty", fieldName)
	}

	for i, step := range tween.Steps {
		if step.Duration <= 0 {
			return fmt.Errorf("%s.steps[%d].duration must be greater than 0", fieldName, i)
		}

		// Validate easing function name
		if !l.isValidEasing(step.Easing) {
			return fmt.Errorf("%s.steps[%d].easing '%s' is not valid", fieldName, i, step.Easing)
		}

		// Validate range data if specified
		if step.ToRange != nil {
			if step.ToRange.Min > step.ToRange.Max {
				return fmt.Errorf("%s.steps[%d].to_range.min must be <= max", fieldName, i)
			}
		}
	}

	return nil
}

// isValidEasing checks if the easing function name is valid
func (l *ConfigLoader) isValidEasing(easing string) bool {
	validEasings := map[string]bool{
		"Linear":     true,
		"InQuad":     true,
		"OutQuad":    true,
		"InOutQuad":  true,
		"InCubic":    true,
		"OutCubic":   true,
		"InOutCubic": true,
		"InQuart":    true,
		"OutQuart":   true,
		"InOutQuart": true,
		"InQuint":    true,
		"OutQuint":   true,
		"InOutQuint": true,
		"InSine":     true,
		"OutSine":    true,
		"InOutSine":  true,
		"InExpo":     true,
		"OutExpo":    true,
		"InOutExpo":  true,
		"InCirc":     true,
		"OutCirc":    true,
		"InOutCirc":  true,
		"InBack":     true,
		"OutBack":    true,
		"InOutBack":  true,
	}

	return validEasings[easing]
}
