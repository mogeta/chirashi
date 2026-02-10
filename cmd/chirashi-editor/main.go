package main

import (
	"log"

	"github.com/mogeta/chirashi/internal/editor"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	ebiten.SetWindowSize(1280, 960)
	ebiten.SetWindowTitle("Chirashi Particle Editor")

	game := editor.NewParticleEditorScene()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
