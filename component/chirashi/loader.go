package chirashi

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
		assetsPath := filepath.Join("assets", "particles", name)
		return l.LoadConfig(assetsPath)
	}
	assetsPath := filepath.Join("assets", "particles", name+".yaml")
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

	if config.Spawn.Interval <= 0 {
		return fmt.Errorf("interval must be greater than 0")
	}

	dur := config.Animation.Duration
	if dur.Range != nil {
		if dur.Range.Min <= 0 {
			return fmt.Errorf("animation.duration.range.min must be greater than 0")
		}
	} else if dur.Value <= 0 {
		return fmt.Errorf("animation.duration.value must be greater than 0")
	}

	switch config.Emitter.Shape.Type {
	case "", "point", "circle", "box", "line":
	default:
		return fmt.Errorf("emitter.shape.type must be point, circle, box, or line")
	}
	if vector := config.Emitter.Vector; vector != nil {
		switch vector.Type {
		case "rect", "polyline":
		default:
			return fmt.Errorf("emitter.vector.type must be rect or polyline")
		}
		switch vector.Placement {
		case "", "fill", "surface":
		default:
			return fmt.Errorf("emitter.vector.placement must be fill or surface")
		}
		switch vector.Type {
		case "rect":
			if vector.Rect == nil {
				return fmt.Errorf("emitter.vector.rect is required")
			}
			if vector.Rect.Width <= 0 {
				return fmt.Errorf("emitter.vector.rect.width must be greater than 0")
			}
			if vector.Rect.Height <= 0 {
				return fmt.Errorf("emitter.vector.rect.height must be greater than 0")
			}
		case "polyline":
			if vector.Placement != "" && vector.Placement != "surface" {
				return fmt.Errorf("emitter.vector.placement must be surface for polyline")
			}
			if vector.Polyline == nil {
				return fmt.Errorf("emitter.vector.polyline is required")
			}
			if len(vector.Polyline.Points) < 2 {
				return fmt.Errorf("emitter.vector.polyline.points must contain at least 2 points")
			}
			switch vector.Polyline.Interpolation {
			case "", "linear", "quadratic":
			default:
				return fmt.Errorf("emitter.vector.polyline.interpolation must be linear or quadratic")
			}
			if vector.Polyline.CurveSteps < 0 {
				return fmt.Errorf("emitter.vector.polyline.curve_steps must be greater than or equal to 0")
			}
			if vector.Polyline.Interpolation == "quadratic" {
				if len(vector.Polyline.Points) < 3 || len(vector.Polyline.Points)%2 == 0 {
					return fmt.Errorf("emitter.vector.polyline.points must alternate anchor/control/anchor for quadratic interpolation")
				}
				if vector.Polyline.Closed {
					return fmt.Errorf("emitter.vector.polyline.closed is not supported for quadratic interpolation")
				}
			}
		}
	}
	switch config.Emitter.Space {
	case EmitterSpaceDefault, EmitterSpaceLocal, EmitterSpaceWorld:
	default:
		return fmt.Errorf("emitter.space must be local or world")
	}
	if trail := config.Trail; trail != nil {
		switch trail.Mode {
		case "", "emitter", "particle":
		default:
			return fmt.Errorf("trail.mode must be emitter or particle")
		}
		switch trail.Space {
		case "", "local", "world":
		default:
			return fmt.Errorf("trail.space must be local or world")
		}
		if trail.MaxPoints != 0 && trail.MaxPoints < 2 {
			return fmt.Errorf("trail.max_points must be 2 or greater, or 0 to use the default")
		}
		if trail.MinPointDistance < 0 {
			return fmt.Errorf("trail.min_point_distance must be greater than or equal to 0")
		}
		if trail.MaxPointAge < 0 {
			return fmt.Errorf("trail.max_point_age must be greater than or equal to 0")
		}
	}

	if flow := config.Animation.Position.Flow; flow != nil {
		switch flow.Type {
		case "", "curl":
		default:
			return fmt.Errorf("animation.position.flow.type must be curl")
		}
		if flow.Strength != nil && flow.Strength.Min > flow.Strength.Max {
			return fmt.Errorf("animation.position.flow.strength.min must be less than or equal to max")
		}
		if flow.Scale < 0 {
			return fmt.Errorf("animation.position.flow.scale must be greater than or equal to 0")
		}
		if flow.Octaves < 0 || flow.Octaves > 3 {
			return fmt.Errorf("animation.position.flow.octaves must be within [0,3]")
		}
		if flow.Persistence < 0 {
			return fmt.Errorf("animation.position.flow.persistence must be greater than or equal to 0")
		}
		if flow.TimeScale < 0 {
			return fmt.Errorf("animation.position.flow.time_scale must be greater than or equal to 0")
		}
		if flow.Drag < 0 || flow.Drag > 1 {
			return fmt.Errorf("animation.position.flow.drag must be within [0,1]")
		}
		switch flow.Space {
		case "", "local", "world":
		default:
			return fmt.Errorf("animation.position.flow.space must be local or world")
		}
		if flow.BoundRadius < 0 {
			return fmt.Errorf("animation.position.flow.bound_radius must be greater than or equal to 0")
		}
	}

	if config.Emitter.Shape.Radius != nil && config.Emitter.Shape.Radius.Min > config.Emitter.Shape.Radius.Max {
		return fmt.Errorf("emitter.shape.radius.min must be less than or equal to max")
	}

	if config.Emitter.Shape.Type == "circle" && config.Emitter.Shape.StartAngle == 0 && config.Emitter.Shape.EndAngle == 0 {
		config.Emitter.Shape.EndAngle = 6.2831855
	}

	return nil
}
