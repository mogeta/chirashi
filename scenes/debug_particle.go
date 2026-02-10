package scenes

import (
	"chirashi/component/chirashi"
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

	particleSys := chirashi.NewSystem()
	container.AddSystem(particleSys.Update)
	container.AddRenderer(0, particleSys.Draw)

	// Create debug particle image
	img := ebiten.NewImage(8, 8)
	img.Fill(color.White)

	chirashi.NewParticles(world, img, 100, 100)
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
