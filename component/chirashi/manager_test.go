package chirashi

import (
	"testing"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
)

func TestCopyConfigDeepCopiesPropertySteps(t *testing.T) {
	step := StepConfig{
		From:      0,
		To:        1,
		Duration:  0.5,
		Easing:    "linear",
		FromRange: &RangeFloat{Min: 0, Max: 0.1},
		ToRange:   &RangeFloat{Min: 0.9, Max: 1.0},
	}
	src := &ParticleConfig{
		Animation: AnimationConfig{
			Alpha: PropertyConfig{
				Type:  "sequence",
				Steps: []StepConfig{step},
			},
			Scale: PropertyConfig{
				Type:  "sequence",
				Steps: []StepConfig{step},
			},
			Rotation: PropertyConfig{
				Type:  "sequence",
				Steps: []StepConfig{step},
			},
		},
	}

	dst := copyConfig(src)

	// Mutate dst steps and verify src is unaffected
	dst.Animation.Alpha.Steps[0].From = 99
	dst.Animation.Alpha.Steps[0].FromRange.Min = 99
	if src.Animation.Alpha.Steps[0].From == 99 {
		t.Error("Alpha.Steps[0].From: src was modified by dst change")
	}
	if src.Animation.Alpha.Steps[0].FromRange.Min == 99 {
		t.Error("Alpha.Steps[0].FromRange: src was modified by dst change")
	}

	dst.Animation.Scale.Steps[0].To = 99
	if src.Animation.Scale.Steps[0].To == 99 {
		t.Error("Scale.Steps[0].To: src was modified by dst change")
	}

	dst.Animation.Rotation.Steps[0].Duration = 99
	if src.Animation.Rotation.Steps[0].Duration == 99 {
		t.Error("Rotation.Steps[0].Duration: src was modified by dst change")
	}
}

func TestCopyConfigDeepCopiesPositionSequences(t *testing.T) {
	step := StepConfig{From: 0, To: 100, Duration: 1.0}
	prop := PropertyConfig{Type: "sequence", Steps: []StepConfig{step}}
	src := &ParticleConfig{
		Animation: AnimationConfig{
			Position: PositionConfig{
				X: &prop,
				Y: &prop,
			},
		},
	}

	dst := copyConfig(src)

	dst.Animation.Position.X.Steps[0].From = 99
	if src.Animation.Position.X.Steps[0].From == 99 {
		t.Error("Position.X.Steps: src was modified by dst change")
	}

	dst.Animation.Position.Y.Steps[0].To = 99
	if src.Animation.Position.Y.Steps[0].To == 99 {
		t.Error("Position.Y.Steps: src was modified by dst change")
	}
}

func TestCopyConfigDeepCopiesPositionRanges(t *testing.T) {
	src := &ParticleConfig{
		Animation: AnimationConfig{
			Position: PositionConfig{
				StartX:   &RangeFloat{Min: 0, Max: 10},
				EndX:     &RangeFloat{Min: 100, Max: 200},
				Angle:    &RangeFloat{Min: 0, Max: 6.28},
				Distance: &RangeFloat{Min: 10, Max: 50},
				Flow: &FlowConfig{
					Type:     "curl",
					Strength: &RangeFloat{Min: 4, Max: 12},
					Scale:    180,
				},
			},
		},
	}

	dst := copyConfig(src)
	dst.Animation.Position.StartX.Min = 99
	dst.Animation.Position.Angle.Max = 99
	dst.Animation.Position.Flow.Strength.Min = 99

	if src.Animation.Position.StartX.Min == 99 {
		t.Error("Position.StartX: src was modified by dst change")
	}
	if src.Animation.Position.Angle.Max == 99 {
		t.Error("Position.Angle: src was modified by dst change")
	}
	if src.Animation.Position.Flow.Strength.Min == 99 {
		t.Error("Position.Flow.Strength: src was modified by dst change")
	}
}

func TestCopyConfigDeepCopiesDurationRange(t *testing.T) {
	src := &ParticleConfig{
		Animation: AnimationConfig{
			Duration: DurationConfig{
				Value: 1.0,
				Range: &RangeFloat{Min: 0.5, Max: 1.5},
			},
		},
	}

	dst := copyConfig(src)
	dst.Animation.Duration.Range.Min = 99

	if src.Animation.Duration.Range.Min == 99 {
		t.Error("Duration.Range: src was modified by dst change")
	}
}

