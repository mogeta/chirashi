package chirashi

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestParseBlendMode(t *testing.T) {
	cases := []struct {
		name string
		want ebiten.Blend
	}{
		{"", ebiten.BlendSourceOver},
		{"normal", ebiten.BlendSourceOver},
		{"additive", ebiten.BlendLighter},
		{"lighter", ebiten.BlendLighter},
		{"unknown", ebiten.BlendSourceOver},
	}
	for _, c := range cases {
		if got := ParseBlendMode(c.name); got != c.want {
			t.Errorf("ParseBlendMode(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestBuildSystemDataFromConfigSetsBlend(t *testing.T) {
	config := &ParticleConfig{
		Blend: "additive",
		Spawn: SpawnConfig{MaxParticles: 4},
	}
	data := buildSystemDataFromConfig(nil, nil, config, 0, 0)
	if data.Blend != ebiten.BlendLighter {
		t.Errorf("Blend = %v, want BlendLighter", data.Blend)
	}
}

func TestBuildSystemDataFromConfigDefaultsEmissionScale(t *testing.T) {
	config := &ParticleConfig{
		Spawn: SpawnConfig{MaxParticles: 4},
	}
	data := buildSystemDataFromConfig(nil, nil, config, 0, 0)
	if data.EmissionScale != 1 {
		t.Errorf("EmissionScale = %v, want 1", data.EmissionScale)
	}
}
