# chirashi

[![CI](https://github.com/mogeta/chirashi/actions/workflows/ci.yml/badge.svg)](https://github.com/mogeta/chirashi/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/mogeta/chirashi.svg)](https://pkg.go.dev/github.com/mogeta/chirashi)
[![Go Report Card](https://goreportcard.com/badge/github.com/mogeta/chirashi)](https://goreportcard.com/report/github.com/mogeta/chirashi)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/mogeta/chirashi)](https://github.com/mogeta/chirashi/releases)

[日本語はこちら](README_JP.md)

`chirashi` is a GPU-oriented particle library and editor for Ebitengine.
It uses [donburi](https://github.com/yohamta/donburi) as the ECS library.
![chirashi editor screenshot](docs/images/editor.png)

## Demo

You can try it in your browser:
[https://muzigen.net/ebiten/chirashi/](https://muzigen.net/ebiten/chirashi/)

## Features

- GPU batch rendering with `DrawTrianglesShader`
- Pool-based particle lifecycle with compact active/free index management
- Built-in editor for real-time parameter tuning and YAML save/load
- Position modes: `cartesian`, `polar`, `attractor`
- Emitter shapes: `point`, `circle`, `box`, `line`
- Property animation with easing and multi-step sequences
- Runtime attractor target updates for UI/item-collection effects
- Save/load particle configs as YAML
- donburi (ECS) integration

## Status

`chirashi` is usable as a game particle library today, but it is still in `v0.x`.
Expect iterative improvements to config validation, editor UX, and public API polish.

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
  shape:
    type: "circle"
    radius: { min: 0, max: 48 }
    start_angle: 0
    end_angle: 6.2831855

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

Config highlights:

- `emitter.shape` controls where particles are spawned around the emitter origin.
- `animation.position.type: attractor` curves particles toward a runtime target.
- `PropertyConfig` supports both simple `start/end/easing` and multi-step `sequence` mode.
- Example effects are available under `assets/particles/`.

Notable samples:

- `sample.yaml`: basic radial burst
- `collect_coins.yaml`: attractor-based pickup flow into UI
- `rune_ring.yaml`: circular ring emission
- `fountain_arc.yaml`: arc-shaped directional spray
- `muzzle_flash_cone.yaml`: short forward cone burst
- `barrier_edge.yaml`: perimeter emission around a box

## Runtime Notes

- The library is optimized around spawn-time randomization and batched draw submission.
- Shape sampling happens when particles spawn; it does not add per-frame draw cost.
- `ParticleManager.SpawnLoop` returns an entity so the effect can be removed manually later.
- `SetAttractor` can be called each frame for moving attractor targets.

## Public API

Primary import path:

```go
import "github.com/mogeta/chirashi"
```

See `docs/PUBLIC_API.md` for the intended stable surface during `v0.x`.

## Editor

The editor is included as a tuning tool and sample app for authoring YAML configs.

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

## Examples

- `examples/minimal`: looped particle effect
- `examples/oneshot`: one-shot spawning on input
- `examples/web`: WASM/browser example

## License

MIT License

## Release Notes

- Changelog: `CHANGELOG.md`
- Release process: `docs/RELEASE_PROCESS.md`
- Migration notes: `docs/MIGRATIONS.md`
