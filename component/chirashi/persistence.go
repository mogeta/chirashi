package chirashi

import "github.com/hajimehoshi/ebiten/v2"

// PersistenceEffect accumulates rendered frames into an offscreen buffer that
// fades a little each frame, producing an afterimage ("light trail") look.
//
// Usage per frame:
//
//	target := effect.Target(w, h) // fades previous frames, returns draw target
//	drawParticles(target)         // render this frame's content into target
//	effect.Compose(screen)        // composite the accumulated image onto screen
type PersistenceEffect struct {
	// Decay is the per-frame retention of previous frames in [0, 1).
	// 0.9 keeps 90% of the previous frame's brightness each frame.
	Decay float32

	front  *ebiten.Image
	back   *ebiten.Image
	fadeOp ebiten.DrawImageOptions
}

// NewPersistenceEffect creates an afterimage effect with the given per-frame decay.
func NewPersistenceEffect(decay float32) *PersistenceEffect {
	return &PersistenceEffect{Decay: decay}
}

// Target prepares and returns the accumulation buffer to draw this frame's
// content into. Buffers are (re)allocated when the size changes.
func (e *PersistenceEffect) Target(width, height int) *ebiten.Image {
	if e.front == nil || e.front.Bounds().Dx() != width || e.front.Bounds().Dy() != height {
		e.front = ebiten.NewImage(width, height)
		e.back = ebiten.NewImage(width, height)
	}
	e.front, e.back = e.back, e.front
	e.front.Clear()
	// Copying through a second buffer (instead of fading in place) lets the
	// scaled-down values round toward zero, so old frames fully disappear.
	e.fadeOp.ColorScale.Reset()
	e.fadeOp.ColorScale.ScaleAlpha(e.Decay)
	e.front.DrawImage(e.back, &e.fadeOp)
	return e.front
}

// Compose draws the accumulated buffer onto dst with regular alpha blending.
func (e *PersistenceEffect) Compose(dst *ebiten.Image) {
	if e.front == nil {
		return
	}
	dst.DrawImage(e.front, nil)
}

// Clear resets the accumulation buffers.
func (e *PersistenceEffect) Clear() {
	if e.front != nil {
		e.front.Clear()
		e.back.Clear()
	}
}
