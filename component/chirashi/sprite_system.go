package chirashi

import (
	"chirashi/component"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

// SpriteSystem manages particle systems that create sprite entities
type SpriteSystem struct {
	query *donburi.Query
	cnt   int
}

func NewSpriteSystem() *SpriteSystem {
	return &SpriteSystem{
		query: donburi.NewQuery(filter.Contains(Component)),
		cnt:   0,
	}
}

func (sys *SpriteSystem) Update(ecs *ecs.ECS) {
	sys.cnt++
	for entry := range sys.query.Iter(ecs.World) {
		particleComponent := Component.Get(entry)

		spawn(ecs, sys, particleComponent)
		updateEachSprite(ecs, particleComponent)

		// Handle lifetime
		if !particleComponent.IsLoop {
			particleComponent.LifeTime--
			if particleComponent.LifeTime <= 0 {
				// Clean up all sprite entities before removing particle system
				for i := range particleComponent.ParticlePool {
					particle := &particleComponent.ParticlePool[i]
					if particle.SpriteEntity != nil && particle.SpriteEntity.Valid() {
						ecs.World.Remove(particle.SpriteEntity.Entity())
					}
				}
				ecs.World.Remove(entry.Entity())
			}
		}
	}
}

func updateEachSprite(ecs *ecs.ECS, particleComponent *SystemData) {
	// Update existing particles and their sprite entities
	for i := range particleComponent.ParticlePool {
		particle := &particleComponent.ParticlePool[i]
		if !particle.Active {
			continue
		}

		// Update particle animation sequences
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

		f4 := true
		if particle.SequenceRotate != nil {
			r, _, finished := particle.SequenceRotate.Update(deltaTime)
			particle.Rotation = float64(r)
			f4 = finished
		}

		f5 := true
		if particle.SequenceScale != nil {
			s, _, finished := particle.SequenceScale.Update(deltaTime)
			particle.Scale = float64(s)
			f5 = finished
		}

		// Update corresponding sprite entity if it exists
		if particle.SpriteEntity != nil && particle.SpriteEntity.Valid() {
			// Update sprite data
			spriteData := component.NewParticleSprite(
				particleComponent.SourceImage,
				particle.Alpha,
				particle.Rotation,
				particle.Scale,
			)
			donburi.SetValue(particle.SpriteEntity, component.Sprite, spriteData)

			// Update position
			positionData := component.PositionData{
				X: particle.Position.X,
				Y: particle.Position.Y,
			}
			donburi.SetValue(particle.SpriteEntity, component.Position, positionData)
		}

		// Deactivate particle if all sequences finished
		if f1 && f2 && f3 && f4 && f5 {
			particle.Active = false
			particleComponent.ActiveCount--

			// Remove sprite entity
			if particle.SpriteEntity != nil && particle.SpriteEntity.Valid() {
				ecs.World.Remove(particle.SpriteEntity.Entity())
				particle.SpriteEntity = nil
			}
		}
	}
}

func spawn(ecs *ecs.ECS, sys *SpriteSystem, particleComponent *SystemData) {
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
				particle.SequenceX = particleComponent.SequenceFactoryX()
				particle.SequenceY = particleComponent.SequenceFactoryY()

				if particleComponent.SequenceFactoryAlpha != nil {
					particle.SequenceAlpha = particleComponent.SequenceFactoryAlpha()
				}
				if particleComponent.SequenceFactoryR != nil {
					particle.SequenceRotate = particleComponent.SequenceFactoryR()
				}
				if particleComponent.SequenceFactoryS != nil {
					particle.SequenceScale = particleComponent.SequenceFactoryS()
				}

				// Create sprite entity
				spriteEntity := ecs.World.Entry(ecs.World.Create(component.Sprite, component.Position))

				spriteData := component.NewParticleSprite(
					particleComponent.SourceImage,
					particle.Alpha,
					particle.Rotation,
					particle.Scale,
				)
				donburi.SetValue(spriteEntity, component.Sprite, spriteData)

				positionData := component.PositionData{
					X: particle.Position.X,
					Y: particle.Position.Y,
				}
				donburi.SetValue(spriteEntity, component.Position, positionData)

				// Store sprite entity reference
				particle.SpriteEntity = spriteEntity
				particleComponent.ActiveCount++
				break
			}
		}
	}
}
