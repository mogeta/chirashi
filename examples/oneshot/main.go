package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"

	"github.com/mogeta/chirashi"
	"github.com/mogeta/chirashi/assets"
)

type Game struct {
	world     donburi.World
	container *ecs.ECS
	manager   *chirashi.ParticleManager
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

	return &Game{
		world:     world,
		container: container,
		manager:   manager,
	}
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if err := g.manager.SpawnOneShot(g.world, "sample", float32(x), float32(y), 45); err != nil {
			return err
		}
	}

	g.container.Update()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 0x10, G: 0x14, B: 0x20, A: 0xff})
	g.container.Draw(screen)
	ebitenutil.DebugPrint(screen, "Click or press Space to spawn one-shot particles")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 800, 600
}

func main() {
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("chirashi example: one-shot")

	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
