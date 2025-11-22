package scenes

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"chirashi/component"
	"chirashi/component/chirashi"

	"github.com/ebitengine/debugui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

type ParticleEditorScene struct {
	world     donburi.World
	container *ecs.ECS
	config    *chirashi.ParticleConfig
	img       *ebiten.Image
	debugui   debugui.DebugUI
}

func NewParticleEditorScene() *ParticleEditorScene {
	world := donburi.NewWorld()
	container := ecs.NewECS(world)

	particleSys := chirashi.NewSpriteSystem()
	container.AddSystem(particleSys.Update)

	spriteRender := component.NewSpriteRender()
	container.AddRenderer(0, spriteRender.Draw)

	// Create debug particle image
	img := ebiten.NewImage(8, 8)
	img.Fill(color.White)

	// Load configuration from file
	configPath := "assets/particles/sample.yaml"
	config, err := chirashi.GetConfigLoader().LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)

	}

	// Create particles from config
	if err := chirashi.NewParticlesFromConfig(world, img, config, 640, 480); err != nil {
		log.Fatal(err)
	}

	return &ParticleEditorScene{
		world:     world,
		container: container,
		config:    config,
		img:       img,
	}
}

func (s *ParticleEditorScene) Update() error {
	// Update DebugUI
	if _, err := s.debugui.Update(func(ctx *debugui.Context) error {
		// Window 1: General Settings (Top Left)
		s.drawGeneralSettingsWindow(ctx)

		// Window 2: Movement X (Bottom Left)
		s.drawMovementXWindow(ctx)

		// Window 3: Movement Y (Top Right)
		s.drawMovementYWindow(ctx)

		// Window 4: Appearance (Bottom Right)
		s.drawAppearanceWindow(ctx)

		// Window 5: Debug Info (Top Center)
		s.drawDebugWindow(ctx)

		return nil
	}); err != nil {
		return err
	}

	s.container.Update()
	return nil
}

func (s *ParticleEditorScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0x20, 0x20, 0x20, 0xff})
	s.container.Draw(screen)
	s.debugui.Draw(screen)
}

func (s *ParticleEditorScene) recreateParticles() {
	// Clear existing particles
	// Query all entities with the Particle Component and remove them
	query := donburi.NewQuery(filter.Contains(chirashi.Component))
	// We need to collect entries first to avoid modification during iteration issues if any
	var entries []*donburi.Entry
	query.Each(s.world, func(entry *donburi.Entry) {
		entries = append(entries, entry)
	})

	for _, entry := range entries {
		// Clean up sprite entities associated with this particle system
		particleComponent := chirashi.Component.Get(entry)
		for i := range particleComponent.ParticlePool {
			particle := &particleComponent.ParticlePool[i]
			if particle.SpriteEntity != nil && particle.SpriteEntity.Valid() {
				s.world.Remove(particle.SpriteEntity.Entity())
			}
		}
		s.world.Remove(entry.Entity())
	}

	// Create new particles at center of 1280x960 screen
	chirashi.NewParticlesFromConfig(s.world, s.img, s.config, 640, 480)
}

