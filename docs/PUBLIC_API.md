# Public API Policy

This document defines the recommended public import path and stable API surface.

## Public Import Path

Use:

```go
import "github.com/mogeta/chirashi"
```

Compatibility import:

```go
import "github.com/mogeta/chirashi/particle"
```

`component/chirashi` remains available but is treated as implementation-oriented.

## Stable API (v0 Target)

The following are intended as the primary API for consumers:

- Runtime setup
  - `chirashi.NewSystem`
  - `chirashi.NewParticleManager`
- Spawning/helpers
  - `chirashi.NewParticlesFromConfig`
  - `chirashi.NewParticlesFromFile`
- Configuration
  - `chirashi.ParticleConfig` and nested config types
  - `chirashi.NewConfigLoader`
  - `chirashi.GetConfigLoader`
- ECS integration
  - `chirashi.Component`

## Compatibility Notes

- During `v0.x`, minor breaking changes are still possible, but should be minimized.
- Breaking API changes should be documented in release notes.
- New features should be added to root package `chirashi` first.
