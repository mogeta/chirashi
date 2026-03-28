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

	// Attractor mode: quadratic bezier control point (randomized at spawn)
	ControlX, ControlY float32
	HasAttractor       bool

	// Polar velocity mode (speed-based radial movement)
	HasPolarVelocity bool
	DirX, DirY       float32 // unit direction vector (cos/sin of spawn angle)
	SpawnDist        float32 // initial distance from emitter at spawn
	Speed            float32 // radial speed in units/sec
	HasFlow            bool
	FlowGain           float32
	FlowOffsetX        float32
	FlowOffsetY        float32
	FlowVelX           float32
	FlowVelY           float32
	FlowSeedX          float32
	FlowSeedY          float32
	CurrentX           float32
	CurrentY           float32
	CurrentPosValid    bool
	CurrentPosTime     float32

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

	// Per-property sequence snapshots and flags
	// Each flag is true when the corresponding sequence was configured at spawn time.
	HasPosXSeq  bool
	PosXSnap    SequenceSnapshot
	HasPosYSeq  bool
	PosYSnap    SequenceSnapshot
	HasScaleSeq bool
	ScaleSnap   SequenceSnapshot
	HasRotSeq   bool
	RotSnap     SequenceSnapshot
	HasAlphaSeq bool
	AlphaSnap   SequenceSnapshot

	// Particle trail history (used only when trail.mode == "particle")
	TrailPoints []TrailPoint

	// State
	Active bool
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
	EmitterShape       EmitterShapeParams
	EmitterVector      EmitterVectorParams
	EmitterLocalSpace  bool

	// Spawn configuration
	SpawnInterval     int
	ParticlesPerSpawn int
	MaxParticles      int

	// Rendering
	SourceImage *ebiten.Image
	ImageWidth  float32 // Cached image width
	ImageHeight float32 // Cached image height
	Trail       TrailData

	// Internal state
	ActiveCount int
	IsLoop      bool
	LifeTime    int // Remaining lifetime in frames (if not looping)

	// Animation parameters (from config, used for spawning)
	AnimParams AnimationParams

	// Attractor target — set at runtime to define where particles converge.
	// Only used when position type is "attractor".
	AttractorX, AttractorY float32

	// Multi-step sequence configurations (nil = simple mode)
	PosXSeq  *SequenceConfig
	PosYSeq  *SequenceConfig
	ScaleSeq *SequenceConfig
	RotSeq   *SequenceConfig
	AlphaSeq *SequenceConfig

	// Performance metrics
	Metrics Metrics
}

// EmitterShapeParams holds runtime emitter shape configuration.
type EmitterShapeParams struct {
	Type       EmitterShapeType
	RadiusMin  float32
	RadiusMax  float32
	StartAngle float32
	EndAngle   float32
	Width      float32
	Height     float32
	Length     float32
	Rotation   float32
	FromEdge   bool
}

// EmitterShapeType identifies how particles are spawned around the emitter.
type EmitterShapeType int

const (
	EmitterShapePoint EmitterShapeType = iota
	EmitterShapeCircle
	EmitterShapeBox
	EmitterShapeLine
)

// EmitterVectorParams stores normalized vector-based spawn placement.
type EmitterVectorParams struct {
	Enabled   bool
	Type      EmitterVectorType
	Placement EmitterVectorPlacement
	Rect      EmitterVectorRectParams
	Polyline  EmitterVectorPolylineParams
}

type EmitterVectorType int

const (
	EmitterVectorNone EmitterVectorType = iota
	EmitterVectorRect
	EmitterVectorPolyline
)

type EmitterVectorPlacement int

const (
	EmitterVectorFill EmitterVectorPlacement = iota
	EmitterVectorSurface
)

type EmitterVectorRectParams struct {
	Width    float32
	Height   float32
	Rotation float32
}

type EmitterVectorPolylineParams struct {
	Closed         bool
	Interpolation  string
	CurveSteps     int
	Points         []EmitterVectorPointParams
	SegmentLengths []float32
	TotalLength    float32
}

type EmitterVectorPointParams struct {
	X float32
	Y float32
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

// PositionParams holds spawn position configuration.
type PositionParams struct {
	UsePolar     bool // true = polar
	UseAttractor bool // true = quadratic bezier toward AttractorX/Y

	// Cartesian
	StartXMin, StartXMax float32
	EndXMin, EndXMax     float32
	StartYMin, StartYMax float32
	EndYMin, EndYMax     float32

	// Polar
	AngleMin, AngleMax float32 // Radians
	DistMin, DistMax   float32
	SpeedMin, SpeedMax float32 // units/sec; non-zero enables velocity mode

	// Attractor: bezier control point offset from emitter
	ControlXMin, ControlXMax float32
	ControlYMin, ControlYMax float32

	HasFlow             bool
	FlowStrengthMin     float32
	FlowStrengthMax     float32
	FlowScale           float32
	FlowOctaves         int
	FlowPersistence     float32
	FlowTimeScale       float32
	FlowDrag            float32
	FlowLocalSpace      bool
	FlowBoundRadius     float32
	FlowRespawnOnEscape bool

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

// TrailPoint stores one sampled emitter position for ribbon trail rendering.
type TrailPoint struct {
	X, Y       float32
	CapturedAt float32
}

// TrailGhost stores a detached particle trail after the source particle expires.
type TrailGhost struct {
	Points []TrailPoint
}

// TrailParams stores normalized trail configuration values.
type TrailParams struct {
	Enabled          bool
	Mode             string
	LocalSpace       bool
	MaxPoints        int
	MinPointDistance float32
	MaxPointAge      float32
	WidthStart       float32
	WidthEnd         float32
	WidthEasing      EasingType
	AlphaStart       float32
	AlphaEnd         float32
	AlphaEasing      EasingType
	ColorStartR      float32
	ColorStartG      float32
	ColorStartB      float32
	ColorEndR        float32
	ColorEndG        float32
	ColorEndB        float32
	ColorEasing      EasingType
}

// TrailRuntime stores mutable trail history and draw buffers.
type TrailRuntime struct {
	Points   []TrailPoint
	Ghosts   []TrailGhost
	Vertices []ebiten.Vertex
	Indices  []uint16
}

// TrailData stores ribbon trail configuration and runtime state.
type TrailData struct {
	Params  TrailParams
	Runtime TrailRuntime
}

// Component is the Donburi component type for GPU particle systems
var Component = donburi.NewComponentType[SystemData]()
