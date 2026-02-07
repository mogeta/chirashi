package aburi

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
)

var (
	// Global configuration loader instance
	configLoader = NewConfigLoader()
)

// NewParticlesFromConfig creates a GPU particle system from a configuration struct
func NewParticlesFromConfig(w donburi.World, shader *ebiten.Shader, image *ebiten.Image, config *ParticleConfig, x, y float32) error {
	return createParticlesFromConfig(w, shader, image, config, x, y)
}

// NewParticlesFromFile creates a GPU particle system from a configuration file path
func NewParticlesFromFile(w donburi.World, shader *ebiten.Shader, image *ebiten.Image, configPath string, x, y float32) error {
	config, err := configLoader.LoadConfig(configPath)
	if err != nil {
		return err
	}

	return createParticlesFromConfig(w, shader, image, config, x, y)
}

// createParticlesFromConfig creates particles from a loaded configuration
func createParticlesFromConfig(w donburi.World, shader *ebiten.Shader, image *ebiten.Image, config *ParticleConfig, x, y float32) error {
	particles := w.Entry(w.Create(Component))

	// Calculate emitter position (base position + config offset)
	emitterX := x + config.Emitter.X
	emitterY := y + config.Emitter.Y

	// Build animation parameters from config
	animParams := buildAnimationParams(config)

	// Initialize free indices pool (all particles are free initially)
	freeIndices := make([]int, config.Spawn.MaxParticles)
	for i := range freeIndices {
		freeIndices[i] = config.Spawn.MaxParticles - 1 - i // Reverse order for stack
	}

	// Pre-allocate vertex/index buffers
	// Each particle = 4 vertices, 6 indices
	maxVertices := config.Spawn.MaxParticles * 4
	maxIndices := config.Spawn.MaxParticles * 6

	// Cache image dimensions
	var imgWidth, imgHeight float32
	if image != nil {
		bounds := image.Bounds()
		imgWidth = float32(bounds.Dx())
		imgHeight = float32(bounds.Dy())
	}

	// Build particle system data
	systemData := SystemData{
		ParticlePool:      make([]Instance, config.Spawn.MaxParticles),
		ActiveIndices:     make([]int, 0, config.Spawn.MaxParticles),
		FreeIndices:       freeIndices,
		Vertices:          make([]ebiten.Vertex, 0, maxVertices),
		Indices:           make([]uint16, 0, maxIndices),
		Shader:            shader,
		CurrentTime:       0,
		EmitterX:          emitterX,
		EmitterY:          emitterY,
		SpawnInterval:     config.Spawn.Interval,
		ParticlesPerSpawn: config.Spawn.ParticlesPerSpawn,
		MaxParticles:      config.Spawn.MaxParticles,
		SourceImage:       image,
		ImageWidth:        imgWidth,
		ImageHeight:       imgHeight,
		ActiveCount:       0,
		IsLoop:            config.Spawn.IsLoop,
		LifeTime:          config.Spawn.LifeTime,
		AnimParams:        animParams,
	}

	donburi.SetValue(particles, Component, systemData)
	return nil
}

// buildAnimationParams converts config to runtime animation parameters
func buildAnimationParams(config *ParticleConfig) AnimationParams {
	params := AnimationParams{
		DurationBase: config.Animation.Duration.Value,
	}

	// Duration range
	if config.Animation.Duration.Range != nil {
		params.DurationRange = (config.Animation.Duration.Range.Max - config.Animation.Duration.Range.Min) / 2
		params.DurationBase = (config.Animation.Duration.Range.Max + config.Animation.Duration.Range.Min) / 2
	}

	// Position mode
	params.UsePolar = config.Animation.Position.Type == "polar"

	if params.UsePolar {
		// Polar mode
		if config.Animation.Position.Angle != nil {
			params.AngleMin = config.Animation.Position.Angle.Min
			params.AngleMax = config.Animation.Position.Angle.Max
		}
		if config.Animation.Position.Distance != nil {
			params.DistanceMin = config.Animation.Position.Distance.Min
			params.DistanceMax = config.Animation.Position.Distance.Max
		}
		fmt.Printf("buildAnimationParams: Polar mode - Angle(%.2f-%.2f) Dist(%.0f-%.0f)\n",
			params.AngleMin, params.AngleMax, params.DistanceMin, params.DistanceMax)
	} else {
		// Cartesian mode
		if config.Animation.Position.StartX != nil {
			params.StartXMin = config.Animation.Position.StartX.Min
			params.StartXMax = config.Animation.Position.StartX.Max
		}
		if config.Animation.Position.EndX != nil {
			params.EndXMin = config.Animation.Position.EndX.Min
			params.EndXMax = config.Animation.Position.EndX.Max
		}
		if config.Animation.Position.StartY != nil {
			params.StartYMin = config.Animation.Position.StartY.Min
			params.StartYMax = config.Animation.Position.StartY.Max
		}
		if config.Animation.Position.EndY != nil {
			params.EndYMin = config.Animation.Position.EndY.Min
			params.EndYMax = config.Animation.Position.EndY.Max
		}
	}

	// Position easing
	params.PositionEasing = ParseEasing(config.Animation.Position.Easing)

	// Alpha
	params.StartAlpha = config.Animation.Alpha.Start
	params.EndAlpha = config.Animation.Alpha.End
	params.AlphaEasing = ParseEasing(config.Animation.Alpha.Easing)

	// Scale
	params.StartScale = config.Animation.Scale.Start
	params.EndScale = config.Animation.Scale.End
	if params.StartScale == 0 && params.EndScale == 0 {
		params.StartScale = 1.0
		params.EndScale = 1.0
	}
	params.ScaleEasing = ParseEasing(config.Animation.Scale.Easing)

	// Rotation
	params.StartRotation = config.Animation.Rotation.Start
	params.EndRotation = config.Animation.Rotation.End
	params.RotationEasing = ParseEasing(config.Animation.Rotation.Easing)

	// Color
	if config.Animation.Color != nil {
		params.UseColor = true
		params.StartR = config.Animation.Color.StartR
		params.StartG = config.Animation.Color.StartG
		params.StartB = config.Animation.Color.StartB
		params.EndR = config.Animation.Color.EndR
		params.EndG = config.Animation.Color.EndG
		params.EndB = config.Animation.Color.EndB
		params.ColorEasing = ParseEasing(config.Animation.Color.Easing)
	} else {
		// Default: white (no color tinting)
		params.StartR, params.StartG, params.StartB = 1, 1, 1
		params.EndR, params.EndG, params.EndB = 1, 1, 1
	}

	return params
}

// ReloadConfig reloads all cached configurations (useful for development)
func ReloadConfig() {
	configLoader.ClearCache()
}

// GetConfigLoader returns the global config loader for advanced usage
func GetConfigLoader() *ConfigLoader {
	return configLoader
}
