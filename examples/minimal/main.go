package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"

	"github.com/mogeta/chirashi"
	"github.com/mogeta/chirashi/assets"
)

type Game struct {
	container *ecs.ECS
}

func NewGame() *Game {
	world := donburi.NewWorld()
	container := ecs.NewECS(world)

	particleSystem := chirashi.NewSystem()
	container.AddSystem(particleSystem.Update)
	container.AddRenderer(0, particleSystem.Draw)

	shader, err := ebiten.NewShader(assets.ParticleShader)
	if err != nil {
		log.Fatalf("failed to create particle shader: %v", err)
	}

	img := ebiten.NewImage(8, 8)
	img.Fill(color.White)

	manager := chirashi.NewParticleManager(shader, img)
	if err := manager.PreloadFromBytes("sample", assets.SampleParticleConfig); err != nil {
		log.Fatalf("failed to preload particle config: %v", err)
	}

	if _, err := manager.SpawnLoop(world, "sample", 320, 240); err != nil {
		log.Fatalf("failed to spawn looping effect: %v", err)
	}

	return &Game{container: container}
}

func (g *Game) Update() error {
	g.container.Update()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 0x12, G: 0x12, B: 0x18, A: 0xff})
	g.container.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 640, 480
}

func main() {
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("chirashi example: minimal")

	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
