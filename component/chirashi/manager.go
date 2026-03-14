package chirashi

import (
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
)

// ParticleManager manages particle configurations and provides easy spawning API
type ParticleManager struct {
	shader  *ebiten.Shader
	image   *ebiten.Image
	configs map[string]*ParticleConfig
	loader  *ConfigLoader
	mutex   sync.RWMutex
}

// NewParticleManager creates a new particle manager
func NewParticleManager(shader *ebiten.Shader, image *ebiten.Image) *ParticleManager {
	return &ParticleManager{
		shader:  shader,
		image:   image,
		configs: make(map[string]*ParticleConfig),
		loader:  NewConfigLoader(),
	}
}

// Preload loads and caches a particle configuration
func (m *ParticleManager) Preload(name string, path string) error {
	config, err := m.loader.LoadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to preload %s: %w", name, err)
	}

	m.mutex.Lock()
	m.configs[name] = config
	m.mutex.Unlock()

	return nil
}

// PreloadFromBytes loads and caches a particle configuration from embedded bytes
func (m *ParticleManager) PreloadFromBytes(name string, data []byte) error {
	config, err := m.loader.LoadConfigFromBytes(data, name)
	if err != nil {
		return fmt.Errorf("failed to preload %s: %w", name, err)
	}

	m.mutex.Lock()
	m.configs[name] = config
	m.mutex.Unlock()

	return nil
}

// SpawnOneShot spawns a one-shot particle effect at the given position
// The particle system will automatically be removed after the specified lifetime (in frames)
func (m *ParticleManager) SpawnOneShot(world donburi.World, name string, x, y float32, lifetimeFrames int) error {
	m.mutex.RLock()
	baseConfig, exists := m.configs[name]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("particle config '%s' not found, call Preload first", name)
	}

	// Copy config to avoid modifying the cached version
	config := copyConfig(baseConfig)
	config.Spawn.IsLoop = false
	config.Spawn.LifeTime = lifetimeFrames

	return NewParticlesFromConfig(world, m.shader, m.image, config, x, y)
}

// SpawnLoop spawns a looping particle effect at the given position
// Returns the entity for manual removal later
func (m *ParticleManager) SpawnLoop(world donburi.World, name string, x, y float32) (donburi.Entity, error) {
	m.mutex.RLock()
	baseConfig, exists := m.configs[name]
	m.mutex.RUnlock()

	if !exists {
		return 0, fmt.Errorf("particle config '%s' not found, call Preload first", name)
	}

	// Copy config
	config := copyConfig(baseConfig)
	config.Spawn.IsLoop = true
	normalizeParticleConfig(config)

	// Create entity
	entity := world.Create(Component)
	entry := world.Entry(entity)

	// Build system data
	emitterX := x + config.Emitter.X
	emitterY := y + config.Emitter.Y
	animParams := buildAnimationParams(config)

	freeIndices := make([]int, config.Spawn.MaxParticles)
	for i := range freeIndices {
		freeIndices[i] = config.Spawn.MaxParticles - 1 - i
	}

	maxVertices := config.Spawn.MaxParticles * 4
	maxIndices := config.Spawn.MaxParticles * 6

	var imgWidth, imgHeight float32
	if m.image != nil {
		bounds := m.image.Bounds()
		imgWidth = float32(bounds.Dx())
		imgHeight = float32(bounds.Dy())
	}

	systemData := SystemData{
		ParticlePool:      make([]Instance, config.Spawn.MaxParticles),
		ActiveIndices:     make([]int, 0, config.Spawn.MaxParticles),
		FreeIndices:       freeIndices,
		Vertices:          make([]ebiten.Vertex, 0, maxVertices),
		Indices:           make([]uint16, 0, maxIndices),
		Shader:            m.shader,
		CurrentTime:       0,
		EmitterX:          emitterX,
		EmitterY:          emitterY,
		EmitterShape:      buildEmitterShapeParams(config.Emitter.Shape),
		SpawnInterval:     config.Spawn.Interval,
		ParticlesPerSpawn: config.Spawn.ParticlesPerSpawn,
		MaxParticles:      config.Spawn.MaxParticles,
		SourceImage:       m.image,
		ImageWidth:        imgWidth,
		ImageHeight:       imgHeight,
		ActiveCount:       0,
		IsLoop:            true,
		LifeTime:          0,
		AnimParams:        animParams,
	}

	// Apply sequence configurations if present
	buildSequenceConfigs(config, &systemData)

	donburi.SetValue(entry, Component, systemData)
	return entity, nil
}

