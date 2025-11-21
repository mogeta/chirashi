package main

import (
	"log"

	"chirashi/scenes"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Debug Particle")
	if err := ebiten.RunGame(scenes.NewParticleEditorScene()); err != nil {
		log.Fatal(err)
	}
}