func TestCopyConfigDeepCopiesColor(t *testing.T) {
	src := &ParticleConfig{
		Animation: AnimationConfig{
			Color: &ColorConfig{StartR: 1.0, EndR: 0.5},
		},
	}

	dst := copyConfig(src)
	dst.Animation.Color.StartR = 99

	if src.Animation.Color.StartR == 99 {
		t.Error("Color: src was modified by dst change")
	}
}

func TestCopyConfigDeepCopiesEmitterShape(t *testing.T) {
	src := &ParticleConfig{
		Emitter: EmitterConfig{
			Shape: EmitterShapeConfig{
				Type:   "circle",
				Radius: &RangeFloat{Min: 10, Max: 20},
			},
		},
	}

	dst := copyConfig(src)
	dst.Emitter.Shape.Type = "line"
	dst.Emitter.Shape.Radius.Min = 99

	if src.Emitter.Shape.Type != "circle" {
		t.Error("Emitter.Shape.Type: src was modified by dst change")
	}
	if src.Emitter.Shape.Radius.Min == 99 {
		t.Error("Emitter.Shape.Radius: src was modified by dst change")
	}
}

func TestCopyConfigDeepCopiesTrail(t *testing.T) {
	src := &ParticleConfig{
		Trail: &TrailConfig{
			Enabled:          true,
			Mode:             "emitter",
			Space:            "world",
			MaxPoints:        8,
			MinPointDistance: 4,
			MaxPointAge:      0.5,
			Width:            TrailScalarConfig{Start: 16, End: 0, Easing: "OutQuad"},
			Alpha:            TrailScalarConfig{Start: 0.8, End: 0, Easing: "Linear"},
			Color:            &ColorConfig{StartR: 1, StartG: 0.8, StartB: 0.3, EndR: 1, EndG: 0.1, EndB: 0.0, Easing: "OutQuad"},
		},
	}

	dst := copyConfig(src)
	dst.Trail.Space = "local"
	dst.Trail.Color.StartR = 99

	if src.Trail.Space != "world" {
		t.Error("Trail.Space: src was modified by dst change")
	}
	if src.Trail.Color.StartR == 99 {
		t.Error("Trail.Color: src was modified by dst change")
	}
}

func TestApplyConfigLiveUpdatesActiveParticles(t *testing.T) {
	world := donburi.NewWorld()
	entity := world.Create(Component)
	entry := world.Entry(entity)

	donburi.SetValue(entry, Component, SystemData{
		CurrentTime:   5,
		EmitterX:      100,
		EmitterY:      200,
		ActiveIndices: []int{0},
		ActiveCount:   1,
		ParticlePool: []Instance{
			{
				Active:         true,
				SpawnTime:      2,
				Duration:       6,
				StartX:         110,
				EndX:           140,
				StartY:         210,
				EndY:           260,
				ControlX:       120,
				ControlY:       180,
				StartAlpha:     1,
				EndAlpha:       0,
				StartScale:     1,
				EndScale:       2,
				StartRotation:  0,
				EndRotation:    1,
				StartR:         1,
				StartG:         1,
				StartB:         1,
				EndR:           1,
				EndG:           1,
				EndB:           1,
				PositionEasing: EasingLinear,
				AlphaEasing:    EasingLinear,
				ScaleEasing:    EasingLinear,
				RotationEasing: EasingLinear,
				ColorEasing:    EasingLinear,
			},
		},
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 6},
			Position: PositionParams{Easing: EasingLinear},
			Appearance: AppearanceParams{
				StartAlpha:     1,
				EndAlpha:       0,
				AlphaEasing:    EasingLinear,
				StartScale:     1,
				EndScale:       2,
				ScaleEasing:    EasingLinear,
				StartRotation:  0,
				EndRotation:    1,
				RotationEasing: EasingLinear,
			},
			Color: ColorParams{
				Enabled: true,
				StartR:  1, StartG: 1, StartB: 1,
				EndR: 1, EndG: 1, EndB: 1,
				Easing: EasingLinear,
			},
		},
	})

	cfg := &ParticleConfig{
		Emitter: EmitterConfig{X: 20, Y: -10},
		Animation: AnimationConfig{
			Duration: DurationConfig{Value: 3},
			Position: PositionConfig{
				Easing: "OutSine",
				Flow:   &FlowConfig{Type: "curl", Strength: &RangeFloat{Min: 6, Max: 10}},
			},
			Alpha:    PropertyConfig{Start: 0.4, End: 0.1, Easing: "OutQuad"},
			Scale:    PropertyConfig{Start: 2.5, End: 0.5, Easing: "InQuad"},
			Rotation: PropertyConfig{Start: 0.3, End: 2.1, Easing: "InOutSine"},
			Color:    &ColorConfig{StartR: 0.1, StartG: 0.2, StartB: 0.3, EndR: 0.7, EndG: 0.8, EndB: 0.9, Easing: "OutCubic"},
		},
		Spawn: SpawnConfig{Interval: 4, ParticlesPerSpawn: 9, IsLoop: true},
	}

	ApplyConfigLive(world, entity, cfg, 1000, 500)

	data := Component.Get(entry)
	p := data.ParticlePool[0]

	if data.EmitterX != 1020 || data.EmitterY != 490 {
		t.Fatalf("unexpected emitter position: (%v,%v)", data.EmitterX, data.EmitterY)
	}
	if p.StartX != 1030 || p.EndX != 1060 || p.StartY != 500 || p.EndY != 550 {
		t.Fatalf("expected active particle to shift with emitter, got start=(%v,%v) end=(%v,%v)", p.StartX, p.StartY, p.EndX, p.EndY)
	}
	if p.StartAlpha != 0.4 || p.EndAlpha != 0.1 || p.StartScale != 2.5 || p.EndScale != 0.5 {
		t.Fatalf("appearance was not updated: startAlpha=%v endAlpha=%v startScale=%v endScale=%v", p.StartAlpha, p.EndAlpha, p.StartScale, p.EndScale)
	}
	if p.StartR != 0.1 || p.EndB != 0.9 {
		t.Fatalf("color was not updated: startR=%v endB=%v", p.StartR, p.EndB)
	}
	if p.Duration != 3 {
		t.Fatalf("duration was not updated: %v", p.Duration)
	}
	if !p.HasFlow || p.FlowGain != 8 {
		t.Fatalf("flow was not enabled with midpoint strength, got hasFlow=%v gain=%v", p.HasFlow, p.FlowGain)
	}
	if data.SpawnInterval != 4 || data.ParticlesPerSpawn != 9 {
		t.Fatalf("spawn settings were not updated: interval=%d particles=%d", data.SpawnInterval, data.ParticlesPerSpawn)
	}
}

