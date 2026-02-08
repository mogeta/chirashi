package aburi

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
	mutex   sync.RWMutex
}

// NewParticleManager creates a new particle manager
func NewParticleManager(shader *ebiten.Shader, image *ebiten.Image) *ParticleManager {
	return &ParticleManager{
		shader:  shader,
		image:   image,
		configs: make(map[string]*ParticleConfig),
	}
}

// Preload loads and caches a particle configuration
func (m *ParticleManager) Preload(name string, path string) error {
	config, err := GetConfigLoader().LoadConfig(path)
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
	config, err := GetConfigLoader().LoadConfigFromBytes(data, name)
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

// copyConfig creates a deep copy of ParticleConfig
func copyConfig(src *ParticleConfig) *ParticleConfig {
	dst := *src // Shallow copy

	// Deep copy nested pointers
	if src.Animation.Duration.Range != nil {
		r := *src.Animation.Duration.Range
		dst.Animation.Duration.Range = &r
	}

	if src.Animation.Position.StartX != nil {
		r := *src.Animation.Position.StartX
		dst.Animation.Position.StartX = &r
	}
	if src.Animation.Position.EndX != nil {
		r := *src.Animation.Position.EndX
		dst.Animation.Position.EndX = &r
	}
	if src.Animation.Position.StartY != nil {
		r := *src.Animation.Position.StartY
		dst.Animation.Position.StartY = &r
	}
	if src.Animation.Position.EndY != nil {
		r := *src.Animation.Position.EndY
		dst.Animation.Position.EndY = &r
	}
	if src.Animation.Position.Angle != nil {
		r := *src.Animation.Position.Angle
		dst.Animation.Position.Angle = &r
	}
	if src.Animation.Position.Distance != nil {
		r := *src.Animation.Position.Distance
		dst.Animation.Position.Distance = &r
	}

	if src.Animation.Color != nil {
		c := *src.Animation.Color
		dst.Animation.Color = &c
	}

	return &dst
}
