package assets

import (
	_ "embed"
)

//go:embed shaders/bloom.kage
var BloomShader []byte