func (s *ParticleEditorScene) tweenControls(ctx *debugui.Context, label string, config *chirashi.TweenConfig, min, max, stepVal float64) {
	ctx.IDScope(label, func() {
		ctx.SetGridLayout([]int{-1}, nil)
		ctx.Text(label)

		// Iterate over all steps
		for i := range config.Steps {
			// Use index to create unique ID scope for each step
			ctx.IDScope(fmt.Sprintf("Step_%d", i), func() {
				step := &config.Steps[i]

				ctx.Text(fmt.Sprintf("Step %d", i+1))

				// Random Start Toggle
				isRandomStart := step.FromRange != nil
				ctx.Button(fmt.Sprintf("Random Start: %v", isRandomStart)).On(func() {
					if isRandomStart {
						step.FromRange = nil
					} else {
						step.FromRange = &chirashi.RangeData{Min: step.From, Max: step.From}
					}
					s.recreateParticles()
				})

				if step.FromRange != nil {
					s.sliderControl(ctx, "From Min", &step.FromRange.Min, min, max, stepVal)
					s.sliderControl(ctx, "From Max", &step.FromRange.Max, min, max, stepVal)
				} else {
					s.sliderControl(ctx, "From", &step.From, min, max, stepVal)
				}

				// Relative Toggle
				ctx.Button(fmt.Sprintf("Relative: %v", step.IsRelative)).On(func() {
					step.IsRelative = !step.IsRelative
					s.recreateParticles()
				})

				// Random Range Toggle
				isRandom := step.ToRange != nil
				ctx.Button(fmt.Sprintf("Random Target: %v", isRandom)).On(func() {
					if isRandom {
						step.ToRange = nil
					} else {
						step.ToRange = &chirashi.RangeData{Min: step.To, Max: step.To}
					}
					s.recreateParticles()
				})

				if step.ToRange != nil {
					s.sliderControl(ctx, "To Min", &step.ToRange.Min, min, max, stepVal)
					s.sliderControl(ctx, "To Max", &step.ToRange.Max, min, max, stepVal)
				} else {
					s.sliderControl(ctx, "To", &step.To, min, max, stepVal)
				}
				s.sliderControl(ctx, "Duration", &step.Duration, 0, 600, 0.1)

				// Easing Cycler
				ctx.Text("  Ease: " + step.Easing)
				ctx.Button("  Cycle Ease").On(func() {
					step.Easing = s.cycleEasing(step.Easing)
					s.recreateParticles()
				})

				// Remove Step Button
				ctx.Button("  Remove Step").On(func() {
					// Remove element at index i
					config.Steps = append(config.Steps[:i], config.Steps[i+1:]...)
					s.recreateParticles()
				})

				ctx.Text("----------------")
			})
		}

		// Add Step Button
		ctx.Button("Add Step").On(func() {
			// Add a new step with default values (or copy previous)
			newStep := chirashi.TweenStep{Duration: 1, Easing: "Linear"}
			if len(config.Steps) > 0 {
				lastStep := config.Steps[len(config.Steps)-1]
				newStep.From = lastStep.To
				newStep.To = lastStep.To // Start where last ended
			}
			config.Steps = append(config.Steps, newStep)
			s.recreateParticles()
		})
	})
}

func (s *ParticleEditorScene) sliderControl(ctx *debugui.Context, label string, value *float64, min, max, step float64) {
	ctx.IDScope(label, func() {
		ctx.Text(fmt.Sprintf("  %s: %.2f", label, *value))

		// Layout: [-] [Slider] [+]
		// We use 3 columns: fixed width for buttons, remaining for slider
		ctx.SetGridLayout([]int{30, -1, 30}, nil)

		ctx.Button("-").On(func() {
			*value -= step
			if *value < min {
				*value = min
			}
			s.recreateParticles()
		})

		ctx.SliderF(value, min, max, step, 2).On(func() {
			s.recreateParticles()
		})

		ctx.Button("+").On(func() {
			*value += step
			if *value > max {
				*value = max
			}
			s.recreateParticles()
		})

		// Reset layout to single column full width
		ctx.SetGridLayout([]int{-1}, nil)
	})
}

func (s *ParticleEditorScene) cycleEasing(current string) string {
	easings := []string{
		"Linear",
		"InQuad", "OutQuad", "InOutQuad",
		"InCubic", "OutCubic", "InOutCubic",
		"InQuart", "OutQuart", "InOutQuart",
		"InQuint", "OutQuint", "InOutQuint",
		"InSine", "OutSine", "InOutSine",
		"InExpo", "OutExpo", "InOutExpo",
		"InCirc", "OutCirc", "InOutCirc",
		"InBack", "OutBack", "InOutBack",
		"InElastic", "OutElastic", "InOutElastic",
		"InBounce", "OutBounce", "InOutBounce",
	}

	for i, e := range easings {
		if e == current {
			return easings[(i+1)%len(easings)]
		}
	}
	return "Linear"
}

func (s *ParticleEditorScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 1280, 960 // Larger resolution for editor
}

