package particle

// ParticleConfig represents the complete configuration for a particle system
type ParticleConfig struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Image       ImageConfig      `yaml:"image"`
	Emitter     EmitterConfig    `yaml:"emitter"`
	Movement    MovementConfig   `yaml:"movement"`
	Appearance  AppearanceConfig `yaml:"appearance"`
	Spawn       SpawnConfig      `yaml:"spawn"`
}

// ImageConfig defines image source parameters
type ImageConfig struct {
	ImageFrom string `yaml:"image_from"`
	ImageID   int    `yaml:"image_id"`
}

// EmitterConfig defines the emitter properties
type EmitterConfig struct {
	Position PositionConfig `yaml:"position"`
}

// PositionConfig defines position parameters
type PositionConfig struct {
	X float64 `yaml:"x"`
	Y float64 `yaml:"y"`
}

// MovementConfig defines movement animations
type MovementConfig struct {
	X TweenConfig `yaml:"x"`
	Y TweenConfig `yaml:"y"`
}

// TweenConfig defines a tween animation configuration
type TweenConfig struct {
	Type  string      `yaml:"type"` // "sequence" or "single"
	Steps []TweenStep `yaml:"steps"`
}

// TweenStep defines a single tween step
type TweenStep struct {
	From     float64    `yaml:"from"`
	To       float64    `yaml:"to"`
	ToRange  *RangeData `yaml:"to_range,omitempty"` // Optional random range for 'to' value
	Duration float64    `yaml:"duration"`
	Easing   string     `yaml:"easing"` // "OutCirc", "InBack", etc
}

// RangeData defines a random value range
type RangeData struct {
	Min float64 `yaml:"min"`
	Max float64 `yaml:"max"`
}

// AppearanceConfig defines visual appearance animations
type AppearanceConfig struct {
	Alpha    TweenConfig `yaml:"alpha"`
	Rotation TweenConfig `yaml:"rotation,omitempty"`
	Scale    TweenConfig `yaml:"scale,omitempty"`
}

// SpawnConfig defines particle spawning parameters
type SpawnConfig struct {
	Interval          int  `yaml:"interval"`
	ParticlesPerSpawn int  `yaml:"particles_per_spawn"`
	MaxParticles      int  `yaml:"max_particles"`
	IsLoop            bool `yaml:"is_loop,omitempty"`
	LifeTime          int  `yaml:"life_time,omitempty"`
}
