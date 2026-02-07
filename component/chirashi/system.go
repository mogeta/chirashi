package chirashi

import (
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

// drawOptsPool pools DrawImageOptions to reduce GC pressure
var drawOptsPool = sync.Pool{
	New: func() interface{} {
		return &ebiten.DrawImageOptions{}
	},
}

// System manages particle systems with direct rendering
type System struct {
	query *donburi.Query
	cnt   int
}

func NewSystem() *System {
	return &System{
		query: donburi.NewQuery(filter.Contains(Component)),
		cnt:   0,
	}
}

func (sys *System) Update(ecs *ecs.ECS) {
	sys.cnt++
	for entry := range sys.query.Iter(ecs.World) {
		particleComponent := Component.Get(entry)

		// Track update time
		startTime := time.Now()

		sys.spawn(particleComponent)
		sys.updateParticles(particleComponent)

		// Update metrics
		particleComponent.Metrics.UpdateTimeUs = time.Since(startTime).Microseconds()
		particleComponent.Metrics.FrameCount++

		// Handle lifetime
		if !particleComponent.IsLoop {
			particleComponent.LifeTime--
			if particleComponent.LifeTime <= 0 {
				ecs.World.Remove(entry.Entity())
			}
		}
	}
}

func (sys *System) updateParticles(particleComponent *SystemData) {
	deltaTime := float32(1.0 / float64(ebiten.TPS()))

	// Track indices to remove (particles that finished)
	indicesToRemove := []int{}

	// Iterate only active particles
	for i := 0; i < len(particleComponent.ActiveIndices); i++ {
		particleIdx := particleComponent.ActiveIndices[i]
		particle := &particleComponent.ParticlePool[particleIdx]

		// Update movement based on type
		f1, f2 := true, true
		if particleComponent.MovementType == "polar" {
			// Polar Movement
			angle, _, finished1 := particle.SequenceAngle.Update(deltaTime)
			dist, _, finished2 := particle.SequenceDist.Update(deltaTime)
			f1, f2 = finished1, finished2

			// Convert angle to radians (assuming config is in degrees)
			rad := float64(angle) * math.Pi / 180.0

			// Calculate position relative to emitter
			particle.Position.X = particleComponent.EmitterPosition.X + float64(dist)*math.Cos(rad)
			particle.Position.Y = particleComponent.EmitterPosition.Y + float64(dist)*math.Sin(rad)
		} else {
			// Cartesian Movement (Default)
			x, _, finished1 := particle.SequenceX.Update(deltaTime)
			y, _, finished2 := particle.SequenceY.Update(deltaTime)
			f1, f2 = finished1, finished2
			particle.Position.X = float64(x)
			particle.Position.Y = float64(y)
		}

		// Update alpha
		f3 := true
		if particle.SequenceAlpha != nil {
			a, _, finished := particle.SequenceAlpha.Update(deltaTime)
			particle.Alpha = float64(a)
			f3 = finished
		}

		// Update rotation
		f4 := true
		if particle.SequenceRotate != nil {
			r, _, finished := particle.SequenceRotate.Update(deltaTime)
			particle.Rotation = float64(r)
			f4 = finished
		}

		// Update scale
		f5 := true
		if particle.SequenceScale != nil {
			s, _, finished := particle.SequenceScale.Update(deltaTime)
			particle.Scale = float64(s)
			f5 = finished
		}

		// Mark for removal if all sequences finished
		if f1 && f2 && f3 && f4 && f5 {
			particle.Active = false
			indicesToRemove = append(indicesToRemove, i)
			particleComponent.ActiveCount--
			particleComponent.Metrics.DeactivateCount++
			// Return to free indices pool
			particleComponent.FreeIndices = append(particleComponent.FreeIndices, particleIdx)
		}
	}

	// Remove finished particles from active indices (iterate backwards to avoid index shift issues)
	for i := len(indicesToRemove) - 1; i >= 0; i-- {
		removeIdx := indicesToRemove[i]
		// Swap with last element and truncate
		lastIdx := len(particleComponent.ActiveIndices) - 1
		particleComponent.ActiveIndices[removeIdx] = particleComponent.ActiveIndices[lastIdx]
		particleComponent.ActiveIndices = particleComponent.ActiveIndices[:lastIdx]
	}
}

func (sys *System) spawn(particleComponent *SystemData) {
	// Spawn new particles
	if sys.cnt%particleComponent.SpawnInterval == 0 {
		for i := 0; i < particleComponent.ParticlesPerSpawn && particleComponent.ActiveCount < particleComponent.MaxParticles; i++ {
			// O(1) free index retrieval
			if len(particleComponent.FreeIndices) == 0 {
				break // No free particles available
			}

			// Pop from free indices stack
			freeIdx := particleComponent.FreeIndices[len(particleComponent.FreeIndices)-1]
			particleComponent.FreeIndices = particleComponent.FreeIndices[:len(particleComponent.FreeIndices)-1]

			particle := &particleComponent.ParticlePool[freeIdx]

			// Initialize particle
			particle.Position.X = particleComponent.EmitterPosition.X
			particle.Position.Y = particleComponent.EmitterPosition.Y
			particle.Alpha = 1
			particle.Rotation = 0.0
			particle.Scale = 1.0
			particle.Active = true

			// Create movement sequences based on type
			if particleComponent.MovementType == "polar" {
				particle.SequenceAngle = particleComponent.SequenceFactoryAngle()
				particle.SequenceDist = particleComponent.SequenceFactoryDist()
			} else {
				particle.SequenceX = particleComponent.SequenceFactoryX()
				particle.SequenceY = particleComponent.SequenceFactoryY()
			}

			// Create appearance sequences
			if particleComponent.SequenceFactoryAlpha != nil {
				particle.SequenceAlpha = particleComponent.SequenceFactoryAlpha()
			}
			if particleComponent.SequenceFactoryR != nil {
				particle.SequenceRotate = particleComponent.SequenceFactoryR()
			}
			if particleComponent.SequenceFactoryS != nil {
				particle.SequenceScale = particleComponent.SequenceFactoryS()
			}

			// Add to active indices
			particleComponent.ActiveIndices = append(particleComponent.ActiveIndices, freeIdx)
			particleComponent.ActiveCount++
			particleComponent.Metrics.SpawnCount++
		}
	}
}

// Draw renders all particles directly to the screen
func (sys *System) Draw(ecs *ecs.ECS, screen *ebiten.Image) {
	for entry := range sys.query.Iter(ecs.World) {
		particleComponent := Component.Get(entry)

		if particleComponent.SourceImage == nil {
			continue
		}

		// Track draw time
		startTime := time.Now()

		// Use cached image dimensions
		baseWidth := particleComponent.ImageWidth
		baseHeight := particleComponent.ImageHeight

		// Draw only active particles
		for _, particleIdx := range particleComponent.ActiveIndices {
			particle := &particleComponent.ParticlePool[particleIdx]

			// Get DrawImageOptions from pool
			opts := drawOptsPool.Get().(*ebiten.DrawImageOptions)

			// Reset to default state
			opts.GeoM.Reset()
			opts.ColorScale.Reset()

			width := baseWidth
			height := baseHeight

			// Apply scale
			if particle.Scale > 0 {
				opts.GeoM.Scale(particle.Scale, particle.Scale)
				width *= particle.Scale
				height *= particle.Scale
			}

			// Apply rotation around center
			if particle.Rotation != 0 {
				centerX := width / 2
				centerY := height / 2
				opts.GeoM.Translate(-centerX, -centerY)
				opts.GeoM.Rotate(particle.Rotation)
				opts.GeoM.Translate(centerX, centerY)
			}

			// Apply position (center-anchored)
			opts.GeoM.Translate(particle.Position.X-width/2, particle.Position.Y-height/2)

			// Apply alpha
			if particle.Alpha >= 0 && particle.Alpha <= 1.0 {
				opts.ColorScale.Scale(
					float32(particle.Alpha),
					float32(particle.Alpha),
					float32(particle.Alpha),
					float32(particle.Alpha),
				)
			}

			// Use additive blending for particles
			opts.CompositeMode = ebiten.CompositeModeSourceOver

			screen.DrawImage(particleComponent.SourceImage, opts)

			// Return to pool
			drawOptsPool.Put(opts)
		}

		// Update draw metrics
		particleComponent.Metrics.DrawTimeUs = time.Since(startTime).Microseconds()
	}
}
