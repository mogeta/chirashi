package main

import (
	"flag"
	"log"

	"chirashi/scenes"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	useAburi := flag.Bool("aburi", false, "Use GPU-based aburi particle system")
	flag.Parse()

	ebiten.SetWindowSize(1280, 960)

	var game ebiten.Game
	if *useAburi {
		ebiten.SetWindowTitle("Aburi Particle Editor (GPU)")
		game = scenes.NewAburiEditorScene()
	} else {
		ebiten.SetWindowTitle("Chirashi Particle Editor")
		game = scenes.NewParticleEditorScene()
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
