package chirashi

import (
	"strings"
	"testing"
)

func validParticleConfigForTest() *ParticleConfig {
	return &ParticleConfig{
		Name:        "test",
		Description: "test config",
		Animation: AnimationConfig{
			Duration: DurationConfig{Value: 1.0},
			Position: PositionConfig{Easing: "Linear"},
			Alpha:    PropertyConfig{Start: 1, End: 0, Easing: "Linear"},
			Scale:    PropertyConfig{Start: 1, End: 1, Easing: "Linear"},
			Rotation: PropertyConfig{Start: 0, End: 0, Easing: "Linear"},
		},
		Spawn: SpawnConfig{
			Interval:          1,
			ParticlesPerSpawn: 1,
			MaxParticles:      16,
			IsLoop:            true,
		},
	}
}

func TestValidateConfigAcceptsValidConfig(t *testing.T) {
	loader := NewConfigLoader()
	cfg := validParticleConfigForTest()

	if err := loader.validateConfig(cfg); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestValidateConfigAcceptsDurationRangeWithoutValue(t *testing.T) {
	loader := NewConfigLoader()
	cfg := validParticleConfigForTest()
	// value=0 is fine when range provides valid min/max
	cfg.Animation.Duration.Value = 0
	cfg.Animation.Duration.Range = &RangeFloat{Min: 0.5, Max: 1.5}

	if err := loader.validateConfig(cfg); err != nil {
		t.Fatalf("expected range-only duration to be valid, got: %v", err)
	}
}

func TestValidateConfigRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*ParticleConfig)
		wantErr string
	}{
		{
			name: "missing name",
			mutate: func(c *ParticleConfig) {
				c.Name = ""
			},
			wantErr: "name is required",
		},
		{
			name: "invalid max_particles",
			mutate: func(c *ParticleConfig) {
				c.Spawn.MaxParticles = 0
			},
			wantErr: "max_particles",
		},
		{
			name: "invalid particles_per_spawn",
			mutate: func(c *ParticleConfig) {
				c.Spawn.ParticlesPerSpawn = 0
			},
			wantErr: "particles_per_spawn",
		},
		{
			name: "invalid interval",
			mutate: func(c *ParticleConfig) {
				c.Spawn.Interval = 0
			},
			wantErr: "interval",
		},
		{
			name: "invalid duration",
			mutate: func(c *ParticleConfig) {
				c.Animation.Duration.Value = 0
			},
			wantErr: "animation.duration.value",
		},
		{
			name: "duration range with non-positive min",
			mutate: func(c *ParticleConfig) {
				c.Animation.Duration.Value = 0
				c.Animation.Duration.Range = &RangeFloat{Min: 0, Max: 1.0}
			},
			wantErr: "animation.duration.range.min",
		},
		{
			name: "invalid emitter shape type",
			mutate: func(c *ParticleConfig) {
				c.Emitter.Shape.Type = "mesh"
			},
			wantErr: "emitter.shape.type",
		},
		{
			name: "invalid emitter radius range",
			mutate: func(c *ParticleConfig) {
				c.Emitter.Shape.Type = "circle"
				c.Emitter.Shape.Radius = &RangeFloat{Min: 10, Max: 5}
			},
			wantErr: "emitter.shape.radius.min",
		},
	}

	loader := NewConfigLoader()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validParticleConfigForTest()
			tt.mutate(cfg)

			err := loader.validateConfig(cfg)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}