// SetShader updates the shader used for rendering
func (m *ParticleManager) SetShader(shader *ebiten.Shader) {
	m.shader = shader
}

// SetImage updates the default image used for particles
func (m *ParticleManager) SetImage(image *ebiten.Image) {
	m.image = image
}

// SetAttractor updates the attractor target for a particle entity.
// Call each frame when the target moves (e.g. a score counter that slides around).
// Has no effect on particles that do not use position type "attractor".
func SetAttractor(world donburi.World, entity donburi.Entity, x, y float32) {
	entry := world.Entry(entity)
	if !world.Valid(entity) {
		return
	}
	data := Component.Get(entry)
	data.AttractorX = x
	data.AttractorY = y
}

// copyConfig creates a deep copy of ParticleConfig
func copyConfig(src *ParticleConfig) *ParticleConfig {
	dst := *src

	if src.Animation.Duration.Range != nil {
		r := *src.Animation.Duration.Range
		dst.Animation.Duration.Range = &r
	}

	dst.Animation.Position = copyPositionConfig(src.Animation.Position)
	dst.Animation.Alpha = copyPropertyConfig(src.Animation.Alpha)
	dst.Animation.Scale = copyPropertyConfig(src.Animation.Scale)
	dst.Animation.Rotation = copyPropertyConfig(src.Animation.Rotation)

	if src.Animation.Color != nil {
		c := *src.Animation.Color
		dst.Animation.Color = &c
	}

	dst.Emitter = copyEmitterConfig(src.Emitter)

	return &dst
}

func copyEmitterConfig(src EmitterConfig) EmitterConfig {
	dst := src
	if src.Shape.Radius != nil {
		r := *src.Shape.Radius
		dst.Shape.Radius = &r
	}
	return dst
}

func copyPositionConfig(src PositionConfig) PositionConfig {
	dst := src
	copyRangePtr := func(r *RangeFloat) *RangeFloat {
		if r == nil {
			return nil
		}
		c := *r
		return &c
	}
	dst.StartX = copyRangePtr(src.StartX)
	dst.EndX = copyRangePtr(src.EndX)
	dst.StartY = copyRangePtr(src.StartY)
	dst.EndY = copyRangePtr(src.EndY)
	dst.Angle = copyRangePtr(src.Angle)
	dst.Distance = copyRangePtr(src.Distance)
	dst.ControlX = copyRangePtr(src.ControlX)
	dst.ControlY = copyRangePtr(src.ControlY)
	if src.X != nil {
		x := copyPropertyConfig(*src.X)
		dst.X = &x
	}
	if src.Y != nil {
		y := copyPropertyConfig(*src.Y)
		dst.Y = &y
	}
	return dst
}

func copyPropertyConfig(src PropertyConfig) PropertyConfig {
	dst := src
	if len(src.Steps) == 0 {
		return dst
	}
	dst.Steps = make([]StepConfig, len(src.Steps))
	for i, step := range src.Steps {
		dst.Steps[i] = copyStepConfig(step)
	}
	return dst
}

func copyStepConfig(src StepConfig) StepConfig {
	dst := src
	if src.FromRange != nil {
		r := *src.FromRange
		dst.FromRange = &r
	}
	if src.ToRange != nil {
		r := *src.ToRange
		dst.ToRange = &r
	}
	return dst
}
