package chirashi

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tanema/gween"
	"github.com/yohamta/donburi"
)

var (
	// Global configuration loader instance
	configLoader = NewConfigLoader()
	tweenFactory = NewTweenFactory()
)

// NewParticlesFromConfig creates a particle system from a configuration struct
func NewParticlesFromConfig(w donburi.World, image *ebiten.Image, config *ParticleConfig, x, y float64) error {
	return createParticlesFromConfig(w, image, config, x, y)
}

// NewParticlesFromConfigWithAssets creates a particle system from a named configuration, using image settings from config
//func NewParticlesFromConfigWithAssets(w donburi.World, configName string, x, y float64) error {
//	config, err := configLoader.LoadFromAssets(configName)
//	if err != nil {
//		return fmt.Errorf("failed to load config '%s': %w", configName, err)
//	}
//
//	// Get image from assets based on config
//	var image *ebiten.Image
//	if config.Image.ImageFrom != "" && config.Image.ImageID != 0 {
//		imageSource := assets.StatusUnknown
//		// Parse image source string
//		switch config.Image.ImageFrom {
//		case "ef1", "EF1":
//			imageSource = assets.EF1
//		case "cs1", "CS1":
//			imageSource = assets.CS1
//		case "direct":
//			imageSource = assets.Direct
//		}
//		image = assets.Assets.GetImage(imageSource, config.Image.ImageID)
//	}
//
//	// Fallback to default image if no image config or failed to get image
//	if image == nil {
//		image = assets.Assets.GetSpriteEf1(26) // Default fallback
//	}
//
//	return createParticlesFromConfig(w, image, config, x, y)
//}

// NewParticlesFromFile creates a particle system from a configuration file path
func NewParticlesFromFile(w donburi.World, image *ebiten.Image, configPath string, x, y float64) error {
	config, err := configLoader.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config from file '%s': %w", configPath, err)
	}

	return createParticlesFromConfig(w, image, config, x, y)
}

// createParticlesFromConfig creates particles from a loaded configuration
func createParticlesFromConfig(w donburi.World, image *ebiten.Image, config *ParticleConfig, x, y float64) error {
	particles := w.Entry(w.Create(Component))

	// Calculate emitter position (base position + config offset)
	emitterX := x + config.Emitter.Position.X
	emitterY := y + config.Emitter.Position.Y

	// Create movement sequence factories
	var xFactory, yFactory, angleFactory, distFactory SequenceFunc
	movementType := config.Movement.Type
	if movementType == "" {
		movementType = "cartesian" // Default
	}

	if movementType == "polar" {
		angleFactory, distFactory = tweenFactory.CreatePolarFactories(config.Movement)
	} else {
		xFactory, yFactory = tweenFactory.CreateMovementFactories(config.Movement, emitterX, emitterY)
	}

	// Create rotation sequence factory
	var rotationFactory SequenceFunc
	if len(config.Appearance.Rotation.Steps) > 0 {
		rotationFactory = func() *gween.Sequence {
			return tweenFactory.CreateSequence(config.Appearance.Rotation, 0.0)
		}
	}

	// Create scale sequence factory
	var scaleFactory SequenceFunc
	if len(config.Appearance.Scale.Steps) > 0 {
		scaleFactory = func() *gween.Sequence {

			return tweenFactory.CreateSequence(config.Appearance.Scale, 1.0)
		}
	}

	// Create alpha sequence factory
	var alphaFactory SequenceFunc
	if len(config.Appearance.Alpha.Steps) > 0 {
		alphaFactory = func() *gween.Sequence {
			return tweenFactory.CreateSequence(config.Appearance.Alpha, 0)
		}
	}

	// Initialize free indices pool (all particles are free initially)
	freeIndices := make([]int, config.Spawn.MaxParticles)
	for i := range freeIndices {
		freeIndices[i] = config.Spawn.MaxParticles - 1 - i // Reverse order for stack
	}

	// Cache image dimensions
	var imgWidth, imgHeight float64
	if image != nil {
		bounds := image.Bounds()
		imgWidth = float64(bounds.Dx())
		imgHeight = float64(bounds.Dy())
	}

	// Build particle system data
	systemData := SystemData{
		ParticlePool:         make([]Instance, config.Spawn.MaxParticles),
		ActiveIndices:        make([]int, 0, config.Spawn.MaxParticles),
		FreeIndices:          freeIndices,
		EmitterPosition:      Position{X: emitterX, Y: emitterY},
		SequenceFactoryX:     xFactory,
		SequenceFactoryY:     yFactory,
		SequenceFactoryAngle: angleFactory,
		SequenceFactoryDist:  distFactory,
		SequenceFactoryR:     rotationFactory,
		SequenceFactoryS:     scaleFactory,
		SequenceFactoryAlpha: alphaFactory,
		SpawnInterval:        config.Spawn.Interval,
		ParticlesPerSpawn:    config.Spawn.ParticlesPerSpawn,
		MaxParticles:         config.Spawn.MaxParticles,
		SourceImage:          image,
		ImageWidth:           imgWidth,
		ImageHeight:          imgHeight,
		ActiveCount:          0,
		IsLoop:               config.Spawn.IsLoop,
		LifeTime:             config.Spawn.LifeTime,
		MovementType:         movementType,
	}

	donburi.SetValue(particles, Component, systemData)
	return nil
}

// ReloadConfig reloads all cached configurations (useful for development)
func ReloadConfig() {
	configLoader.ClearCache()
}

// GetConfigLoader returns the global config loader for advanced usage
func GetConfigLoader() *ConfigLoader {
	return configLoader
}

// GetTweenFactory returns the global tween factory for advanced usage
func GetTweenFactory() *TweenFactory {
	return tweenFactory
}

// NewParticlesFromConfigWithCustom creates particles with custom overrides
func NewParticlesFromConfigWithCustom(w donburi.World, image *ebiten.Image, configName string, x, y float64, overrides ConfigOverrides) error {
	config, err := configLoader.LoadFromAssets(configName)
	if err != nil {
		return fmt.Errorf("failed to load config '%s': %w", configName, err)
	}

	// Apply overrides
	if overrides.MaxParticles != nil {
		config.Spawn.MaxParticles = *overrides.MaxParticles
	}
	if overrides.ParticlesPerSpawn != nil {
		config.Spawn.ParticlesPerSpawn = *overrides.ParticlesPerSpawn
	}
	if overrides.SpawnInterval != nil {
		config.Spawn.Interval = *overrides.SpawnInterval
	}
	if overrides.IsLoop != nil {
		config.Spawn.IsLoop = *overrides.IsLoop
	}
	if overrides.LifeTime != nil {
		config.Spawn.LifeTime = *overrides.LifeTime
	}

	return createParticlesFromConfig(w, image, config, x, y)
}

// ConfigOverrides allows runtime configuration overrides
type ConfigOverrides struct {
	MaxParticles      *int
	ParticlesPerSpawn *int
	SpawnInterval     *int
	IsLoop            *bool
	LifeTime          *int
}
