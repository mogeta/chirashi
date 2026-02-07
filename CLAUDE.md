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
   - `component.go`: Defines `SystemData` and `Instance` structs for particle data
   - `system.go`: Update logic for particle lifecycle (spawn, animate, deactivate)
   - `sprite_system.go`: Alternative rendering system using sprite entities
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
1. Spawned from pool at configured interval
2. Initialized with tween sequences from factories
3. Updated each frame by `System.Update()`
4. Deactivated when all tween sequences finish
5. Returned to pool for reuse

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
