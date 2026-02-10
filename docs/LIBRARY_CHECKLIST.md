# Library Packaging Checklist

This project can already be used as a Go module.  
This checklist focuses on making it clean and stable as a reusable library.

## Phase 0 (done)

- [x] Set module path to a public import path (`github.com/mogeta/chirashi`)
- [x] Update in-repo imports to use the module path
- [x] Confirm compile via `go test ./...`

## Phase 1: Repository Layout

- [x] Move executable entrypoint to `cmd/chirashi-editor/main.go`
- [x] Keep reusable code outside `cmd/` (`component/chirashi`, `assets`, etc.)
- [x] Move editor-only scene code under an editor package (`internal/editor`)
- [x] Ensure root package is not required for library consumers

Definition of done:
- `go run ./cmd/chirashi-editor` works
- library can be imported without pulling editor entrypoint assumptions

## Phase 2: Public API Surface

- [x] Decide stable public package path:
  - Selected: root package `github.com/mogeta/chirashi`
  - Compatibility: `github.com/mogeta/chirashi/particle` remains available
- [x] Document which types/functions are public and stable:
  - `ParticleManager`
  - `System`
  - `ParticleConfig` and related config structs
  - `GetConfigLoader` / loader API
- [ ] Hide implementation details (move unstable code to `internal/` or unexport)

Definition of done:
- A small public API list exists in docs
- Breaking changes can be identified clearly

## Phase 3: Configuration Contract

- [x] Write YAML schema expectations:
  - required fields
  - optional fields
  - defaults
  - validation and error behavior
- [x] Add schema examples for:
  - one-shot effect
  - looping effect
  - polar mode
  - sequence-based animation
- [x] Clarify compatibility policy for config format

Definition of done:
- users can author config files without reading source code

## Phase 4: Logging and Errors

- [x] Remove direct stdout/stderr logs from library runtime paths
- [x] Return structured `error` values instead of printing diagnostics in core packages
- [x] Keep debug logging only in app/editor side

Definition of done:
- library behaves silently unless caller handles errors/logging

## Phase 5: Documentation and Examples

- [x] Restructure README around library usage first, editor second
- [x] Add GoDoc comments to exported types/functions in `component/chirashi`
- [x] Add runnable examples:
  - `examples/minimal`
  - `examples/oneshot`
  - `examples/web`

Definition of done:
- a new project can copy an example and run in minutes

## Phase 6: Tests and CI

- [x] Add tests for:
  - config loading/validation
  - easing behavior basics
  - sequence snapshot/evaluation
  - spawn/lifetime behavior (logic level)
- [x] Add CI workflow:
  - `go test ./...`
  - `go vet ./...`
  - `staticcheck ./...`

Definition of done:
- pull requests have automatic quality checks

## Phase 7: Release Management

- [x] Choose versioning policy (start with `v0.x` until API is stable)
- [x] Create `CHANGELOG.md`
- [x] Tag releases (`v0.1.0`, etc.) after compatibility checkpoints
- [x] Add migration notes when public API/config changes

Definition of done:
- external consumers can track upgrade risk

## Suggested Execution Order

1. Phase 1 (layout)
2. Phase 2 (public API)
3. Phase 4 (logging/errors)
4. Phase 5 (docs/examples)
5. Phase 3 (config contract) in parallel with examples
6. Phase 6 (tests/CI)
7. Phase 7 (release workflow)
