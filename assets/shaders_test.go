package assets

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestEmbeddedShadersCompile(t *testing.T) {
	cases := []struct {
		name string
		src  []byte
	}{
		{"particle", ParticleShader},
		{"particle_blur", ParticleShaderBlur},
		{"bloom", BloomShader},
	}
	for _, c := range cases {
		if _, err := ebiten.NewShader(c.src); err != nil {
			t.Errorf("shader %s failed to compile: %v", c.name, err)
		}
	}
}
