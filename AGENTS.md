# AGENTS.md

This file gives repository-local instructions to coding agents working on `chirashi`.

## Goal

Use this repository to create, tune, and ship reusable particle effects for Ebitengine.
Agents should optimize for:

- Creative visual variety
- Practical reuse in games
- Low runtime overhead
- Clean YAML configs and understandable defaults

## Primary Workflow

When the user asks for a new effect, prefer this order:

1. Inspect existing particle samples in `assets/particles/`
2. Reuse the current system before changing engine code
3. Create or edit YAML configs first
4. Only change runtime/editor code when the requested effect cannot be expressed cleanly
5. Verify with `go test ./...`

Do not jump to engine changes if a new YAML or minor editor improvement is enough.

## Creative Effect Authoring Rules

- Prefer building effects from strong silhouettes:
  - ring
  - cone
  - line burst
  - box edge
  - upward drift
  - inward attractor pull
- Use `emitter.shape` aggressively to differentiate effects before changing colors or easing.
- Use `polar` for bursts, sprays, thrust, explosions, muzzle flashes.
- Use `cartesian` for smoke, rain, ambient drift, UI sparkles, layered motion.
- Use `attractor` for pickups, UI trails, score effects, homing sparkles.
- Use `sequence` only when it materially improves motion or opacity; do not overcomplicate simple effects.

## Performance Rules

This project is a performance-sensitive particle library. Agents must protect runtime efficiency.

- Prefer spawn-time randomization over per-frame logic.
- Do not add per-particle allocations inside update or draw paths.
- Do not add linear scans over the full particle pool when active-index iteration is sufficient.
- Prefer extending sampling at spawn time over adding draw-time branches.
- Avoid effect definitions that require extreme `max_particles` unless the visual payoff is clear.
- If introducing a new runtime feature, explain its update-time and draw-time cost.

## YAML Authoring Guidelines

Every new effect YAML should:

- Have a game-usable name, not a temporary experiment name
- Include a concise description
- Use stable, readable numeric ranges
- Avoid magic values when a simpler range works
- Be visually distinctive from existing samples

Recommended effect categories:

- Combat: hit sparks, muzzle flashes, slash trails, charge effects
- Environment: smoke, rain, embers, fog bursts
- UI/feedback: pickups, heal sparkles, reward bursts, portal highlights
- Magic/Sci-fi: barriers, runes, shockwaves, plasma arcs

When adding new sample files, prefer one of these intents:

- “common gameplay effect”
- “showcase of a specific feature”
- “editor regression sample”

## Editor Usage Guidance

The editor is a tuning tool, not just a demo.

- Default preview center is the screen center in the editor.
- In `attractor` mode, clicking the screen sets the attractor goal.
- If an effect depends on `attractor`, describe that in the YAML `description`.
- If a parameter is hard to tune manually, improve the editor only when the improvement is generally useful.

Good editor improvements:

- Better mode selection
- Clearer one-line controls
- Feature-specific affordances used by multiple effects

Avoid editor changes that are only useful for a single one-off sample.

## When Engine Changes Are Justified

Change runtime/editor code only if one of these is true:

- The target effect is impossible with current YAML features
- Multiple effect categories would benefit from the same capability
- The editor cannot reasonably expose an already-supported feature
- A bug or ambiguous default blocks normal use

When adding engine features:

- Keep config backward compatible where possible
- Add tests for sampling/validation behavior
- Update `README.md` and `docs/CONFIG_SCHEMA.md` if the feature is user-facing

## Release Quality Bar

Before considering an effect or feature “done”, check:

- YAML loads without schema surprises
- The effect is distinct from existing samples
- Naming is production-usable
- The editor can still load and tweak it
- `go test ./...` passes
- Docs are updated if public behavior changed

## Files To Know

- `assets/particles/*.yaml`: effect definitions
- `internal/editor/particle_editor_scene.go`: live editor behavior
- `component/chirashi/config.go`: YAML config types
- `component/chirashi/system.go`: spawn/update/draw runtime
- `component/chirashi/factory.go`: config normalization and system creation
- `docs/CONFIG_SCHEMA.md`: public config contract
- `README.md`: public-facing overview

## Default Agent Behavior

If the user says “make a cool effect”, do not ask for excessive clarification.
Choose a strong game-relevant direction, implement it in YAML, and explain what it is intended for.

If the user asks for “Unity-level” or “more expressive” effects, first exhaust:

- shape
- attractor
- sequence
- layered sample presets

before proposing expensive runtime systems.
