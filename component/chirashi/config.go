package chirashi

// ParticleConfig represents the complete configuration for a GPU particle system
type ParticleConfig struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Image       ImageConfig     `yaml:"image"`
	Emitter     EmitterConfig   `yaml:"emitter"`
	Animation   AnimationConfig `yaml:"animation"`
	Spawn       SpawnConfig     `yaml:"spawn"`
}

// ImageConfig defines image source parameters
type ImageConfig struct {
	ImageFrom string `yaml:"image_from"`
	ImageID   int    `yaml:"image_id"`
}

// EmitterConfig defines the emitter properties
type EmitterConfig struct {
	X float32 `yaml:"x"`
	Y float32 `yaml:"y"`
}

// AnimationConfig defines all animation parameters with simplified lerp-based approach
type AnimationConfig struct {
	Duration DurationConfig `yaml:"duration"`
	Position PositionConfig `yaml:"position"`
	Alpha    PropertyConfig `yaml:"alpha"`
	Scale    PropertyConfig `yaml:"scale"`
	Rotation PropertyConfig `yaml:"rotation"`
	Color    *ColorConfig   `yaml:"color,omitempty"`
}

// DurationConfig defines particle lifetime with optional randomization
type DurationConfig struct {
	Value float32     `yaml:"value"`
	Range *RangeFloat `yaml:"range,omitempty"`
}

// RangeFloat defines a min/max range for randomization
type RangeFloat struct {
	Min float32 `yaml:"min"`
	Max float32 `yaml:"max"`
}

// PositionConfig defines position animation parameters
type PositionConfig struct {
	Type string `yaml:"type,omitempty"` // "cartesian" (default) or "polar"

	// Cartesian mode (simple)
	StartX *RangeFloat `yaml:"start_x,omitempty"`
	EndX   *RangeFloat `yaml:"end_x,omitempty"`
	StartY *RangeFloat `yaml:"start_y,omitempty"`
	EndY   *RangeFloat `yaml:"end_y,omitempty"`

	// Cartesian mode (sequence) - multi-step position tweens
	X *PropertyConfig `yaml:"x,omitempty"` // X axis sequence
	Y *PropertyConfig `yaml:"y,omitempty"` // Y axis sequence

	// Polar mode
	Angle    *RangeFloat `yaml:"angle,omitempty"`    // Radians (0 to 2π for full circle)
	Distance *RangeFloat `yaml:"distance,omitempty"` // Distance from emitter

	Easing string `yaml:"easing"`
}

// PropertyConfig defines an animation with easing.
// Supports two modes:
//   - Simple mode: Start/End/Easing (existing, backward compatible)
//   - Sequence mode: Type="sequence" with Steps (multi-step tween chains)
type PropertyConfig struct {
	// Simple mode (default)
	Start  float32 `yaml:"start"`
	End    float32 `yaml:"end"`
	Easing string  `yaml:"easing"`

	// Multi-step mode
	Type  string       `yaml:"type,omitempty"`  // "sequence" enables multi-step
	Steps []StepConfig `yaml:"steps,omitempty"` // Steps for sequence mode
}

// StepConfig defines one step in a multi-step animation sequence
type StepConfig struct {
	From       float32     `yaml:"from"`
	FromRange  *RangeFloat `yaml:"from_range,omitempty"`
	To         float32     `yaml:"to"`
	ToRange    *RangeFloat `yaml:"to_range,omitempty"`
	IsRelative bool        `yaml:"is_relative,omitempty"`
	Duration   float32     `yaml:"duration"`
	Easing     string      `yaml:"easing"`
}

// IsSequence returns true if this config uses multi-step sequence mode
func (c *PropertyConfig) IsSequence() bool {
	return c.Type == "sequence" && len(c.Steps) > 0
}

// ColorConfig defines color animation (RGB values 0-1)
type ColorConfig struct {
	StartR float32 `yaml:"start_r"`
	StartG float32 `yaml:"start_g"`
	StartB float32 `yaml:"start_b"`
	EndR   float32 `yaml:"end_r"`
	EndG   float32 `yaml:"end_g"`
	EndB   float32 `yaml:"end_b"`
	Easing string  `yaml:"easing"`
}

// SpawnConfig defines particle spawning parameters
type SpawnConfig struct {
	Interval          int  `yaml:"interval"`
	ParticlesPerSpawn int  `yaml:"particles_per_spawn"`
	MaxParticles      int  `yaml:"max_particles"`
	IsLoop            bool `yaml:"is_loop"`
	LifeTime          int  `yaml:"life_time,omitempty"`
}
