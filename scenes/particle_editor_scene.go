package scenes

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"path/filepath"
	"time"

	"github.com/ebitengine/debugui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"

	"chirashi/assets"
	"chirashi/component"
	"chirashi/component/chirashi"
)

type ParticleEditorScene struct {
	world           donburi.World
	container       *ecs.ECS
	config          *chirashi.ParticleConfig
	img             *ebiten.Image
	debugui         debugui.DebugUI
	shader          *ebiten.Shader
	offscreen       *ebiten.Image
	glitchIntensity float64
	time            float64
	fileList        []string
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
	log.Println("Loading particle configuration...")
	config, err := chirashi.GetConfigLoader().LoadConfigFromBytes(assets.SampleParticleConfig, "sample.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}
	log.Println("Particle configuration loaded successfully")

	// Create particles from config
	if err := chirashi.NewParticlesFromConfig(world, img, config, 640, 480); err != nil {
		log.Fatal(err)
	}

	// Load Shader
	shader, err := ebiten.NewShader(assets.BloomShader)
	if err != nil {
		log.Fatal(err)
	}

	return &ParticleEditorScene{
		world:     world,
		container: container,
		config:    config,
		img:       img,
		shader:    shader,
	}
}

func (s *ParticleEditorScene) Update() error {
	s.time += 1.0 / 60.0

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

		// Window 6: File Operations (Bottom Left)
		s.drawFileWindow(ctx)

		return nil
	}); err != nil {
		return err
	}

	s.container.Update()
	return nil
}

func (s *ParticleEditorScene) Draw(screen *ebiten.Image) {
	if s.offscreen == nil {
		s.offscreen = ebiten.NewImage(screen.Bounds().Dx(), screen.Bounds().Dy())
	}

	// Clear offscreen
	s.offscreen.Fill(color.RGBA{0x20, 0x20, 0x20, 0xff})

	// Draw particles to offscreen
	s.container.Draw(s.offscreen)

	// Apply shader and draw to screen
	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = s.offscreen
	op.Uniforms = map[string]interface{}{
		"Time":            float32(s.time),
		"GlitchIntensity": float32(s.glitchIntensity),
	}
	screen.DrawRectShader(screen.Bounds().Dx(), screen.Bounds().Dy(), s.shader, op)

	// Draw UI on top
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
	if err := chirashi.NewParticlesFromConfig(s.world, s.img, s.config, 640, 480); err != nil {
		log.Println("Failed to recreate particles:", err)
	}
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
			newStep := chirashi.TweenStep{Duration: 60, Easing: "Linear"}
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
	if s.offscreen != nil && (s.offscreen.Bounds().Dx() != outsideWidth || s.offscreen.Bounds().Dy() != outsideHeight) {
		s.offscreen.Dispose()
		s.offscreen = nil
	}
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

		ctx.Text("----------------")

		// Glitch Intensity
		s.sliderControl(ctx, "Glitch Intensity", &s.glitchIntensity, 0.0, 1.0, 0.01)
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

func (s *ParticleEditorScene) refreshFileList() {
	files, err := chirashi.GetConfigLoader().ListConfigs("assets/particles/*.yaml")
	if err != nil {
		log.Println("Failed to list files:", err)
		return
	}
	s.fileList = files
}

func (s *ParticleEditorScene) drawFileWindow(ctx *debugui.Context) {
	ctx.Window("File Operations", image.Rect(10, 630, 410, 850), func(layout debugui.ContainerLayout) {
		// Save
		ctx.Button("Save " + s.config.Name + ".yaml").On(func() {
			path := filepath.Join("assets", "particles", s.config.Name+".yaml")
			err := chirashi.GetConfigLoader().SaveConfig(path, s.config)
			if err != nil {
				log.Println("Save error:", err)
			} else {
				log.Println("Saved to", path)
				s.refreshFileList()
			}
		})

		// Save As New
		ctx.Button("Save As New").On(func() {
			timestamp := time.Now().Format("20060102_150405")
			newName := "particle_" + timestamp
			s.config.Name = newName // Update internal name too
			path := filepath.Join("assets", "particles", newName+".yaml")
			err := chirashi.GetConfigLoader().SaveConfig(path, s.config)
			if err != nil {
				log.Println("Save error:", err)
			} else {
				log.Println("Saved to", path)
				s.refreshFileList()
			}
		})

		ctx.Text("----------------")
		ctx.Text("Load File:")

		// Refresh list
		ctx.Button("Refresh List").On(func() {
			s.refreshFileList()
		})

		for _, f := range s.fileList {
			name := filepath.Base(f)
			// Capture loop variable
			f := f
			ctx.Button(name).On(func() {
				// Clear cache to ensure fresh load
				chirashi.GetConfigLoader().ClearCache()

				cfg, err := chirashi.GetConfigLoader().LoadConfig(f)
				if err != nil {
					log.Println("Load error:", err)
				} else {
					s.config = cfg
					s.recreateParticles()
					log.Println("Loaded", name)
				}
			})
		}
	})
}