func TestApplyConfigLiveDoesNotShiftActiveParticlesInWorldSpace(t *testing.T) {
	world := donburi.NewWorld()
	entity := world.Create(Component)
	entry := world.Entry(entity)

	donburi.SetValue(entry, Component, SystemData{
		CurrentTime:       2,
		EmitterX:          100,
		EmitterY:          200,
		EmitterLocalSpace: true,
		ActiveIndices:     []int{0},
		ActiveCount:       1,
		ParticlePool: []Instance{
			{
				Active:    true,
				SpawnTime: 0,
				Duration:  4,
				StartX:    110,
				EndX:      140,
				StartY:    210,
				EndY:      240,
			},
		},
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 4},
			Position: PositionParams{Easing: EasingLinear},
			Appearance: AppearanceParams{
				StartAlpha: 1, EndAlpha: 0, AlphaEasing: EasingLinear,
				StartScale: 1, EndScale: 1, ScaleEasing: EasingLinear,
				StartRotation: 0, EndRotation: 0, RotationEasing: EasingLinear,
			},
		},
	})

	cfg := &ParticleConfig{
		Emitter: EmitterConfig{X: 50, Y: 25, Space: "world"},
		Animation: AnimationConfig{
			Duration: DurationConfig{Value: 4},
			Position: PositionConfig{Easing: "Linear"},
			Alpha:    PropertyConfig{Start: 1, End: 0, Easing: "Linear"},
			Scale:    PropertyConfig{Start: 1, End: 1, Easing: "Linear"},
			Rotation: PropertyConfig{Start: 0, End: 0, Easing: "Linear"},
		},
		Spawn: SpawnConfig{Interval: 1, ParticlesPerSpawn: 1, IsLoop: true},
	}

	ApplyConfigLive(world, entity, cfg, 1000, 500)

	data := Component.Get(entry)
	p := data.ParticlePool[0]
	if data.EmitterLocalSpace {
		t.Fatal("expected world-space emitter behavior")
	}
	if p.StartX != 110 || p.EndX != 140 || p.StartY != 210 || p.EndY != 240 {
		t.Fatalf("expected active particle to keep world position, got start=(%v,%v) end=(%v,%v)", p.StartX, p.StartY, p.EndX, p.EndY)
	}
}

