# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

chirashi is a particle system component and editor for Ebitengine (Ebiten game engine). It uses the donburi ECS (Entity Component System) library for entity management and provides a visual editor for designing particle effects.

**Key Technologies:**
- **Ebitengine (v2)**: Japanese 2D game library
- **donburi**: ECS library for entity/component management
- **gween**: Tween animation library for particle motion
- **debugui**: ImGui-style debug interface for the editor
- **Build tool**: Mage (not Make)

## Build Commands

This project uses **Mage** as the build system. All commands are defined in `magefile.go`.

### Common Commands

```bash
# Build for current platform
mage build

# Build and run
mage run

# Build for web (WASM)
mage buildweb

# Serve web build with local dev server (port 8080)
mage serve

# Run tests
mage test

# Clean build artifacts
mage clean

# Cross-platform builds
mage buildwindows
mage buildmac
mage buildlinux

# Release build (optimized)
mage buildrelease
```

## Code Architecture

### Component-Based Architecture

The codebase follows ECS patterns with donburi:

1. **`component/chirashi/`** - Core particle system implementation
   - `component.go`: Defines `SystemData`, `Instance`, and `Metrics` structs
   - `system.go`: Optimized particle lifecycle with direct rendering (spawn, update, draw)
   - `config.go`: YAML-based particle configuration structs
   - `factory.go`, `configurable_factory.go`: Factory functions for creating particles
   - `tween_factory.go`: Creates tween animation sequences from config
   - `loader.go`: Loads particle configs from YAML files or embedded assets
   - `storage.go`, `storage_desktop.go`, `storage_web.go`: Platform-specific file I/O (interface pattern)

2. **`component/`** - Generic ECS components
   - `sprite.go`, `sprite_render.go`: Sprite rendering components
   - `position.go`, `velocity.go`: Transform components
   - `tween/`: Tween-based animation components

3. **`scenes/`** - Game scenes
   - `particle_editor_scene.go`: Main editor scene with debugui interface
   - `debug_particle.go`: Simple debug scene

4. **`assets/`** - Asset management
   - `assets.go`: Embedded assets and image loading
   - `particles/*.yaml`: Particle configuration files

### Particle System Design

**Rendering Architecture:**
- **Direct Rendering**: Particles are rendered directly without intermediate sprite entities
- Single `System` handles update and draw (no separate systems)
- Optimized for minimal memory allocations and GC pressure

**Two Movement Systems:**
- **Cartesian**: Particles move using X/Y tween sequences
- **Polar**: Particles move using Angle/Distance tween sequences

**Tween-Based Animation:**
- All particle properties (position, alpha, rotation, scale) are animated via tween sequences
- Sequences are created from factory functions that support randomization via ranges
- Config defines steps with easing functions (Linear, OutCirc, InBack, etc.)

**Configuration-Driven:**
- Particle effects are defined in YAML files (`assets/particles/*.yaml`)
- Configurations include: emitter position, movement type, tween sequences, spawn parameters
- See `sample.yaml` for reference structure

**Particle Lifecycle:**
1. Spawned from pool at configured interval using O(1) free index stack
2. Initialized with tween sequences from factories
3. Updated each frame by `System.Update()` (only active particles)
4. Rendered by `System.Draw()` (only active particles)
5. Deactivated when all tween sequences finish
6. Returned to free index stack for reuse

### Performance Optimizations

**Active Index Management (O(1) operations):**
- `ActiveIndices []int`: Compact array of active particle indices
- `FreeIndices []int`: Stack of available particle indices
- Update/Draw iterate only active particles (not entire pool)
- Spawn uses O(1) pop from free stack (no linear search)

**Memory Optimizations:**
- `DrawImageOptions` pooling with `sync.Pool` (zero allocations per frame)
- Image dimensions cached in `SystemData` (no repeated `Bounds()` calls)
- Direct rendering without intermediate sprite entities

**Performance Metrics:**
The `Metrics` struct tracks:
- `UpdateTimeUs`: Update time in microseconds
- `DrawTimeUs`: Draw time in microseconds
- `SpawnCount`: Total particles spawned (cumulative)
- `DeactivateCount`: Total particles deactivated (cumulative)
- Displayed in editor's Debug Info window

**Typical Performance (1000 max particles, 10 active):**
- Update: ~10 iterations instead of 1000 (100x improvement)
- Draw: ~10 iterations instead of 1000 (100x improvement)
- Spawn: O(1) instead of O(n) linear search

### Platform-Specific Code

The `ParticleStorage` interface (`component/chirashi/storage.go`) abstracts file operations:
- `storage_desktop.go`: Uses native file I/O
- `storage_web.go`: Uses browser localStorage

Build tags control which implementation is compiled.

### Editor Architecture

The `ParticleEditorScene` provides a live particle editor with multiple debugui windows:
- General Settings: Spawn parameters, lifecycle
- Movement X/Y: Cartesian movement tweens
- Appearance: Alpha, rotation, scale tweens
- File Operations: Save/load configurations

Changes to config in the UI immediately update the particle system.

## Development Guidelines (from .agent/rules/antigravity.md)

- Design components to be lightweight and independently functional (easy to test)
- Use interface separation where appropriate (avoid over-abstraction)
- Keep the balance between simplicity and proper architecture

## Testing

Currently no test files exist. When adding tests:
```bash
mage test              # Run all tests
go test ./...          # Direct go test
go test ./component/... # Test specific package
```
