# Changelog

All notable changes to this project are documented in this file.

The format is based on Keep a Changelog and this project follows Semantic Versioning.

## [Unreleased]

### Added
- Public root API import path (`github.com/mogeta/chirashi`) via root re-exports.
- Backward-compatible compatibility package at `github.com/mogeta/chirashi/particle` (deprecated in docs).
- Library packaging checklist and public API policy docs.
- YAML config schema/contract docs with examples.
- Runnable examples:
  - `examples/minimal`
  - `examples/oneshot`
  - `examples/web`
- Unit tests for config validation, easing, sequence behavior, and system spawn/lifetime logic.
- GitHub Actions CI workflow (`go test`, `go vet`, `staticcheck`).

### Changed
- Repository layout split into app/editor and reusable library parts:
  - editor entrypoint moved to `cmd/chirashi-editor`
  - editor implementation moved to `internal/editor`
- README/README_JP restructured around library-first usage.
- Library runtime logging reduced in core package codepaths.

### Fixed
- Web storage behavior now returns explicit errors for unsupported file save/load operations.

## [0.1.0] - TBD

Initial public release target.

