package chirashi

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
)

// Instance represents a single GPU-based particle
type Instance struct {
	// Timing
	SpawnTime float32 // Time when particle was spawned
	Duration  float32 // Total animation duration in seconds

	// Position animation
	StartX, EndX float32
	StartY, EndY float32

	// Appearance animation
	StartAlpha, EndAlpha       float32
	StartScale, EndScale       float32
	StartRotation, EndRotation float32

	// Color animation (RGB 0-1)
	StartR, StartG, StartB float32
	EndR, EndG, EndB       float32

	// Easing types for each property
	PositionEasing EasingType
	AlphaEasing    EasingType
	ScaleEasing    EasingType
	RotationEasing EasingType
	ColorEasing    EasingType

	// Multi-step sequence snapshots (used when MultiStep=true)
	PosXSnap  SequenceSnapshot
	PosYSnap  SequenceSnapshot
	ScaleSnap SequenceSnapshot
	RotSnap   SequenceSnapshot
	AlphaSnap SequenceSnapshot

	// State
	Active    bool
	MultiStep bool // true=use sequence evaluation, false=simple lerp
}

// SystemData represents the GPU-based particle system component data
type SystemData struct {
	// Pool of particles for efficient memory management
	ParticlePool []Instance

	// Index management for O(1) operations
	ActiveIndices []int // Indices of active particles (compact array)
	FreeIndices   []int // Stack of free particle indices

	// Pre-allocated vertex/index buffers for batch rendering
	Vertices []ebiten.Vertex
	Indices  []uint16

	// GPU resources
	Shader *ebiten.Shader

	// Timing
	CurrentTime float32

	// Emitter configuration
	EmitterX, EmitterY float32

	// Spawn configuration
	SpawnInterval     int
	ParticlesPerSpawn int
	MaxParticles      int

	// Rendering
	SourceImage *ebiten.Image
	ImageWidth  float32 // Cached image width
	ImageHeight float32 // Cached image height

	// Internal state
	ActiveCount int
	IsLoop      bool
	LifeTime    int // Remaining lifetime in frames (if not looping)

	// Animation parameters (from config, used for spawning)
	AnimParams AnimationParams

	// Multi-step sequence configurations (nil = simple mode)
	PosXSeq  *SequenceConfig
	PosYSeq  *SequenceConfig
	ScaleSeq *SequenceConfig
	RotSeq   *SequenceConfig
	AlphaSeq *SequenceConfig

	// Performance metrics
	Metrics Metrics
}

// AnimationParams holds the configuration for particle animations, grouped by concern.
type AnimationParams struct {
	Duration   DurationParams
	Position   PositionParams
	Appearance AppearanceParams
	Color      ColorParams
}

// DurationParams holds lifetime randomization for particles.
type DurationParams struct {
	Base  float32 // Base duration in seconds
	Range float32 // +/- randomization range
}

// PositionParams holds spawn position configuration (cartesian or polar).
type PositionParams struct {
	UsePolar bool // true = polar, false = cartesian

	// Cartesian
	StartXMin, StartXMax float32
	EndXMin, EndXMax     float32
	StartYMin, StartYMax float32
	EndYMin, EndYMax     float32

	// Polar
	AngleMin, AngleMax float32 // Radians
	DistMin, DistMax   float32

	Easing EasingType
}

// AppearanceParams holds alpha, scale, and rotation animation configuration.
type AppearanceParams struct {
	StartAlpha, EndAlpha       float32
	AlphaEasing                EasingType
	StartScale, EndScale       float32
	ScaleEasing                EasingType
	StartRotation, EndRotation float32
	RotationEasing             EasingType
}

// ColorParams holds color animation configuration.
type ColorParams struct {
	Enabled                bool
	StartR, StartG, StartB float32
	EndR, EndG, EndB       float32
	Easing                 EasingType
}

// Metrics tracks performance data for a particle system
type Metrics struct {
	UpdateTimeUs    int64 // Update time in microseconds
	DrawTimeUs      int64 // Draw time in microseconds
	SpawnCount      int   // Total particles spawned (cumulative)
	DeactivateCount int   // Total particles deactivated (cumulative)
	FrameCount      int   // Frame counter
}

// Component is the Donburi component type for GPU particle systems
var Component = donburi.NewComponentType[SystemData]()
