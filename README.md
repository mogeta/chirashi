# chirashi

[![CI](https://github.com/mogeta/chirashi/actions/workflows/ci.yml/badge.svg)](https://github.com/mogeta/chirashi/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/mogeta/chirashi.svg)](https://pkg.go.dev/github.com/mogeta/chirashi)
[![Go Report Card](https://goreportcard.com/badge/github.com/mogeta/chirashi)](https://goreportcard.com/report/github.com/mogeta/chirashi)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/mogeta/chirashi)](https://github.com/mogeta/chirashi/releases)

[日本語はこちら](README_JP.md)

`chirashi` is a GPU-oriented particle component and editor for Ebitengine.
It uses [donburi](https://github.com/yohamta/donburi) as the ECS library.

## Features

- GPU batch rendering with `DrawTrianglesShader`
- Built-in editor for real-time parameter tuning
- `polar` / `cartesian` position modes
- Property animation with easing and multi-step sequences
- Save/load particle configs as YAML
- donburi (ECS) integration

## Requirements

- Go 1.24+
- A platform supported by Ebitengine (desktop or web build target)

## Library Quick Start

```go
import (
    "github.com/mogeta/chirashi"
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/yohamta/donburi"
    "github.com/yohamta/donburi/ecs"
)

world := donburi.NewWorld()
gameECS := ecs.NewECS(world)

particleSystem := chirashi.NewSystem()
gameECS.AddSystem(particleSystem.Update)
gameECS.AddRenderer(0, particleSystem.Draw)

shader, _ := ebiten.NewShader([]byte("..."))
image := ebiten.NewImage(8, 8)

manager := chirashi.NewParticleManager(shader, image)
_ = manager.Preload("sample", "assets/particles/sample.yaml")
_, _ = manager.SpawnLoop(world, "sample", 640, 480)
```

Runnable examples:

```bash
go run ./examples/minimal
go run ./examples/oneshot
```

Web example build:

```bash
GOOS=js GOARCH=wasm go build -o build/web/examples_web.wasm ./examples/web
```

## Config Format (YAML)

Particle effects are defined in YAML (see `assets/particles/*.yaml`):
Full schema and compatibility policy: `docs/CONFIG_SCHEMA.md`.

```yaml
name: "sample"
description: "sample particle"

emitter:
  x: 0
  y: 0

animation:
  duration:
    value: 1.0
  position:
    type: "polar"
    angle: { min: 0.0, max: 6.28 }
    distance: { min: 50, max: 150 }
    easing: "OutCirc"
  alpha:
    start: 1.0
    end: 0.0
    easing: "Linear"
  scale:
    start: 1.0
    end: 0.5
    easing: "OutQuad"
  rotation:
    start: 0.0
    end: 3.14
    easing: "Linear"

spawn:
  interval: 1
  particles_per_spawn: 10
  max_particles: 1000
  is_loop: true
```

## Editor

Run the editor directly:

```bash
go run ./cmd/chirashi-editor
```

Or use Mage tasks:

```bash
mage run
```

Available tasks:

```bash
mage -l
```

Key build commands:

```bash
mage build         # Build native binary to build/
mage buildRelease  # Optimized build
mage buildWeb      # Build WASM files into build/web
mage serve         # Build web assets and serve on localhost:8080
mage test          # Run go test ./...
```

## Demo

You can try it in your browser:
[https://muzigen.net/ebiten/chirashi/](https://muzigen.net/ebiten/chirashi/)

## License

MIT License
