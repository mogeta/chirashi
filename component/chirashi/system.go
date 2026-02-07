package chirashi

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

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

		sys.spawn(particleComponent)
		sys.updateParticles(particleComponent)

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

	// Update existing particles
	for i := range particleComponent.ParticlePool {
		particle := &particleComponent.ParticlePool[i]
		if !particle.Active {
			continue
		}

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

		// Deactivate particle if all sequences finished
		if f1 && f2 && f3 && f4 && f5 {
			particle.Active = false
			particleComponent.ActiveCount--
		}
	}
}

func (sys *System) spawn(particleComponent *SystemData) {
	// Spawn new particles
	if sys.cnt%particleComponent.SpawnInterval == 0 {
		for i := 0; i < particleComponent.ParticlesPerSpawn && particleComponent.ActiveCount < particleComponent.MaxParticles; i++ {
			for j := range particleComponent.ParticlePool {
				particle := &particleComponent.ParticlePool[j]
				if particle.Active {
					continue
				}

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

				particleComponent.ActiveCount++
				break
			}
		}
	}
}

// Draw renders all particles directly to the screen
func (sys *System) Draw(ecs *ecs.ECS, screen *ebiten.Image) {
	for entry := range sys.query.Iter(ecs.World) {
		particleComponent := Component.Get(entry)

		// Draw all active particles
		for i := range particleComponent.ParticlePool {
			particle := &particleComponent.ParticlePool[i]
			if !particle.Active || particleComponent.SourceImage == nil {
				continue
			}

			opts := &ebiten.DrawImageOptions{}

			// Get image dimensions
			bounds := particleComponent.SourceImage.Bounds()
			width := float64(bounds.Dx())
			height := float64(bounds.Dy())

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
		}
	}
}
