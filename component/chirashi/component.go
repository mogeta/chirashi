package chirashi

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tanema/gween"
	"github.com/yohamta/donburi"
)

// SequenceFunc represents a function that creates a new tween sequence
type SequenceFunc func() *gween.Sequence

// SystemData represents the particle system component data
type SystemData struct {
	// Pool of particles for efficient memory management
	ParticlePool []Instance
	// Index management for O(1) operations
	ActiveIndices []int // Indices of active particles (compact array)
	FreeIndices   []int // Stack of free particle indices
	// Emitter configuration
	EmitterPosition      Position
	SequenceFactoryX     SequenceFunc
	SequenceFactoryY     SequenceFunc
	SequenceFactoryAngle SequenceFunc // Polar Angle
	SequenceFactoryDist  SequenceFunc // Polar Distance
	SequenceFactoryAlpha SequenceFunc // Alpha
	SequenceFactoryR     SequenceFunc // Rotation
	SequenceFactoryS     SequenceFunc // Scale
	// Spawn configuration
	SpawnInterval     int
	ParticlesPerSpawn int
	MaxParticles      int
	// Rendering
	SourceImage *ebiten.Image //
	// Internal state
	ActiveCount  int
	IsLoop       bool   // IsLoop indicates whether the particle system should loop its behavior.
	LifeTime     int    // LifeTime specifies the total duration (in frames) the particle system remains active.
	MovementType string // "cartesian" or "polar"
}

// Position represents a 2D position
type Position struct {
	X, Y float64
}

// Instance represents a single particle
type Instance struct {
	// Transform
	Position Position
	Alpha    float64
	Rotation float64 // Current rotation in radians
	Scale    float64 // Current scale factor
	// Animation sequences
	SequenceX      *gween.Sequence
	SequenceY      *gween.Sequence
	SequenceAngle  *gween.Sequence // Polar Angle
	SequenceDist   *gween.Sequence // Polar Distance
	SequenceAlpha  *gween.Sequence
	SequenceRotate *gween.Sequence // Rotation sequence
	SequenceScale  *gween.Sequence // Scale sequence
	// State
	Active bool
}

// Component is the Donburi component type for particle systems
var Component = donburi.NewComponentType[SystemData]()