func TestParticleManagerPreloadFromBytesAndSpawn(t *testing.T) {
	// Use YAML bytes equivalent to minConfig
	yaml := []byte(`
name: test
animation:
  duration:
    value: 1.0
  alpha:
    start: 1.0
    end: 0.0
  scale:
    start: 1.0
    end: 1.0
spawn:
  interval: 1
  particles_per_spawn: 1
  max_particles: 10
  is_loop: true
`)

	m := NewParticleManager(nil, nil)
	if err := m.PreloadFromBytes("test", yaml); err != nil {
		t.Fatalf("PreloadFromBytes failed: %v", err)
	}

	world := donburi.NewWorld()
	if err := m.SpawnOneShot(world, "test", 0, 0, 60); err != nil {
		t.Fatalf("SpawnOneShot failed: %v", err)
	}

	count := 0
	q := donburi.NewQuery(filter.Contains(Component))
	q.Each(world, func(e *donburi.Entry) { count++ })
	if count != 1 {
		t.Fatalf("expected 1 particle entity, got %d", count)
	}
}

func TestParticleManagerSpawnOneShotNotFound(t *testing.T) {
	m := NewParticleManager(nil, nil)
	world := donburi.NewWorld()

	err := m.SpawnOneShot(world, "nonexistent", 0, 0, 60)
	if err == nil {
		t.Fatal("expected error for unknown config name, got nil")
	}
}

func TestParticleManagerSpawnLoopNotFound(t *testing.T) {
	m := NewParticleManager(nil, nil)
	world := donburi.NewWorld()

	_, err := m.SpawnLoop(world, "nonexistent", 0, 0)
	if err == nil {
		t.Fatal("expected error for unknown config name, got nil")
	}
}

func TestParticleManagerCopiesConfigOnSpawn(t *testing.T) {
	yaml := []byte(`
name: copytest
animation:
  duration:
    value: 1.0
  alpha:
    start: 1.0
    end: 0.0
  scale:
    start: 1.0
    end: 1.0
spawn:
  interval: 1
  particles_per_spawn: 1
  max_particles: 5
  is_loop: false
`)

	m := NewParticleManager(nil, nil)
	if err := m.PreloadFromBytes("copytest", yaml); err != nil {
		t.Fatalf("PreloadFromBytes failed: %v", err)
	}

	world := donburi.NewWorld()
	// Spawn twice to ensure each gets an independent copy
	if err := m.SpawnOneShot(world, "copytest", 0, 0, 10); err != nil {
		t.Fatalf("first SpawnOneShot failed: %v", err)
	}
	if err := m.SpawnOneShot(world, "copytest", 100, 100, 20); err != nil {
		t.Fatalf("second SpawnOneShot failed: %v", err)
	}

	count := 0
	q := donburi.NewQuery(filter.Contains(Component))
	q.Each(world, func(e *donburi.Entry) { count++ })
	if count != 2 {
		t.Fatalf("expected 2 particle entities, got %d", count)
	}
}

func TestAttractorSpawnSetsControlPoint(t *testing.T) {
	yaml := []byte(`
name: attractor_test
animation:
  duration:
    value: 1.0
  position:
    type: "attractor"
    control_x: { min: -100, max: 100 }
    control_y: { min: -200, max: -50 }
    easing: "inquad"
  alpha:
    start: 1.0
    end: 0.0
  scale:
    start: 1.0
    end: 1.0
spawn:
  interval: 1
  particles_per_spawn: 5
  max_particles: 20
  is_loop: true
`)

	m := NewParticleManager(nil, nil)
	if err := m.PreloadFromBytes("attractor_test", yaml); err != nil {
		t.Fatalf("PreloadFromBytes: %v", err)
	}

	world := donburi.NewWorld()
	entity, err := m.SpawnLoop(world, "attractor_test", 100, 200)
	if err != nil {
		t.Fatalf("SpawnLoop: %v", err)
	}

	// Set attractor target
	SetAttractor(world, entity, 500, 50)

	entry := world.Entry(entity)
	data := Component.Get(entry)

	if data.AttractorX != 500 || data.AttractorY != 50 {
		t.Errorf("attractor not set: got (%v, %v), want (500, 50)", data.AttractorX, data.AttractorY)
	}
	if !data.AnimParams.Position.UseAttractor {
		t.Error("UseAttractor should be true for attractor position type")
	}
}

