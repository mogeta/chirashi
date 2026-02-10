# Release Process

This document defines how releases are created for `github.com/mogeta/chirashi`.

## Versioning Policy

- Use Semantic Versioning (`MAJOR.MINOR.PATCH`).
- Current policy: `v0.x` while API may still evolve.
- For `v0.x`:
  - `MINOR` can include breaking changes.
  - `PATCH` should be backward-compatible bug fixes/docs/test changes.
- Move to `v1.0.0` when public API and config format are considered stable.

## Release Checklist

1. Ensure CI passes on `main` (`go test`, `go vet`, `staticcheck`).
2. Update `CHANGELOG.md`:
   - Move entries from `[Unreleased]` to a versioned section.
   - Add release date.
3. If there are breaking changes:
   - Add/update migration notes in `docs/MIGRATIONS.md`.
4. Create and push an annotated tag:
   - `git tag -a vX.Y.Z -m "vX.Y.Z"`
   - `git push origin vX.Y.Z`
5. Create a GitHub Release using the tag and changelog summary.

## Tagging Rules

- Tag format: `vX.Y.Z` (example: `v0.1.0`).
- Tags are created from `main` only.
- Do not reuse or move existing tags.

## Breaking Change Rules

- Any public API behavior/signature change must be recorded in:
  - `CHANGELOG.md`
  - `docs/MIGRATIONS.md`
- If config semantics or field meaning changes, include:
  - before/after YAML snippets
  - expected user action

