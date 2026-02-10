package editor

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

	"github.com/mogeta/chirashi/assets"
	"github.com/mogeta/chirashi/component/chirashi"
)

type ParticleEditorScene struct {
	world           donburi.World
	container       *ecs.ECS
	config          *chirashi.ParticleConfig
	img             *ebiten.Image
	debugui         debugui.DebugUI
	shader          *ebiten.Shader
	bloomShader     *ebiten.Shader
	offscreen       *ebiten.Image
	glitchIntensity float64
	time            float64
	fileList        []string
}

func NewParticleEditorScene() *ParticleEditorScene {
	world := donburi.NewWorld()
	container := ecs.NewECS(world)

	particleSys := chirashi.NewSystem()
	container.AddSystem(particleSys.Update)
	container.AddRenderer(0, particleSys.Draw)

	// Create debug particle image
	img := ebiten.NewImage(8, 8)
	img.Fill(color.White)

	// Load particle shader
	shader, err := ebiten.NewShader(assets.ParticleShader)
	if err != nil {
		log.Fatalf("Failed to load particle shader: %v\n", err)
	}

	// Load bloom shader
	bloomShader, err := ebiten.NewShader(assets.BloomShader)
	if err != nil {
		log.Fatalf("Failed to load bloom shader: %v\n", err)
	}

	// Load configuration from file
	log.Println("Loading particle configuration...")
	config, err := chirashi.GetConfigLoader().LoadConfigFromBytes(assets.SampleParticleConfig, "sample.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}
	log.Println("Particle configuration loaded successfully")

	// Create particles from config
	if err := chirashi.NewParticlesFromConfig(world, shader, img, config, 640, 480); err != nil {
		log.Fatal(err)
	}

	return &ParticleEditorScene{
		world:       world,
		container:   container,
		config:      config,
		img:         img,
		shader:      shader,
		bloomShader: bloomShader,
	}
}

func (s *ParticleEditorScene) Update() error {
	s.time += 1.0 / 60.0

	// Update DebugUI
	if _, err := s.debugui.Update(func(ctx *debugui.Context) error {
		s.drawGeneralSettingsWindow(ctx)
		s.drawAnimationWindow(ctx)
		s.drawDebugWindow(ctx)
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
	screen.DrawRectShader(screen.Bounds().Dx(), screen.Bounds().Dy(), s.bloomShader, op)

	// Draw UI on top
	s.debugui.Draw(screen)
}

func (s *ParticleEditorScene) recreateParticles() {
	// Clear existing particles
	query := donburi.NewQuery(filter.Contains(chirashi.Component))
	var entries []*donburi.Entry
	query.Each(s.world, func(entry *donburi.Entry) {
		entries = append(entries, entry)
	})

	log.Printf("Removing %d existing particle systems", len(entries))
	for _, entry := range entries {
		s.world.Remove(entry.Entity())
	}

	// Create new particles
	log.Printf("Creating particles with config: %s, PosType=%s, UsePolar=%v",
		s.config.Name,
		s.config.Animation.Position.Type,
		s.config.Animation.Position.Type == "polar")
	if err := chirashi.NewParticlesFromConfig(s.world, s.shader, s.img, s.config, 640, 480); err != nil {
		log.Println("Failed to recreate particles:", err)
	} else {
		log.Println("Particles recreated successfully")
	}
}

func (s *ParticleEditorScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	if s.offscreen != nil && (s.offscreen.Bounds().Dx() != outsideWidth || s.offscreen.Bounds().Dy() != outsideHeight) {
		s.offscreen.Deallocate()
		s.offscreen = nil
	}
	return 1280, 960
}

func (s *ParticleEditorScene) drawGeneralSettingsWindow(ctx *debugui.Context) {
	ctx.Window("General Settings", image.Rect(10, 10, 410, 350), func(layout debugui.ContainerLayout) {
		ctx.Text("Spawn Config")

		ctx.Text(fmt.Sprintf("Interval: %d", s.config.Spawn.Interval))
		ctx.SetGridLayout([]int{60, 60}, nil)
		ctx.Button("I+").On(func() { s.config.Spawn.Interval++; s.recreateParticles() })
		ctx.Button("I-").On(func() {
			if s.config.Spawn.Interval > 1 {
				s.config.Spawn.Interval--
			}
			s.recreateParticles()
		})
		ctx.SetGridLayout([]int{-1}, nil)

		ctx.Text(fmt.Sprintf("Per Spawn: %d", s.config.Spawn.ParticlesPerSpawn))
		ctx.SetGridLayout([]int{60, 60}, nil)
		ctx.Button("P+").On(func() { s.config.Spawn.ParticlesPerSpawn += 10; s.recreateParticles() })
		ctx.Button("P-").On(func() {
			if s.config.Spawn.ParticlesPerSpawn > 1 {
				s.config.Spawn.ParticlesPerSpawn -= 10
			}
			s.recreateParticles()
		})
		ctx.SetGridLayout([]int{-1}, nil)

		ctx.Text(fmt.Sprintf("Max Particles: %d", s.config.Spawn.MaxParticles))
		ctx.SetGridLayout([]int{60, 60}, nil)
		ctx.Button("M+").On(func() { s.config.Spawn.MaxParticles += 1000; s.recreateParticles() })
		ctx.Button("M-").On(func() {
			if s.config.Spawn.MaxParticles > 100 {
				s.config.Spawn.MaxParticles -= 1000
			}
			s.recreateParticles()
		})
		ctx.SetGridLayout([]int{-1}, nil)

		ctx.Text("----------------")
		ctx.Text("Emitter")

		ctx.Text(fmt.Sprintf("X: %.1f", s.config.Emitter.X))
		ctx.SetGridLayout([]int{60, 60}, nil)
		ctx.Button("X+").On(func() { s.config.Emitter.X += 10; s.recreateParticles() })
		ctx.Button("X-").On(func() { s.config.Emitter.X -= 10; s.recreateParticles() })
		ctx.SetGridLayout([]int{-1}, nil)

		ctx.Text(fmt.Sprintf("Y: %.1f", s.config.Emitter.Y))
		ctx.SetGridLayout([]int{60, 60}, nil)
		ctx.Button("Y+").On(func() { s.config.Emitter.Y += 10; s.recreateParticles() })
		ctx.Button("Y-").On(func() { s.config.Emitter.Y -= 10; s.recreateParticles() })
		ctx.SetGridLayout([]int{-1}, nil)

		ctx.Text("----------------")

		// Glitch Intensity
		s.sliderControl(ctx, "Glitch Intensity", &s.glitchIntensity, 0.0, 1.0, 0.01)
	})
}

func (s *ParticleEditorScene) drawAnimationWindow(ctx *debugui.Context) {
	ctx.Window("Animation", image.Rect(10, 360, 410, 950), func(layout debugui.ContainerLayout) {
		// Duration
		ctx.Text("Duration")
		dv := float64(s.config.Animation.Duration.Value)
		s.sliderControl(ctx, "Value", &dv, 0.1, 10.0, 0.1)
		s.config.Animation.Duration.Value = float32(dv)

		ctx.Text("----------------")

		// Position mode toggle
		posType := s.config.Animation.Position.Type
		if posType == "" {
			posType = "cartesian"
		}
		ctx.Text("Position Mode: " + posType)
		ctx.Button("Toggle Mode").On(func() {
			if s.config.Animation.Position.Type == "polar" {
				s.config.Animation.Position.Type = "cartesian"
				// Initialize cartesian defaults
				if s.config.Animation.Position.EndX == nil {
					s.config.Animation.Position.EndX = &chirashi.RangeFloat{Min: -100, Max: 100}
				}
				if s.config.Animation.Position.EndY == nil {
					s.config.Animation.Position.EndY = &chirashi.RangeFloat{Min: -100, Max: 100}
				}
			} else {
				s.config.Animation.Position.Type = "polar"
				// Initialize polar defaults
				if s.config.Animation.Position.Angle == nil {
					s.config.Animation.Position.Angle = &chirashi.RangeFloat{Min: 0, Max: 6.283185}
				}
				if s.config.Animation.Position.Distance == nil {
					s.config.Animation.Position.Distance = &chirashi.RangeFloat{Min: 50, Max: 150}
				}
			}
			s.recreateParticles()
		})

		if s.config.Animation.Position.Type == "polar" {
			// Polar mode controls
			s.rangeControl(ctx, "Angle", s.config.Animation.Position.Angle, 0, 6.283185, 0.1)
			s.rangeControl(ctx, "Distance", s.config.Animation.Position.Distance, 0, 500, 10)
		} else {
			// Cartesian mode controls
			s.rangeControl(ctx, "End X", s.config.Animation.Position.EndX, -500, 500, 10)
			s.rangeControl(ctx, "End Y", s.config.Animation.Position.EndY, -500, 500, 10)
		}

		ctx.Text("  Easing: " + s.config.Animation.Position.Easing)
		ctx.Button("  Cycle Easing").On(func() {
			s.config.Animation.Position.Easing = s.cycleEasing(s.config.Animation.Position.Easing)
			s.recreateParticles()
		})

		ctx.Text("----------------")

		// Alpha
		ctx.Text("Alpha")
		s.propertyModeToggle(ctx, "Alpha", &s.config.Animation.Alpha)
		if s.config.Animation.Alpha.IsSequence() {
			s.sequenceControls(ctx, "Alpha", &s.config.Animation.Alpha, 0.0, 1.0, 0.05)
		} else {
			startAlpha := float64(s.config.Animation.Alpha.Start)
			endAlpha := float64(s.config.Animation.Alpha.End)
			s.sliderControl(ctx, "Start", &startAlpha, 0.0, 1.0, 0.05)
			s.sliderControl(ctx, "End", &endAlpha, 0.0, 1.0, 0.05)
			s.config.Animation.Alpha.Start = float32(startAlpha)
			s.config.Animation.Alpha.End = float32(endAlpha)
			ctx.Text("  Easing: " + s.config.Animation.Alpha.Easing)
			ctx.Button("  Cycle Easing##alpha").On(func() {
				s.config.Animation.Alpha.Easing = s.cycleEasing(s.config.Animation.Alpha.Easing)
				s.recreateParticles()
			})
		}

		ctx.Text("----------------")

		// Scale
		ctx.Text("Scale")
		s.propertyModeToggle(ctx, "Scale", &s.config.Animation.Scale)
		if s.config.Animation.Scale.IsSequence() {
			s.sequenceControls(ctx, "Scale", &s.config.Animation.Scale, 0.0, 5.0, 0.1)
		} else {
			startScale := float64(s.config.Animation.Scale.Start)
			endScale := float64(s.config.Animation.Scale.End)
			s.sliderControl(ctx, "Start##scale", &startScale, 0.0, 5.0, 0.1)
			s.sliderControl(ctx, "End##scale", &endScale, 0.0, 5.0, 0.1)
			s.config.Animation.Scale.Start = float32(startScale)
			s.config.Animation.Scale.End = float32(endScale)
			ctx.Text("  Easing: " + s.config.Animation.Scale.Easing)
			ctx.Button("  Cycle Easing##scale").On(func() {
				s.config.Animation.Scale.Easing = s.cycleEasing(s.config.Animation.Scale.Easing)
				s.recreateParticles()
			})
		}

		ctx.Text("----------------")

		// Rotation
		ctx.Text("Rotation")
		s.propertyModeToggle(ctx, "Rotation", &s.config.Animation.Rotation)
		if s.config.Animation.Rotation.IsSequence() {
			s.sequenceControls(ctx, "Rotation", &s.config.Animation.Rotation, -6.28, 6.28, 0.1)
		} else {
			startRot := float64(s.config.Animation.Rotation.Start)
			endRot := float64(s.config.Animation.Rotation.End)
			s.sliderControl(ctx, "Start##rot", &startRot, -6.28, 6.28, 0.1)
			s.sliderControl(ctx, "End##rot", &endRot, -6.28, 6.28, 0.1)
			s.config.Animation.Rotation.Start = float32(startRot)
			s.config.Animation.Rotation.End = float32(endRot)
			ctx.Text("  Easing: " + s.config.Animation.Rotation.Easing)
			ctx.Button("  Cycle Easing##rot").On(func() {
				s.config.Animation.Rotation.Easing = s.cycleEasing(s.config.Animation.Rotation.Easing)
				s.recreateParticles()
			})
		}

		ctx.Text("----------------")

		// Color
		ctx.Text("Color")
		// Initialize color config if nil
		if s.config.Animation.Color == nil {
			ctx.Button("Enable Color").On(func() {
				s.config.Animation.Color = &chirashi.ColorConfig{
					StartR: 1.0, StartG: 1.0, StartB: 1.0,
					EndR: 1.0, EndG: 0.2, EndB: 0.0,
					Easing: "Linear",
				}
				s.recreateParticles()
			})
		} else {
			ctx.Button("Disable Color").On(func() {
				s.config.Animation.Color = nil
				s.recreateParticles()
			})
			startR := float64(s.config.Animation.Color.StartR)
			startG := float64(s.config.Animation.Color.StartG)
			startB := float64(s.config.Animation.Color.StartB)
			endR := float64(s.config.Animation.Color.EndR)
			endG := float64(s.config.Animation.Color.EndG)
			endB := float64(s.config.Animation.Color.EndB)
			s.sliderControl(ctx, "Start R", &startR, 0.0, 1.0, 0.05)
			s.sliderControl(ctx, "Start G", &startG, 0.0, 1.0, 0.05)
			s.sliderControl(ctx, "Start B", &startB, 0.0, 1.0, 0.05)
			s.sliderControl(ctx, "End R", &endR, 0.0, 1.0, 0.05)
			s.sliderControl(ctx, "End G", &endG, 0.0, 1.0, 0.05)
			s.sliderControl(ctx, "End B", &endB, 0.0, 1.0, 0.05)
			s.config.Animation.Color.StartR = float32(startR)
			s.config.Animation.Color.StartG = float32(startG)
			s.config.Animation.Color.StartB = float32(startB)
			s.config.Animation.Color.EndR = float32(endR)
			s.config.Animation.Color.EndG = float32(endG)
			s.config.Animation.Color.EndB = float32(endB)
			ctx.Text("  Easing: " + s.config.Animation.Color.Easing)
			ctx.Button("  Cycle Easing##color").On(func() {
				s.config.Animation.Color.Easing = s.cycleEasing(s.config.Animation.Color.Easing)
				s.recreateParticles()
			})
		}
	})
}

func (s *ParticleEditorScene) drawDebugWindow(ctx *debugui.Context) {
	ctx.Window("Debug Info", image.Rect(420, 10, 720, 200), func(layout debugui.ContainerLayout) {
		fps := ebiten.ActualFPS()
		ctx.Text(fmt.Sprintf("FPS: %.2f", fps))

		// Collect metrics from all particle systems
		var activeCount, totalSpawned, totalDeactivated int
		var updateTimeUs, drawTimeUs int64
		query := donburi.NewQuery(filter.Contains(chirashi.Component))
		query.Each(s.world, func(entry *donburi.Entry) {
			data := chirashi.Component.Get(entry)
			activeCount += data.ActiveCount
			totalSpawned += data.Metrics.SpawnCount
			totalDeactivated += data.Metrics.DeactivateCount
			updateTimeUs += data.Metrics.UpdateTimeUs
			drawTimeUs += data.Metrics.DrawTimeUs
		})

		ctx.Text(fmt.Sprintf("Active: %d", activeCount))
		ctx.Text(fmt.Sprintf("Spawned: %d", totalSpawned))
		ctx.Text(fmt.Sprintf("Deactivated: %d", totalDeactivated))
		ctx.Text(fmt.Sprintf("Update: %d us", updateTimeUs))
		ctx.Text(fmt.Sprintf("Draw: %d us", drawTimeUs))
		ctx.Text(fmt.Sprintf("Total: %.2f ms", float64(updateTimeUs+drawTimeUs)/1000.0))
		ctx.Text("GPU Batch: 1 draw call")
	})
}

func (s *ParticleEditorScene) drawFileWindow(ctx *debugui.Context) {
	ctx.Window("File Operations", image.Rect(420, 210, 720, 450), func(layout debugui.ContainerLayout) {
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
			s.config.Name = newName
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

		ctx.Button("Refresh List").On(func() {
			s.refreshFileList()
		})

		for idx, f := range s.fileList {
			name := filepath.Base(f)
			filePath := f
			ctx.IDScope(fmt.Sprintf("file_%d", idx), func() {
				ctx.Button(name).On(func() {
					log.Printf("Button clicked: file=%s", name)
					chirashi.GetConfigLoader().ClearCache()

					cfg, err := chirashi.GetConfigLoader().LoadConfig(filePath)
					if err != nil {
						log.Println("Load error:", err)
					} else {
						s.config = cfg
						s.recreateParticles()
						log.Printf("Loaded %s", name)
					}
				})
			})
		}
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

func (s *ParticleEditorScene) sliderControl(ctx *debugui.Context, label string, value *float64, min, max, step float64) {
	ctx.IDScope(label, func() {
		ctx.Text(fmt.Sprintf("  %s: %.2f", label, *value))

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

		ctx.SetGridLayout([]int{-1}, nil)
	})
}

func (s *ParticleEditorScene) rangeControl(ctx *debugui.Context, label string, r *chirashi.RangeFloat, min, max, step float64) {
	if r == nil {
		return
	}
	ctx.IDScope(label, func() {
		minVal := float64(r.Min)
		maxVal := float64(r.Max)

		ctx.Text(fmt.Sprintf("  %s Min: %.1f", label, minVal))
		ctx.SetGridLayout([]int{30, -1, 30}, nil)
		ctx.Button("-##min").On(func() {
			r.Min -= float32(step)
			s.recreateParticles()
		})
		ctx.SliderF(&minVal, min, max, step, 1).On(func() {
			r.Min = float32(minVal)
			s.recreateParticles()
		})
		ctx.Button("+##min").On(func() {
			r.Min += float32(step)
			s.recreateParticles()
		})
		ctx.SetGridLayout([]int{-1}, nil)

		ctx.Text(fmt.Sprintf("  %s Max: %.1f", label, maxVal))
		ctx.SetGridLayout([]int{30, -1, 30}, nil)
		ctx.Button("-##max").On(func() {
			r.Max -= float32(step)
			s.recreateParticles()
		})
		ctx.SliderF(&maxVal, min, max, step, 1).On(func() {
			r.Max = float32(maxVal)
			s.recreateParticles()
		})
		ctx.Button("+##max").On(func() {
			r.Max += float32(step)
			s.recreateParticles()
		})
		ctx.SetGridLayout([]int{-1}, nil)
	})
}

func (s *ParticleEditorScene) sequenceControls(ctx *debugui.Context, label string, config *chirashi.PropertyConfig, min, max, stepVal float64) {
	ctx.IDScope(label+"_seq", func() {
		ctx.SetGridLayout([]int{-1}, nil)

		for i := range config.Steps {
			ctx.IDScope(fmt.Sprintf("Step_%d", i), func() {
				step := &config.Steps[i]

				ctx.Text(fmt.Sprintf("Step %d", i+1))

				// Random Start Toggle
				isRandomStart := step.FromRange != nil
				ctx.Button(fmt.Sprintf("Random Start: %v", isRandomStart)).On(func() {
					if isRandomStart {
						step.FromRange = nil
					} else {
						step.FromRange = &chirashi.RangeFloat{Min: step.From, Max: step.From}
					}
					s.recreateParticles()
				})

				if step.FromRange != nil {
					fromMin := float64(step.FromRange.Min)
					fromMax := float64(step.FromRange.Max)
					s.sliderControl(ctx, "From Min", &fromMin, min, max, stepVal)
					s.sliderControl(ctx, "From Max", &fromMax, min, max, stepVal)
					step.FromRange.Min = float32(fromMin)
					step.FromRange.Max = float32(fromMax)
				} else {
					fromVal := float64(step.From)
					s.sliderControl(ctx, "From", &fromVal, min, max, stepVal)
					step.From = float32(fromVal)
				}

				// Random Target Toggle
				isRandom := step.ToRange != nil
				ctx.Button(fmt.Sprintf("Random Target: %v", isRandom)).On(func() {
					if isRandom {
						step.ToRange = nil
					} else {
						step.ToRange = &chirashi.RangeFloat{Min: step.To, Max: step.To}
					}
					s.recreateParticles()
				})

				if step.ToRange != nil {
					toMin := float64(step.ToRange.Min)
					toMax := float64(step.ToRange.Max)
					s.sliderControl(ctx, "To Min", &toMin, min, max, stepVal)
					s.sliderControl(ctx, "To Max", &toMax, min, max, stepVal)
					step.ToRange.Min = float32(toMin)
					step.ToRange.Max = float32(toMax)
				} else {
					toVal := float64(step.To)
					s.sliderControl(ctx, "To", &toVal, min, max, stepVal)
					step.To = float32(toVal)
				}

				durVal := float64(step.Duration)
				s.sliderControl(ctx, "Duration", &durVal, 0.01, 10.0, 0.1)
				step.Duration = float32(durVal)

				ctx.Text("  Easing: " + step.Easing)
				ctx.Button("  Cycle Easing").On(func() {
					step.Easing = s.cycleEasing(step.Easing)
					s.recreateParticles()
				})

				ctx.Button("  Remove Step").On(func() {
					config.Steps = append(config.Steps[:i], config.Steps[i+1:]...)
					if len(config.Steps) == 0 {
						config.Type = ""
					}
					s.recreateParticles()
				})

				ctx.Text("----------------")
			})
		}

		ctx.Button("Add Step").On(func() {
			newStep := chirashi.StepConfig{Duration: 1.0, Easing: "Linear"}
			if len(config.Steps) > 0 {
				lastStep := config.Steps[len(config.Steps)-1]
				newStep.From = lastStep.To
				newStep.To = lastStep.To
			}
			config.Steps = append(config.Steps, newStep)
			s.recreateParticles()
		})
	})
}

func (s *ParticleEditorScene) propertyModeToggle(ctx *debugui.Context, label string, config *chirashi.PropertyConfig) {
	isSeq := config.IsSequence()
	modeLabel := "Simple"
	if isSeq {
		modeLabel = "Sequence"
	}
	ctx.Button(fmt.Sprintf("%s Mode: %s", label, modeLabel)).On(func() {
		if isSeq {
			config.Type = ""
			config.Steps = nil
		} else {
			config.Type = "sequence"
			config.Steps = []chirashi.StepConfig{
				{From: config.Start, To: config.End, Duration: 1.0, Easing: config.Easing},
			}
		}
		s.recreateParticles()
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
	}

	for i, e := range easings {
		if e == current {
			return easings[(i+1)%len(easings)]
		}
	}
	return "Linear"
}
