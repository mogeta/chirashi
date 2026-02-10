package chirashi

import (
	core "github.com/mogeta/chirashi/component/chirashi"
)

// Core runtime types.
type (
	System          = core.System
	ParticleManager = core.ParticleManager
	ConfigLoader    = core.ConfigLoader
)

// Configuration types.
type (
	ParticleConfig  = core.ParticleConfig
	ImageConfig     = core.ImageConfig
	EmitterConfig   = core.EmitterConfig
	AnimationConfig = core.AnimationConfig
	DurationConfig  = core.DurationConfig
	RangeFloat      = core.RangeFloat
	PositionConfig  = core.PositionConfig
	PropertyConfig  = core.PropertyConfig
	StepConfig      = core.StepConfig
	ColorConfig     = core.ColorConfig
	SpawnConfig     = core.SpawnConfig
)

// Component/data types for ECS integration.
type (
	Instance        = core.Instance
	SystemData      = core.SystemData
	AnimationParams = core.AnimationParams
	Metrics         = core.Metrics
	ParticleStorage = core.ParticleStorage
)

// Easing and sequence helpers.
type (
	EasingType       = core.EasingType
	SequenceConfig   = core.SequenceConfig
	SequenceStep     = core.SequenceStep
	SequenceSnapshot = core.SequenceSnapshot
)

var (
	// ECS component registration.
	Component = core.Component

	// Runtime constructors.
	NewSystem          = core.NewSystem
	NewParticleManager = core.NewParticleManager
	NewConfigLoader    = core.NewConfigLoader

	// Config loader access.
	GetConfigLoader = core.GetConfigLoader
	ReloadConfig    = core.ReloadConfig

	// Particle creation helpers.
	NewParticlesFromConfig = core.NewParticlesFromConfig
	NewParticlesFromFile   = core.NewParticlesFromFile

	// Easing and sequence helpers.
	ParseEasing       = core.ParseEasing
	ApplyEasing       = core.ApplyEasing
	NewSequenceConfig = core.NewSequenceConfig
	GenerateSnapshot  = core.GenerateSnapshot
	EvaluateSequence  = core.EvaluateSequence
)
