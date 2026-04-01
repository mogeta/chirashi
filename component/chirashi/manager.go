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
	return createParticleEntityFromConfig(world, m.shader, m.image, config, x, y)
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
	if !world.Valid(entity) {
		return
	}
	entry := world.Entry(entity)
	if !entry.HasComponent(Component) {
		return
	}
	data := Component.Get(entry)
	data.AttractorX = x
	data.AttractorY = y
}

// SetEmitterPosition updates the emitter origin for a particle entity.
// In local emitter space, active particles move with the emitter.
func SetEmitterPosition(world donburi.World, entity donburi.Entity, x, y float32) {
	if !world.Valid(entity) {
		return
	}
	entry := world.Entry(entity)
	data := Component.Get(entry)
	dx := x - data.EmitterX
	dy := y - data.EmitterY
	data.EmitterX = x
	data.EmitterY = y
	shiftActiveParticlesForEmitterDelta(data, dx, dy)
	if data.Trail.Params.Enabled && data.Trail.Params.LocalSpace {
		if data.Trail.Params.Mode == "particle" {
			for i := range data.ParticlePool {
				shiftTrailPoints(data.ParticlePool[i].TrailPoints, dx, dy)
			}
			for i := range data.Trail.Runtime.Ghosts {
				shiftTrailPoints(data.Trail.Runtime.Ghosts[i].Points, dx, dy)
			}
		} else {
			shiftTrailPoints(data.Trail.Runtime.Points, dx, dy)
		}
	}
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
	if src.Trail != nil {
		trail := *src.Trail
		if src.Trail.Color != nil {
			c := *src.Trail.Color
			trail.Color = &c
		}
		dst.Trail = &trail
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
	if src.Vector != nil {
		vector := *src.Vector
		if src.Vector.Rect != nil {
			rect := *src.Vector.Rect
			vector.Rect = &rect
		}
		if src.Vector.Polyline != nil {
			polyline := *src.Vector.Polyline
			if len(src.Vector.Polyline.Points) > 0 {
				polyline.Points = append([]EmitterVectorPoint(nil), src.Vector.Polyline.Points...)
			}
			vector.Polyline = &polyline
		}
		dst.Vector = &vector
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
	if src.Flow != nil {
		flow := *src.Flow
		flow.Strength = copyRangePtr(src.Flow.Strength)
		dst.Flow = &flow
	}
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
