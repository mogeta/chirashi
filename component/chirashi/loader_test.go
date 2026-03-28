package chirashi

import (
	"path/filepath"
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
			name: "invalid emitter space",
			mutate: func(c *ParticleConfig) {
				c.Emitter.Space = "screen"
			},
			wantErr: "emitter.space",
		},
		{
			name: "invalid emitter vector type",
			mutate: func(c *ParticleConfig) {
				c.Emitter.Vector = &EmitterVectorConfig{Type: "glyphs"}
			},
			wantErr: "emitter.vector.type",
		},
		{
			name: "invalid polyline placement",
			mutate: func(c *ParticleConfig) {
				c.Emitter.Vector = &EmitterVectorConfig{
					Type:      "polyline",
					Placement: "fill",
					Polyline: &EmitterVectorPolylineConfig{
						Points: []EmitterVectorPoint{{X: -10, Y: 0}, {X: 10, Y: 0}},
					},
				}
			},
			wantErr: "emitter.vector.placement must be surface for polyline",
		},
		{
			name: "invalid emitter vector placement",
			mutate: func(c *ParticleConfig) {
				c.Emitter.Vector = &EmitterVectorConfig{
					Type:      "rect",
					Placement: "inside",
					Rect:      &EmitterVectorRectConfig{Width: 10, Height: 10},
				}
			},
			wantErr: "emitter.vector.placement",
		},
		{
			name: "missing emitter vector rect",
			mutate: func(c *ParticleConfig) {
				c.Emitter.Vector = &EmitterVectorConfig{Type: "rect"}
			},
			wantErr: "emitter.vector.rect is required",
		},
		{
			name: "invalid emitter vector rect width",
			mutate: func(c *ParticleConfig) {
				c.Emitter.Vector = &EmitterVectorConfig{Type: "rect", Rect: &EmitterVectorRectConfig{Width: 0, Height: 10}}
			},
			wantErr: "emitter.vector.rect.width",
		},
		{
			name: "missing emitter vector polyline",
			mutate: func(c *ParticleConfig) {
				c.Emitter.Vector = &EmitterVectorConfig{Type: "polyline", Placement: "surface"}
			},
			wantErr: "emitter.vector.polyline is required",
		},
		{
			name: "polyline requires at least two points",
			mutate: func(c *ParticleConfig) {
				c.Emitter.Vector = &EmitterVectorConfig{
					Type:      "polyline",
					Placement: "surface",
					Polyline:  &EmitterVectorPolylineConfig{Points: []EmitterVectorPoint{{X: 0, Y: 0}}},
				}
			},
			wantErr: "emitter.vector.polyline.points",
		},
		{
			name: "invalid polyline interpolation",
			mutate: func(c *ParticleConfig) {
				c.Emitter.Vector = &EmitterVectorConfig{
					Type:      "polyline",
					Placement: "surface",
					Polyline: &EmitterVectorPolylineConfig{
						Interpolation: "cubic",
						Points:        []EmitterVectorPoint{{X: -10, Y: 0}, {X: 10, Y: 0}},
					},
				}
			},
			wantErr: "emitter.vector.polyline.interpolation",
		},
		{
			name: "quadratic polyline requires anchor control anchor pattern",
			mutate: func(c *ParticleConfig) {
				c.Emitter.Vector = &EmitterVectorConfig{
					Type:      "polyline",
					Placement: "surface",
					Polyline: &EmitterVectorPolylineConfig{
						Interpolation: "quadratic",
						Points:        []EmitterVectorPoint{{X: -10, Y: 0}, {X: 0, Y: 10}},
					},
				}
			},
			wantErr: "anchor/control/anchor",
		},
		{
			name: "invalid emitter radius range",
			mutate: func(c *ParticleConfig) {
				c.Emitter.Shape.Type = "circle"
				c.Emitter.Shape.Radius = &RangeFloat{Min: 10, Max: 5}
			},
			wantErr: "emitter.shape.radius.min",
		},
		{
			name: "invalid flow type",
			mutate: func(c *ParticleConfig) {
				c.Animation.Position.Flow = &FlowConfig{Type: "swirl"}
			},
			wantErr: "animation.position.flow.type",
		},
		{
			name: "invalid flow octaves",
			mutate: func(c *ParticleConfig) {
				c.Animation.Position.Flow = &FlowConfig{Type: "curl", Octaves: 4}
			},
			wantErr: "animation.position.flow.octaves",
		},
		{
			name: "invalid flow drag",
			mutate: func(c *ParticleConfig) {
				c.Animation.Position.Flow = &FlowConfig{Type: "curl", Drag: 1.5}
			},
			wantErr: "animation.position.flow.drag",
		},
		{
			name: "invalid trail max points",
			mutate: func(c *ParticleConfig) {
				c.Trail = &TrailConfig{Enabled: true, MaxPoints: 1}
			},
			wantErr: "trail.max_points",
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

func TestLoadReentryPlasmaWakeSample(t *testing.T) {
	loader := NewConfigLoader()
	path := filepath.Join("..", "..", "assets", "particles", "reentry_plasma_wake.yaml")

	cfg, err := loader.LoadConfig(path)
	if err != nil {
		t.Fatalf("expected reentry_plasma_wake sample to load, got: %v", err)
	}
	if cfg.Emitter.Space != EmitterSpaceWorld {
		t.Fatalf("expected reentry_plasma_wake to use world-space emitter, got: %v", cfg.Emitter.Space)
	}
	if cfg.Animation.Position.Flow == nil || cfg.Animation.Position.Flow.Type != "curl" {
		t.Fatalf("expected reentry_plasma_wake sample to use curl flow, got: %+v", cfg.Animation.Position.Flow)
	}
}