func (s *ParticleEditorScene) drawGeneralSettingsWindow(ctx *debugui.Context) {
	ctx.Window("General Settings", image.Rect(10, 10, 410, 310), func(layout debugui.ContainerLayout) {
		ctx.Text("Spawn Config")
		ctx.Text("Interval: " + fmt.Sprintf("%d", s.config.Spawn.Interval))
		ctx.Button("I+").On(func() { s.config.Spawn.Interval++; s.recreateParticles() })
		ctx.Button("I-").On(func() { s.config.Spawn.Interval--; s.recreateParticles() })

		ctx.Text("Per Spawn: " + fmt.Sprintf("%d", s.config.Spawn.ParticlesPerSpawn))
		ctx.Button("P+").On(func() { s.config.Spawn.ParticlesPerSpawn++; s.recreateParticles() })
		ctx.Button("P-").On(func() { s.config.Spawn.ParticlesPerSpawn--; s.recreateParticles() })

		ctx.Text("Max Particles: " + fmt.Sprintf("%d", s.config.Spawn.MaxParticles))

		ctx.Text("Emitter")
		ctx.Text("X: " + fmt.Sprintf("%.1f", s.config.Emitter.Position.X))
		ctx.Button("X+").On(func() { s.config.Emitter.Position.X += 10; s.recreateParticles() })
		ctx.Button("X-").On(func() { s.config.Emitter.Position.X -= 10; s.recreateParticles() })

		ctx.Text("Y: " + fmt.Sprintf("%.1f", s.config.Emitter.Position.Y))
		ctx.Button("Y+").On(func() { s.config.Emitter.Position.Y += 10; s.recreateParticles() })
		ctx.Button("Y-").On(func() { s.config.Emitter.Position.Y -= 10; s.recreateParticles() })

		ctx.Text("----------------")

		// Movement Type Toggle
		currentType := s.config.Movement.Type
		if currentType == "" {
			currentType = "cartesian"
		}
		ctx.Text("Type: " + currentType)
		ctx.Button("Toggle Type").On(func() {
			if s.config.Movement.Type == "polar" {
				s.config.Movement.Type = "cartesian"
			} else {
				s.config.Movement.Type = "polar"
			}
			s.recreateParticles()
		})
	})
}

func (s *ParticleEditorScene) drawMovementXWindow(ctx *debugui.Context) {
	title := "Movement X"
	if s.config.Movement.Type == "polar" {
		title = "Movement Angle"
	}
	ctx.Window(title, image.Rect(10, 320, 410, 620), func(layout debugui.ContainerLayout) {
		if s.config.Movement.Type == "polar" {
			s.tweenControls(ctx, "Angle", &s.config.Movement.Angle, 0, 360, 15.0)
		} else {
			s.tweenControls(ctx, "X Axis", &s.config.Movement.X, -1000, 1000, 5.0)
		}
	})
}

func (s *ParticleEditorScene) drawMovementYWindow(ctx *debugui.Context) {
	title := "Movement Y"
	if s.config.Movement.Type == "polar" {
		title = "Movement Dist"
	}
	ctx.Window(title, image.Rect(870, 10, 1270, 310), func(layout debugui.ContainerLayout) {
		if s.config.Movement.Type == "polar" {
			s.tweenControls(ctx, "Distance", &s.config.Movement.Distance, 0, 1000, 5.0)
		} else {
			s.tweenControls(ctx, "Y Axis", &s.config.Movement.Y, -1000, 1000, 5.0)
		}
	})
}

func (s *ParticleEditorScene) drawAppearanceWindow(ctx *debugui.Context) {
	ctx.Window("Appearance", image.Rect(870, 320, 1270, 720), func(layout debugui.ContainerLayout) {
		s.tweenControls(ctx, "Alpha", &s.config.Appearance.Alpha, 0, 1, 0.01)
		s.tweenControls(ctx, "Rotation", &s.config.Appearance.Rotation, -360, 360, 5.0)
		s.tweenControls(ctx, "Scale", &s.config.Appearance.Scale, -10, 10, 0.1)
	})
}

func (s *ParticleEditorScene) drawDebugWindow(ctx *debugui.Context) {
	ctx.Window("Debug Info", image.Rect(420, 10, 620, 110), func(layout debugui.ContainerLayout) {
		fps := ebiten.ActualFPS()
		ctx.Text(fmt.Sprintf("FPS: %.2f", fps))

		// Count active sprite entities
		count := donburi.NewQuery(filter.Contains(component.Sprite)).Count(s.world)
		ctx.Text(fmt.Sprintf("Objects: %d", count))
	})
}
