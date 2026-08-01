// Package particle provides backward-compatible imports for chirashi's particle API.
//
// Deprecated: use github.com/mogeta/chirashi directly.
package particle

import (
	root "github.com/mogeta/chirashi"
)

// Core runtime types.
type (
	System          = root.System
	ParticleManager = root.ParticleManager
	ConfigLoader    = root.ConfigLoader
)

// Configuration types.
type (
	ParticleConfig   = root.ParticleConfig
	ImageConfig      = root.ImageConfig
	RenderConfig     = root.RenderConfig
	BloomConfig      = root.BloomConfig
	AfterimageConfig = root.AfterimageConfig
	EmitterConfig    = root.EmitterConfig
	AnimationConfig  = root.AnimationConfig
	DurationConfig   = root.DurationConfig
	RangeFloat       = root.RangeFloat
	PositionConfig   = root.PositionConfig
	PropertyConfig   = root.PropertyConfig
	StepConfig       = root.StepConfig
	ColorConfig      = root.ColorConfig
	SpawnConfig      = root.SpawnConfig
)

// Component/data types for ECS integration.
type (
	Instance          = root.Instance
	SystemData        = root.SystemData
	AnimationParams   = root.AnimationParams
	Metrics           = root.Metrics
	ParticleStorage   = root.ParticleStorage
	BloomEffect       = root.BloomEffect
	PersistenceEffect = root.PersistenceEffect
)

// Easing and sequence helpers.
type (
	EasingType       = root.EasingType
	SequenceConfig   = root.SequenceConfig
	SequenceStep     = root.SequenceStep
	SequenceSnapshot = root.SequenceSnapshot
)

var (
	// ECS component registration.
	Component = root.Component

	// Runtime constructors.
	NewSystem            = root.NewSystem
	NewParticleManager   = root.NewParticleManager
	NewConfigLoader      = root.NewConfigLoader
	NewBloomEffect       = root.NewBloomEffect
	NewPersistenceEffect = root.NewPersistenceEffect

	// Particle creation helpers.
	NewParticlesFromConfig = root.NewParticlesFromConfig
	NewParticlesFromFile   = root.NewParticlesFromFile

	// Runtime particle controls.
	SetEmissionScale = root.SetEmissionScale

	// Easing and sequence helpers.
	ParseEasing       = root.ParseEasing
	ApplyEasing       = root.ApplyEasing
	NewSequenceConfig = root.NewSequenceConfig
	GenerateSnapshot  = root.GenerateSnapshot
	EvaluateSequence  = root.EvaluateSequence
)
