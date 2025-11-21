package scenes

import (
	"chirashi/component"
	"chirashi/component/particle"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
)

type DebugParticleScene struct {
	world     donburi.World
	container *ecs.ECS
}

func NewDebugParticleScene() *DebugParticleScene {
	world := donburi.NewWorld()
	container := ecs.NewECS(world)

	particleSys := particle.NewSpriteSystem()
	container.AddSystem(particleSys.Update)

	spriteRender := component.NewSpriteRender()
	container.AddRenderer(0, spriteRender.Draw)

	// Create debug particle image
	img := ebiten.NewImage(8, 8)
	img.Fill(color.White)

	particle.NewParticles(world, img, 100, 100)
	return &DebugParticleScene{
		world:     world,
		container: container,
	}
}

func (s *DebugParticleScene) Update() error {
	s.container.Update()
	return nil
}

func (s *DebugParticleScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0x20, 0x20, 0x20, 0xff})
	s.container.Draw(screen)
}

func (s *DebugParticleScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 320, 240
}
