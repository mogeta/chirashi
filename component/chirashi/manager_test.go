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
			},
		},
	}

	dst := copyConfig(src)
	dst.Animation.Position.StartX.Min = 99
	dst.Animation.Position.Angle.Max = 99

	if src.Animation.Position.StartX.Min == 99 {
		t.Error("Position.StartX: src was modified by dst change")
	}
	if src.Animation.Position.Angle.Max == 99 {
		t.Error("Position.Angle: src was modified by dst change")
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

// minConfig returns a minimal valid ParticleConfig for use in ParticleManager tests.
func minConfig(name string) *ParticleConfig {
	return &ParticleConfig{
		Name: name,
		Animation: AnimationConfig{
			Duration: DurationConfig{Value: 1.0},
			Alpha:    PropertyConfig{Start: 1, End: 0},
			Scale:    PropertyConfig{Start: 1, End: 1},
		},
		Spawn: SpawnConfig{
			Interval:          1,
			ParticlesPerSpawn: 1,
			MaxParticles:      10,
			IsLoop:            true,
		},
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
