package chirashi

// ParticleConfig represents the complete configuration for a GPU particle system
type ParticleConfig struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Image       ImageConfig     `yaml:"image"`
	Emitter     EmitterConfig   `yaml:"emitter"`
	Animation   AnimationConfig `yaml:"animation"`
	Trail       *TrailConfig    `yaml:"trail,omitempty"`
	Spawn       SpawnConfig     `yaml:"spawn"`
}

// ImageConfig defines image source parameters
type ImageConfig struct {
	ImageFrom string `yaml:"image_from"`
	ImageID   int    `yaml:"image_id"`
}

type EmitterSpaceMode string

const (
	EmitterSpaceDefault EmitterSpaceMode = ""
	EmitterSpaceLocal   EmitterSpaceMode = "local"
	EmitterSpaceWorld   EmitterSpaceMode = "world"
)

// EmitterConfig defines the emitter properties
type EmitterConfig struct {
	X      float32              `yaml:"x"`
	Y      float32              `yaml:"y"`
	Space  EmitterSpaceMode     `yaml:"space,omitempty"` // local (default) or world
	Shape  EmitterShapeConfig   `yaml:"shape,omitempty"`
	Vector *EmitterVectorConfig `yaml:"vector,omitempty"`
}

// EmitterShapeConfig defines where particles are spawned relative to the emitter origin.
type EmitterShapeConfig struct {
	Type       string      `yaml:"type,omitempty"` // point (default), circle, box, line
	Radius     *RangeFloat `yaml:"radius,omitempty"`
	StartAngle float32     `yaml:"start_angle,omitempty"` // Radians, circle only
	EndAngle   float32     `yaml:"end_angle,omitempty"`   // Radians, circle only
	Width      float32     `yaml:"width,omitempty"`
	Height     float32     `yaml:"height,omitempty"`
	Length     float32     `yaml:"length,omitempty"`
	Rotation   float32     `yaml:"rotation,omitempty"` // Radians, used by box/line
	FromEdge   bool        `yaml:"from_edge,omitempty"`
}

// EmitterVectorConfig defines a vector-based placement source for one-shot style bursts.
type EmitterVectorConfig struct {
	Type      string                       `yaml:"type,omitempty"`      // rect, polyline
	Placement string                       `yaml:"placement,omitempty"` // fill or surface
	Rect      *EmitterVectorRectConfig     `yaml:"rect,omitempty"`
	Polyline  *EmitterVectorPolylineConfig `yaml:"polyline,omitempty"`
}

// EmitterVectorRectConfig defines a rectangle placement source centered on the emitter.
type EmitterVectorRectConfig struct {
	Width    float32 `yaml:"width"`
	Height   float32 `yaml:"height"`
	Rotation float32 `yaml:"rotation,omitempty"`
}

// EmitterVectorPolylineConfig defines an open or closed line strip centered on the emitter.
type EmitterVectorPolylineConfig struct {
	Closed        bool                 `yaml:"closed,omitempty"`
	Interpolation string               `yaml:"interpolation,omitempty"` // linear (default) or quadratic
	CurveSteps    int                  `yaml:"curve_steps,omitempty"`   // Samples per quadratic segment
	Points        []EmitterVectorPoint `yaml:"points"`
}

// EmitterVectorPoint defines a 2D point for vector placement.
type EmitterVectorPoint struct {
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
	// Type controls the motion model:
	//   "cartesian" (default) - linear lerp between start and end
	//   "polar"               - radial burst from emitter
	//   "attractor"           - quadratic bezier from emitter through a random
	//                           control point toward AttractorX/Y on SystemData
	Type string `yaml:"type,omitempty"`

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
	Distance *RangeFloat `yaml:"distance,omitempty"` // Distance from emitter (spawn offset in velocity mode)
	Speed        *RangeFloat `yaml:"speed,omitempty"`         // units/sec; presence enables velocity mode (duration = lifetime only)
	AngularSpeed *RangeFloat `yaml:"angular_speed,omitempty"` // rad/sec; positive = CCW, negative = CW

	// Attractor mode - random bezier control point offset from the emitter
	ControlX *RangeFloat `yaml:"control_x,omitempty"` // X offset range for bezier control point
	ControlY *RangeFloat `yaml:"control_y,omitempty"` // Y offset range for bezier control point

	// Flow mode - continuous field offset layered on top of the base path
	Flow *FlowConfig `yaml:"flow,omitempty"`

	Easing string `yaml:"easing"`
}

// FlowConfig defines a continuous vector field layered on top of the base path.
type FlowConfig struct {
	Type            string      `yaml:"type,omitempty"` // curl
	Strength        *RangeFloat `yaml:"strength,omitempty"`
	Scale           float32     `yaml:"scale,omitempty"`             // Larger values produce wider motion
	Octaves         int         `yaml:"octaves,omitempty"`           // Layer count, clamped to a small range
	Persistence     float32     `yaml:"persistence,omitempty"`       // Amplitude falloff per octave
	TimeScale       float32     `yaml:"time_scale,omitempty"`        // Field evolution speed
	Drag            float32     `yaml:"drag,omitempty"`              // Velocity damping per update
	Space           string      `yaml:"space,omitempty"`             // local (default) or world
	BoundRadius     float32     `yaml:"bound_radius,omitempty"`      // Optional reset radius from emitter
	RespawnOnEscape bool        `yaml:"respawn_on_escape,omitempty"` // Reset flow offset when leaving bounds
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
	From      float32     `yaml:"from"`
	FromRange *RangeFloat `yaml:"from_range,omitempty"`
	To        float32     `yaml:"to"`
	ToRange   *RangeFloat `yaml:"to_range,omitempty"`
	Duration  float32     `yaml:"duration"`
	Easing    string      `yaml:"easing"`
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

// TrailConfig defines an optional ribbon trail emitted from the emitter position.
type TrailConfig struct {
	Enabled          bool              `yaml:"enabled,omitempty"`
	Mode             string            `yaml:"mode,omitempty"` // emitter
	Space            string            `yaml:"space,omitempty"`
	MaxPoints        int               `yaml:"max_points,omitempty"`
	MinPointDistance float32           `yaml:"min_point_distance,omitempty"`
	MaxPointAge      float32           `yaml:"max_point_age,omitempty"`
	Width            TrailScalarConfig `yaml:"width"`
	Alpha            TrailScalarConfig `yaml:"alpha"`
	Color            *ColorConfig      `yaml:"color,omitempty"`
}

// TrailScalarConfig defines a simple scalar gradient across trail point age.
type TrailScalarConfig struct {
	Start  float32 `yaml:"start"`
	End    float32 `yaml:"end"`
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
