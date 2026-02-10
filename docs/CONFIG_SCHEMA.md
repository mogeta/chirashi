# Particle Config Schema (YAML)

This document defines the YAML contract for `github.com/mogeta/chirashi`.

## Top-level structure

```yaml
name: string
description: string

image:
  image_from: string
  image_id: int

emitter:
  x: float
  y: float

animation:
  duration:
    value: float
    range: { min: float, max: float } # optional
  position:
    type: "cartesian" | "polar"       # optional
    # cartesian fields (simple mode)
    start_x: { min: float, max: float } # optional
    end_x:   { min: float, max: float } # optional
    start_y: { min: float, max: float } # optional
    end_y:   { min: float, max: float } # optional
    # cartesian fields (sequence mode)
    x: PropertyConfig # optional
    y: PropertyConfig # optional
    # polar fields
    angle:    { min: float, max: float } # optional
    distance: { min: float, max: float } # optional
    easing: string
  alpha: PropertyConfig
  scale: PropertyConfig
  rotation: PropertyConfig
  color: # optional
    start_r: float
    start_g: float
    start_b: float
    end_r: float
    end_g: float
    end_b: float
    easing: string

spawn:
  interval: int
  particles_per_spawn: int
  max_particles: int
  is_loop: bool
  life_time: int # optional
```

`PropertyConfig`:

```yaml
# simple mode
start: float
end: float
easing: string

# sequence mode
type: "sequence"
steps:
  - from: float
    from_range: { min: float, max: float } # optional
    to: float
    to_range: { min: float, max: float }   # optional
    is_relative: bool                       # accepted, currently unused
    duration: float
    easing: string
```

## Validation rules (currently enforced)

Validation is performed by `ConfigLoader`:

- `name` is required.
- `spawn.max_particles` must be `> 0`.
- `spawn.particles_per_spawn` must be `> 0`.
- `spawn.interval` must be `> 0`.
- `animation.duration.value` must be `> 0`.

If validation fails, loading returns an error.

## Runtime defaults and fallback behavior

- Unknown or empty easing names fall back to `Linear`.
- `animation.position.type`:
  - `"polar"` uses `angle` + `distance`.
  - any other value (including empty) is treated as cartesian mode.
- If cartesian ranges are omitted, values default to `0`, so particles can stay at emitter position.
- If both `scale.start` and `scale.end` are `0`, runtime forces both to `1.0`.
- `animation.color` omitted means no color shift (white -> white).
- `spawn.life_time` is only meaningful when `spawn.is_loop: false`.

## Known non-enforced constraints

The loader currently does not reject:

- `range.min > range.max`
- non-positive step duration inside sequence steps
- missing sequence step fields when YAML zero-values are allowed by parser

Recommended practice:

- Always use `min <= max`.
- Keep all sequence `duration > 0`.
- Provide explicit easing for each animated property/step.

## Example: Looping effect (cartesian)

```yaml
name: "loop_smoke"
description: "looping smoke"
image: { image_from: "ef1", image_id: 1 }
emitter: { x: 0, y: 0 }
animation:
  duration: { value: 1.2 }
  position:
    type: "cartesian"
    end_x: { min: -40, max: 40 }
    end_y: { min: -140, max: -60 }
    easing: "OutQuad"
  alpha: { start: 0.7, end: 0.0, easing: "Linear" }
  scale: { start: 0.6, end: 1.2, easing: "OutSine" }
  rotation: { start: -0.3, end: 0.3, easing: "Linear" }
spawn:
  interval: 2
  particles_per_spawn: 8
  max_particles: 1200
  is_loop: true
```

## Example: One-shot effect

```yaml
name: "hit_burst"
description: "short one-shot burst"
image: { image_from: "ef1", image_id: 2 }
emitter: { x: 0, y: 0 }
animation:
  duration: { value: 0.35 }
  position:
    type: "polar"
    angle: { min: 0.0, max: 6.28318 }
    distance: { min: 24, max: 92 }
    easing: "OutQuad"
  alpha: { start: 1.0, end: 0.0, easing: "Linear" }
  scale: { start: 1.0, end: 0.2, easing: "OutQuad" }
  rotation: { start: 0.0, end: 3.14, easing: "Linear" }
spawn:
  interval: 1
  particles_per_spawn: 24
  max_particles: 256
  is_loop: false
  life_time: 30
```

## Example: Polar mode

```yaml
name: "radial_flame"
description: "full-circle flame ring"
image: { image_from: "ef1", image_id: 16 }
emitter: { x: 0, y: 0 }
animation:
  duration:
    value: 1.0
    range: { min: 0.8, max: 1.2 }
  position:
    type: "polar"
    angle: { min: 0.0, max: 6.28318 }
    distance: { min: 50, max: 150 }
    easing: "OutCirc"
  alpha: { start: 1.0, end: 0.0, easing: "Linear" }
  scale: { start: 0.5, end: 1.5, easing: "OutBack" }
  rotation: { start: 0.0, end: 6.28318, easing: "Linear" }
  color:
    start_r: 1.0
    start_g: 1.0
    start_b: 0.8
    end_r: 1.0
    end_g: 0.2
    end_b: 0.0
    easing: "Linear"
spawn:
  interval: 1
  particles_per_spawn: 50
  max_particles: 5000
  is_loop: true
```

## Example: Sequence-based animation

```yaml
name: "sequence_trail"
description: "multi-step movement and scale"
image: { image_from: "ef1", image_id: 7 }
emitter: { x: 0, y: 0 }
animation:
  duration: { value: 1.5 }
  position:
    type: "cartesian"
    x:
      type: "sequence"
      steps:
        - from: 0
          to: 20
          duration: 0.3
          easing: "OutQuad"
        - from: 20
          to: -10
          duration: 0.4
          easing: "InOutSine"
    y:
      type: "sequence"
      steps:
        - from: 0
          to: -80
          duration: 0.6
          easing: "OutCubic"
        - from: -80
          to: -120
          duration: 0.9
          easing: "OutQuad"
    easing: "Linear"
  alpha:
    type: "sequence"
    steps:
      - from: 0.0
        to: 1.0
        duration: 0.2
        easing: "OutQuad"
      - from: 1.0
        to: 0.0
        duration: 1.3
        easing: "InQuad"
  scale:
    type: "sequence"
    steps:
      - from: 0.6
        to: 1.2
        duration: 0.5
        easing: "OutBack"
      - from: 1.2
        to: 0.3
        duration: 1.0
        easing: "InSine"
  rotation: { start: 0.0, end: 2.0, easing: "Linear" }
spawn:
  interval: 2
  particles_per_spawn: 10
  max_particles: 600
  is_loop: true
```

## Config compatibility policy

- Backward-compatible additions are preferred (new optional fields).
- Existing field names/meaning should not change without migration notes.
- If a breaking config change is introduced, bump release version and provide conversion guidance.
