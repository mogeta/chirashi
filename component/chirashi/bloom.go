package chirashi

import "github.com/hajimehoshi/ebiten/v2"

// bloomThresholdShaderSrc extracts pixels brighter than Threshold with a soft knee.
const bloomThresholdShaderSrc = `//kage:unit pixels

package main

var Threshold float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	c := imageSrc0At(srcPos)
	luma := dot(c.rgb, vec3(0.2126, 0.7152, 0.0722))
	if luma <= Threshold {
		return vec4(0)
	}
	scale := (luma - Threshold) / max(luma, 0.0001)
	return c * scale
}
`

// bloomBlurShaderSrc is a 9-tap separable gaussian blur. Direction selects the
// axis (in pixels), so one shader serves both horizontal and vertical passes.
const bloomBlurShaderSrc = `//kage:unit pixels

package main

var Direction vec2

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	sum := imageSrc0At(srcPos) * 0.227027
	sum += (imageSrc0At(srcPos+Direction) + imageSrc0At(srcPos-Direction)) * 0.1945946
	sum += (imageSrc0At(srcPos+Direction*2.0) + imageSrc0At(srcPos-Direction*2.0)) * 0.1216216
	sum += (imageSrc0At(srcPos+Direction*3.0) + imageSrc0At(srcPos-Direction*3.0)) * 0.054054
	sum += (imageSrc0At(srcPos+Direction*4.0) + imageSrc0At(srcPos-Direction*4.0)) * 0.016216
	return sum
}
`

// BloomEffect implements a multi-pass bloom: bright-pass threshold extraction,
// half-resolution downsampling, separable gaussian blur, and additive
// compositing back onto the destination.
//
// Usage per frame, after rendering the scene into src:
//
//	bloom.Apply(dst, src) // dst and src may be the same image
type BloomEffect struct {
	// Threshold is the luminance cutoff in [0, 1]; only brighter pixels glow.
	Threshold float32
	// Intensity scales the additive glow contribution.
	Intensity float32
	// Passes is the number of blur iterations; more passes widen the glow.
	Passes int

	thresholdShader *ebiten.Shader
	blurShader      *ebiten.Shader

	bright   *ebiten.Image // full-res bright-pass output
	halfA    *ebiten.Image // half-res ping
	halfB    *ebiten.Image // half-res pong
	uniforms map[string]interface{}
}

// NewBloomEffect compiles the bloom shaders with sensible defaults.
func NewBloomEffect() (*BloomEffect, error) {
	thresholdShader, err := ebiten.NewShader([]byte(bloomThresholdShaderSrc))
	if err != nil {
		return nil, err
	}
	blurShader, err := ebiten.NewShader([]byte(bloomBlurShaderSrc))
	if err != nil {
		return nil, err
	}
	return &BloomEffect{
		Threshold:       0.6,
		Intensity:       1.2,
		Passes:          2,
		thresholdShader: thresholdShader,
		blurShader:      blurShader,
		uniforms:        make(map[string]interface{}, 2),
	}, nil
}

// Apply extracts bright areas from src, blurs them at half resolution, and
// additively composites the glow onto dst. dst and src may be the same image.
func (b *BloomEffect) Apply(dst, src *ebiten.Image) {
	w := src.Bounds().Dx()
	h := src.Bounds().Dy()
	if w < 2 || h < 2 {
		return
	}
	b.ensureBuffers(w, h)

	// Bright-pass threshold at full resolution.
	b.bright.Clear()
	clear(b.uniforms)
	b.uniforms["Threshold"] = b.Threshold
	thresholdOp := &ebiten.DrawRectShaderOptions{Uniforms: b.uniforms}
	thresholdOp.Images[0] = src
	b.bright.DrawRectShader(w, h, b.thresholdShader, thresholdOp)

	// Downsample to half resolution.
	b.halfA.Clear()
	downOp := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	downOp.GeoM.Scale(0.5, 0.5)
	b.halfA.DrawImage(b.bright, downOp)

	// Separable gaussian blur, ping-ponging between the half buffers.
	halfW := b.halfA.Bounds().Dx()
	halfH := b.halfA.Bounds().Dy()
	passes := b.Passes
	if passes < 1 {
		passes = 1
	}
	for i := 0; i < passes; i++ {
		b.blurPass(b.halfB, b.halfA, halfW, halfH, [2]float32{1, 0})
		b.blurPass(b.halfA, b.halfB, halfW, halfH, [2]float32{0, 1})
	}

	// Upscale and add onto the destination.
	upOp := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
	upOp.GeoM.Scale(float64(w)/float64(halfW), float64(h)/float64(halfH))
	upOp.ColorScale.ScaleAlpha(b.Intensity)
	dst.DrawImage(b.halfA, upOp)
}

func (b *BloomEffect) blurPass(dst, src *ebiten.Image, w, h int, direction [2]float32) {
	dst.Clear()
	clear(b.uniforms)
	b.uniforms["Direction"] = direction
	op := &ebiten.DrawRectShaderOptions{Uniforms: b.uniforms}
	op.Images[0] = src
	dst.DrawRectShader(w, h, b.blurShader, op)
}

func (b *BloomEffect) ensureBuffers(w, h int) {
	if b.bright != nil && b.bright.Bounds().Dx() == w && b.bright.Bounds().Dy() == h {
		return
	}
	b.bright = ebiten.NewImage(w, h)
	b.halfA = ebiten.NewImage(w/2, h/2)
	b.halfB = ebiten.NewImage(w/2, h/2)
}
