package component

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

// SpriteRender handles rendering of sprite components
type SpriteRender struct{}

// NewSpriteRender creates a new sprite render system
func NewSpriteRender() *SpriteRender {
	return &SpriteRender{}
}

// Draw renders all sprite entities with positions
func (r *SpriteRender) Draw(ecs *ecs.ECS, screen *ebiten.Image) {
	// Query all entities that have both Sprite and Position components
	query := donburi.NewQuery(filter.Contains(Sprite, Position))

	query.Each(ecs.World, func(entry *donburi.Entry) {
		sprite := donburi.Get[SpriteData](entry, Sprite)
		position := donburi.Get[PositionData](entry, Position)

		if sprite.Image != nil {
			opts := &ebiten.DrawImageOptions{}

			// Get image dimensions
			bounds := sprite.Image.Bounds()
			width := float64(bounds.Dx())
			height := float64(bounds.Dy())

			// Apply scale transformation first
			if sprite.Scale > 0 {
				opts.GeoM.Scale(sprite.ScaleWidth, sprite.ScaleHeight)
				// Update dimensions for scaled image
				width *= sprite.ScaleWidth
				height *= sprite.ScaleHeight
			}

			// Apply rotation transformation
			if sprite.Rotation != 0 {
				// Rotate around center
				centerX := width / 2
				centerY := height / 2
				opts.GeoM.Translate(-centerX, -centerY)
				opts.GeoM.Rotate(sprite.Rotation)
				opts.GeoM.Translate(centerX, centerY)
			}

			// Apply anchor positioning
			switch sprite.Anchor {
			case AnchorTopLeft:
				opts.GeoM.Translate(position.X, position.Y)
			case AnchorTopCenter:
				opts.GeoM.Translate(position.X-width/2, position.Y)
			case AnchorTopRight:
				opts.GeoM.Translate(position.X-width, position.Y)
			case AnchorMiddleLeft:
				opts.GeoM.Translate(position.X, position.Y-height/2)
			case AnchorCenter:
				opts.GeoM.Translate(position.X-width/2, position.Y-height/2)
			case AnchorMiddleRight:
				opts.GeoM.Translate(position.X-width, position.Y-height/2)
			case AnchorBottomLeft:
				opts.GeoM.Translate(position.X, position.Y-height)
			case AnchorBottomCenter:
				opts.GeoM.Translate(position.X-width/2, position.Y-height)
			case AnchorBottomRight:
				opts.GeoM.Translate(position.X-width, position.Y-height)
			default:
				opts.GeoM.Translate(position.X, position.Y)
			}

			// Apply alpha and composite mode
			if sprite.Alpha >= 0 && sprite.Alpha < 1.0 {
				opts.ColorScale.Scale(
					1*float32(sprite.Alpha),
					1*float32(sprite.Alpha),
					1*float32(sprite.Alpha),
					float32(sprite.Alpha),
				)
			}
			opts.CompositeMode = sprite.CompositeMode

			screen.DrawImage(sprite.Image, opts)
		}
	})
}
