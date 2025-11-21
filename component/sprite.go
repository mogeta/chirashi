package component

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
)

type SpriteData struct {
	Image  *ebiten.Image
	Anchor Anchor
	// Particle-specific properties
	Alpha         float64 // Alpha transparency (0.0-1.0)
	Rotation      float64 // Rotation in radians
	Scale         float64 // Scale factor
	ScaleWidth    float64
	ScaleHeight   float64
	CompositeMode ebiten.CompositeMode // Blending mode
}

type Anchor int

const (
	AnchorTopLeft Anchor = iota
	AnchorTopCenter
	AnchorTopRight
	AnchorMiddleLeft
	AnchorCenter
	AnchorMiddleRight
	AnchorBottomLeft
	AnchorBottomCenter
	AnchorBottomRight
)

var Sprite = donburi.NewComponentType[SpriteData]()

// NewBasicSprite creates a basic sprite with default values
func NewBasicSprite(image *ebiten.Image, anchor Anchor) SpriteData {
	return SpriteData{
		Image:         image,
		Anchor:        anchor,
		Alpha:         1.0,
		Rotation:      0.0,
		Scale:         1.0,
		CompositeMode: ebiten.CompositeModeSourceOver,
	}
}

// NewParticleSprite creates a sprite optimized for particle rendering
func NewParticleSprite(image *ebiten.Image, alpha, rotation, scale float64) SpriteData {
	return SpriteData{
		Image:         image,
		Anchor:        AnchorCenter,
		Alpha:         alpha,
		Rotation:      rotation,
		Scale:         scale,
		ScaleHeight:   scale,
		ScaleWidth:    scale,
		CompositeMode: ebiten.CompositeModeLighter,
	}
}
