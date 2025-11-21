package particle

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

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

		// Update existing particles
		for i := range particleComponent.ParticlePool {
			particle := &particleComponent.ParticlePool[i]
			if !particle.Active {
				continue
			}

			deltaTime := float32(1.0 / float64(ebiten.TPS()))
			x, _, f1 := particle.SequenceX.Update(deltaTime)
			y, _, f2 := particle.SequenceY.Update(deltaTime)

			particle.Position.X = float64(x)
			particle.Position.Y = float64(y)

			f3 := true
			if particle.SequenceAlpha != nil {
				a, _, finished := particle.SequenceAlpha.Update(deltaTime)
				particle.Alpha = float64(a)
				f3 = finished
			}

			// Update rotation if sequence exists
			f4 := true
			if particle.SequenceRotate != nil {
				r, _, finished := particle.SequenceRotate.Update(deltaTime)
				particle.Rotation = float64(r)
				f4 = finished
			}

			// Update scale if sequence exists
			f5 := true
			if particle.SequenceScale != nil {
				s, _, finished := particle.SequenceScale.Update(deltaTime)
				particle.Scale = float64(s)
				f5 = finished
			}

			// All sequences must finish before deactivating particle
			if f1 && f2 && f3 && f4 && f5 {
				particle.Active = false
				particleComponent.ActiveCount--
			}
		}

		// Spawn new particles
		if sys.cnt%particleComponent.SpawnInterval == 0 {
			for i := 0; i < particleComponent.ParticlesPerSpawn && particleComponent.ActiveCount < particleComponent.MaxParticles; i++ {
				for j := range particleComponent.ParticlePool {
					particle := &particleComponent.ParticlePool[j]
					if particle.Active {
						continue
					}

					particle.Position.X = particleComponent.EmitterPosition.X
					particle.Position.Y = particleComponent.EmitterPosition.Y
					particle.Alpha = 0
					particle.Rotation = 0.0
					particle.Scale = 0.1 // Start with small but visible scale
					particle.Active = true
					particle.SequenceX = particleComponent.SequenceFactoryX()
					particle.SequenceY = particleComponent.SequenceFactoryY()

					if particleComponent.SequenceFactoryAlpha != nil {
						particle.SequenceAlpha = particleComponent.SequenceFactoryAlpha()
					}

					// Rotation sequence (use factory if available)
					if particleComponent.SequenceFactoryR != nil {
						particle.SequenceRotate = particleComponent.SequenceFactoryR()
					}

					// Scale sequence (use factory if available)
					if particleComponent.SequenceFactoryS != nil {
						particle.SequenceScale = particleComponent.SequenceFactoryS()
					}

					particleComponent.ActiveCount++
					break
				}
			}
		}
		if !particleComponent.IsLoop {
			particleComponent.LifeTime--
			if particleComponent.LifeTime <= 0 {
				ecs.World.Remove(entry.Entity())
			}
		}

	}
}
