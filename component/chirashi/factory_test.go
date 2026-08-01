package chirashi

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
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

func TestResolveParticleShaderCompilesAndCachesBuiltinBlur(t *testing.T) {
	first, err := resolveParticleShader(nil, "blur")
	if err != nil {
		t.Fatalf("resolveParticleShader: %v", err)
	}
	second, err := resolveParticleShader(nil, "blur")
	if err != nil {
		t.Fatalf("resolveParticleShader second call: %v", err)
	}
	if first == nil || first != second {
		t.Fatalf("built-in blur shader was not cached: first=%p second=%p", first, second)
	}
}

func TestResolveParticleShaderRejectsUnknownName(t *testing.T) {
	if _, err := resolveParticleShader(nil, "pixelate"); err == nil {
		t.Fatal("expected unknown particle shader to fail")
	}
}

func TestCreateParticleEntityUsesConfiguredBlurShader(t *testing.T) {
	world := donburi.NewWorld()
	config := &ParticleConfig{
		Render: RenderConfig{ParticleShader: "blur"},
		Spawn:  SpawnConfig{MaxParticles: 1},
	}
	entity, err := createParticleEntityFromConfig(world, nil, nil, config, 0, 0)
	if err != nil {
		t.Fatalf("createParticleEntityFromConfig: %v", err)
	}
	if shader := Component.Get(world.Entry(entity)).Shader; shader == nil {
		t.Fatal("configured blur shader was not assigned to SystemData")
	}
}
