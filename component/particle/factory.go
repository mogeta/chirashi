package particle

import (
	"fmt"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
	"github.com/yohamta/donburi"
)

func rangeFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func NewParticles(w donburi.World, image *ebiten.Image, x, y float64) {
	particles := w.Entry(w.Create(Component))

	const maxParticles = 1000

	var d = SystemData{
		ParticlePool:    make([]Instance, maxParticles),
		EmitterPosition: Position{X: x, Y: y},
		SequenceFactoryX: func() *gween.Sequence {
			targetx := rangeFloat(x-300, x+300)
			return gween.NewSequence(
				gween.New(float32(x),
					float32(targetx),
					2, ease.OutCirc))
		},
		SequenceFactoryY: func() *gween.Sequence {
			targety := rangeFloat(y-300, y+300)
			return gween.NewSequence(
				gween.New(float32(y),
					float32(targety),
					2, ease.OutCirc))
		},
		SequenceFactoryS: func() *gween.Sequence {
			return gween.NewSequence(
				gween.New(float32(0.1), // Start small
					float32(1.0),       // Scale up to normal
					0.5, ease.OutBack), // Quick scale up
				gween.New(float32(1.0), // Stay at normal size
					float32(0.8),      // Scale down slightly
					1.5, ease.InCirc)) // Slow scale down
		},
		SpawnInterval:     1,
		ParticlesPerSpawn: 5, // Spawn more particles per interval
		MaxParticles:      maxParticles,
		SourceImage:       image,
		ActiveCount:       0,
		IsLoop:            false, // Don't loop forever by default
		LifeTime:          300,   // 5 seconds at 60fps
	}
	donburi.SetValue(particles, Component, d)

	// Debug logging
	fmt.Printf("Debug: NewParticles created at position (%.1f, %.1f) with image size %dx%d\n",
		x, y, image.Bounds().Dx(), image.Bounds().Dy())
}