func TestAttractorParticleHasControlPoint(t *testing.T) {
	// Build system data directly to test spawn sets HasAttractor and ControlX/Y
	data := &SystemData{
		ParticlePool:      make([]Instance, 5),
		ActiveIndices:     make([]int, 0, 5),
		FreeIndices:       []int{4, 3, 2, 1, 0},
		SpawnInterval:     1,
		ParticlesPerSpawn: 5,
		MaxParticles:      5,
		IsLoop:            true,
		EmitterX:          100,
		EmitterY:          200,
		AttractorX:        600,
		AttractorY:        100,
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1.0},
			Position: PositionParams{
				UseAttractor: true,
				ControlXMin:  -50, ControlXMax: 50,
				ControlYMin: -100, ControlYMax: -10,
				Easing: EasingLinear,
			},
			Appearance: AppearanceParams{StartAlpha: 1, EndAlpha: 0, StartScale: 1, EndScale: 1},
			Color:      ColorParams{StartR: 1, StartG: 1, StartB: 1, EndR: 1, EndG: 1, EndB: 1},
		},
	}

	sys := &System{cnt: 0}
	sys.spawn(data)

	if data.ActiveCount == 0 {
		t.Fatal("no particles spawned")
	}

	for _, idx := range data.ActiveIndices {
		p := &data.ParticlePool[idx]
		if !p.HasAttractor {
			t.Errorf("particle[%d]: HasAttractor should be true", idx)
		}
		if p.StartX != 100 || p.StartY != 200 {
			t.Errorf("particle[%d]: start should be emitter pos, got (%v, %v)", idx, p.StartX, p.StartY)
		}
		// Control point must be within emitter + configured range
		if p.ControlX < 100-50 || p.ControlX > 100+50 {
			t.Errorf("particle[%d]: ControlX %v outside emitter±50", idx, p.ControlX)
		}
		if p.ControlY < 200-100 || p.ControlY > 200-10 {
			t.Errorf("particle[%d]: ControlY %v outside expected range", idx, p.ControlY)
		}
	}
}

func TestSetEmitterPositionShiftsLocalParticlesAndTrail(t *testing.T) {
	world := donburi.NewWorld()
	entity := world.Create(Component)
	entry := world.Entry(entity)

	donburi.SetValue(entry, Component, SystemData{
		EmitterX:          10,
		EmitterY:          20,
		EmitterLocalSpace: true,
		ActiveIndices:     []int{0},
		ActiveCount:       1,
		ParticlePool: []Instance{
			{
				Active:   true,
				StartX:   12,
				EndX:     18,
				StartY:   24,
				EndY:     30,
				ControlX: 14,
				ControlY: 16,
			},
		},
		Trail: TrailData{
			Enabled:    true,
			Mode:       "emitter",
			LocalSpace: true,
			Points: []TrailPoint{
				{X: 8, Y: 18},
				{X: 10, Y: 20},
			},
		},
	})

	SetEmitterPosition(world, entity, 25, 35)

	data := Component.Get(entry)
	p := data.ParticlePool[0]
	if p.StartX != 27 || p.EndX != 33 || p.StartY != 39 || p.EndY != 45 {
		t.Fatalf("expected particle path to shift with emitter, got start=(%v,%v) end=(%v,%v)", p.StartX, p.StartY, p.EndX, p.EndY)
	}
	if p.ControlX != 29 || p.ControlY != 31 {
		t.Fatalf("expected control point to shift with emitter, got (%v,%v)", p.ControlX, p.ControlY)
	}
	if data.Trail.Points[0].X != 23 || data.Trail.Points[0].Y != 33 {
		t.Fatalf("expected local-space trail point to shift, got (%v,%v)", data.Trail.Points[0].X, data.Trail.Points[0].Y)
	}
}

func TestSetEmitterPositionShiftsLocalParticleTrails(t *testing.T) {
	world := donburi.NewWorld()
	entity := world.Create(Component)
	entry := world.Entry(entity)

	donburi.SetValue(entry, Component, SystemData{
		EmitterX:          10,
		EmitterY:          20,
		EmitterLocalSpace: true,
		Trail: TrailData{
			Enabled:    true,
			Mode:       "particle",
			LocalSpace: true,
		},
		ParticlePool: []Instance{
			{TrailPoints: []TrailPoint{{X: 10, Y: 20}, {X: 14, Y: 24}}},
		},
	})

	SetEmitterPosition(world, entity, 20, 35)

	data := Component.Get(entry)
	if data.ParticlePool[0].TrailPoints[0].X != 20 || data.ParticlePool[0].TrailPoints[0].Y != 35 {
		t.Fatalf("expected first particle trail point to shift with local emitter, got (%v,%v)", data.ParticlePool[0].TrailPoints[0].X, data.ParticlePool[0].TrailPoints[0].Y)
	}
	if data.ParticlePool[0].TrailPoints[1].X != 24 || data.ParticlePool[0].TrailPoints[1].Y != 39 {
		t.Fatalf("expected second particle trail point to shift with local emitter, got (%v,%v)", data.ParticlePool[0].TrailPoints[1].X, data.ParticlePool[0].TrailPoints[1].Y)
	}
}
