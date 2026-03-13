package main

import (
	"log"

	"github.com/mogeta/chirashi/internal/editor"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	ebiten.SetWindowSize(1280, 960)
	ebiten.SetWindowTitle("Chirashi Particle Editor")

	game, err := editor.NewParticleEditorScene()
	if err != nil {
		log.Fatalf("Failed to initialize editor: %v", err)
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
