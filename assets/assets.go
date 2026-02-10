package assets

import (
	_ "embed"
)

//go:embed shaders/bloom.kage
var BloomShader []byte

//go:embed shaders/particle.kage
var ParticleShader []byte

//go:embed particles/sample.yaml
var SampleParticleConfig []byte
