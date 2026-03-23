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
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"

	"github.com/mogeta/chirashi/assets"
	"github.com/mogeta/chirashi/component/chirashi"
)

const (
	editorScreenWidth  = 1920
	editorScreenHeight = 1080
	editorCenterX      = editorScreenWidth / 2
	editorCenterY      = editorScreenHeight / 2
)

type applyMode int

const (
	applyModeLive applyMode = iota
	applyModeRecreate
)

type ParticleEditorScene struct {
	world           donburi.World
	container       *ecs.ECS
	config          *chirashi.ParticleConfig
	loader          *chirashi.ConfigLoader
	img             *ebiten.Image
	debugui         debugui.DebugUI
	defaultShader   *ebiten.Shader
	shader          *ebiten.Shader
	blurShader      *ebiten.Shader
	bloomShader     *ebiten.Shader
	offscreen       *ebiten.Image
	glitchIntensity float64
	useBlurShader   bool
	vsyncEnabled    bool
	dragEmitter     bool
	time            float64
	fileList        []string
	attractorX      float32
	attractorY      float32
}

func NewParticleEditorScene() (*ParticleEditorScene, error) {
	world := donburi.NewWorld()
	container := ecs.NewECS(world)

	particleSys := chirashi.NewSystem()
	container.AddSystem(particleSys.Update)
	container.AddRenderer(0, particleSys.Draw)

	img := ebiten.NewImage(8, 8)
	img.Fill(color.White)

	shader, err := ebiten.NewShader(assets.ParticleShader)
	if err != nil {
		return nil, fmt.Errorf("load particle shader: %w", err)
	}

	bloomShader, err := ebiten.NewShader(assets.BloomShader)
	if err != nil {
		return nil, fmt.Errorf("load bloom shader: %w", err)
	}

	blurShader, err := ebiten.NewShader(assets.ParticleShaderBlur)
	if err != nil {
		return nil, fmt.Errorf("load blur shader: %w", err)
	}

	loader := chirashi.NewConfigLoader()

	config, err := loader.LoadConfigFromBytes(assets.SampleParticleConfig, "sample.yaml")
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if err := chirashi.NewParticlesFromConfig(world, shader, img, config, editorCenterX, editorCenterY); err != nil {
		return nil, fmt.Errorf("create particles: %w", err)
	}

	scene := &ParticleEditorScene{
		world:         world,
		container:     container,
		config:        config,
		loader:        loader,
		img:           img,
		defaultShader: shader,
		shader:        shader,
		blurShader:    blurShader,
		bloomShader:   bloomShader,
		vsyncEnabled:  false,
		attractorX:    editorCenterX,
		attractorY:    editorCenterY,
	}
	scene.refreshFileList()
	return scene, nil
}

func (s *ParticleEditorScene) Update() error {
	s.time += 1.0 / 60.0

	// Update DebugUI
	if _, err := s.debugui.Update(func(ctx *debugui.Context) error {
		s.drawLeftSidebarWindow(ctx)
		s.drawRightSidebarWindow(ctx)
		return nil
	}); err != nil {
		return err
	}

	if s.config.Animation.Position.Type == "attractor" && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		s.attractorX = float32(x)
		s.attractorY = float32(y)
		s.applyAttractorTarget()
	}
	if s.dragEmitter && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		s.updateEmitterFromCursor()
	}

	s.container.Update()
	return nil
}

func (s *ParticleEditorScene) drawLeftSidebarWindow(ctx *debugui.Context) {
	ctx.Window("Left Sidebar", image.Rect(20, 20, 380, 1060), func(layout debugui.ContainerLayout) {
		ctx.Header("General", false, func() {
			s.drawGeneralSettingsContents(ctx)
		})
		ctx.Header("Motion", false, func() {
			s.drawMotionContents(ctx)
		})
		ctx.Header("Files", false, func() {
			s.drawFileContents(ctx)
		})
	})
}

func (s *ParticleEditorScene) drawRightSidebarWindow(ctx *debugui.Context) {
	ctx.Window("Right Sidebar", image.Rect(1540, 20, 1900, 1060), func(layout debugui.ContainerLayout) {
		ctx.Header("Debug", false, func() {
			s.drawDebugContents(ctx)
		})
		ctx.Header("Alpha", false, func() {
			s.drawPropertyWindow(ctx, "Alpha", &s.config.Animation.Alpha, 0.0, 1.0, 0.05, "alpha")
		})
		ctx.Header("Scale", false, func() {
			s.drawPropertyWindow(ctx, "Scale", &s.config.Animation.Scale, 0.0, 5.0, 0.1, "scale")
		})
		ctx.Header("Rotation", false, func() {
			s.drawPropertyWindow(ctx, "Rotation", &s.config.Animation.Rotation, -6.28, 6.28, 0.1, "rotation")
		})
		if s.config.Animation.Color != nil {
			ctx.Header("Color", false, func() {
				s.drawColorControls(ctx)
			})
		}
	})
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
	log.Printf("Creating particles with config: %s, PosType=%s", s.config.Name, s.config.Animation.Position.Type)
	if err := chirashi.NewParticlesFromConfig(s.world, s.shader, s.img, s.config, editorCenterX, editorCenterY); err != nil {
		log.Println("Failed to recreate particles:", err)
		return
	}
	log.Println("Particles recreated successfully")

	// Re-apply attractor target when in attractor mode
	if s.config.Animation.Position.Type == "attractor" {
		query2 := donburi.NewQuery(filter.Contains(chirashi.Component))
		query2.Each(s.world, func(entry *donburi.Entry) {
			data := chirashi.Component.Get(entry)
			data.AttractorX = s.attractorX
			data.AttractorY = s.attractorY
		})
	}
}

func (s *ParticleEditorScene) applyConfigLive() {
	s.forEachParticleSystem(func(entry *donburi.Entry) {
		chirashi.ApplyConfigLive(s.world, entry.Entity(), s.config, editorCenterX, editorCenterY)
	})

	if s.config.Animation.Position.Type == "attractor" {
		s.applyAttractorTarget()
	}
}

func (s *ParticleEditorScene) applyChange(mode applyMode) {
	if mode == applyModeRecreate {
		s.recreateParticles()
		return
	}
	s.applyConfigLive()
}

func (s *ParticleEditorScene) forEachParticleSystem(fn func(entry *donburi.Entry)) {
	query := donburi.NewQuery(filter.Contains(chirashi.Component))
	query.Each(s.world, fn)
}

func (s *ParticleEditorScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	if s.offscreen != nil && (s.offscreen.Bounds().Dx() != outsideWidth || s.offscreen.Bounds().Dy() != outsideHeight) {
		s.offscreen.Deallocate()
		s.offscreen = nil
	}
	return editorScreenWidth, editorScreenHeight
}

func (s *ParticleEditorScene) drawGeneralSettingsContents(ctx *debugui.Context) {
	s.drawSpawnControls(ctx)
	ctx.Text("----------------")
	s.drawEmitterControls(ctx)
	ctx.Text("----------------")
	s.drawVisualFeatureControls(ctx)
	ctx.Text("----------------")
	s.drawShaderControls(ctx)
}

func (s *ParticleEditorScene) drawSpawnControls(ctx *debugui.Context) {
	ctx.Text("Spawn Config")

	ctx.SetGridLayout([]int{140, 60, 60}, nil)
	ctx.Text(fmt.Sprintf("Interval: %d", s.config.Spawn.Interval))
	ctx.Button("I+").On(func() { s.config.Spawn.Interval++; s.applyChange(applyModeLive) })
	ctx.Button("I-").On(func() {
		if s.config.Spawn.Interval > 1 {
			s.config.Spawn.Interval--
		}
		s.applyChange(applyModeLive)
	})

	ctx.Text(fmt.Sprintf("Per Spawn: %d", s.config.Spawn.ParticlesPerSpawn))
	ctx.Button("P+").On(func() { s.config.Spawn.ParticlesPerSpawn += 10; s.applyChange(applyModeLive) })
	ctx.Button("P-").On(func() {
		if s.config.Spawn.ParticlesPerSpawn > 1 {
			s.config.Spawn.ParticlesPerSpawn -= 10
		}
		s.applyChange(applyModeLive)
	})

	ctx.Text(fmt.Sprintf("Max Particles: %d", s.config.Spawn.MaxParticles))
	ctx.Button("M+").On(func() { s.config.Spawn.MaxParticles += 1000; s.applyChange(applyModeRecreate) })
	ctx.Button("M-").On(func() {
		if s.config.Spawn.MaxParticles > 100 {
			s.config.Spawn.MaxParticles -= 1000
		}
		s.applyChange(applyModeRecreate)
	})
	ctx.SetGridLayout([]int{-1}, nil)

	loopLabel := "Loop: ON"
	if !s.config.Spawn.IsLoop {
		loopLabel = "Loop: OFF"
	}
	ctx.Button(loopLabel).On(func() {
		s.config.Spawn.IsLoop = !s.config.Spawn.IsLoop
		s.applyChange(applyModeLive)
	})

	if !s.config.Spawn.IsLoop {
		ctx.SetGridLayout([]int{140, 60, 60}, nil)
		ctx.Text(fmt.Sprintf("LifeTime: %d", s.config.Spawn.LifeTime))
		ctx.Button("L+").On(func() { s.config.Spawn.LifeTime += 60; s.applyChange(applyModeLive) })
		ctx.Button("L-").On(func() {
			if s.config.Spawn.LifeTime > 60 {
				s.config.Spawn.LifeTime -= 60
			}
			s.applyChange(applyModeLive)
		})
		ctx.SetGridLayout([]int{-1}, nil)
	}

}

func (s *ParticleEditorScene) drawEmitterControls(ctx *debugui.Context) {
	ctx.Text("Emitter")
	ctx.SetGridLayout([]int{140, 60, 60}, nil)
	ctx.Text(fmt.Sprintf("X: %.1f", s.config.Emitter.X))
	ctx.Button("X+").On(func() { s.config.Emitter.X += 10; s.applyChange(applyModeLive) })
	ctx.Button("X-").On(func() { s.config.Emitter.X -= 10; s.applyChange(applyModeLive) })

	ctx.Text(fmt.Sprintf("Y: %.1f", s.config.Emitter.Y))
	ctx.Button("Y+").On(func() { s.config.Emitter.Y += 10; s.applyChange(applyModeLive) })
	ctx.Button("Y-").On(func() { s.config.Emitter.Y -= 10; s.applyChange(applyModeLive) })
	ctx.SetGridLayout([]int{-1}, nil)

	space := s.config.Emitter.Space
	if space == chirashi.EmitterSpaceDefault {
		space = chirashi.EmitterSpaceLocal
	}
	ctx.SetGridLayout([]int{180, 180}, nil)
	ctx.Text("Particle Space: " + string(space))
	ctx.Button("Toggle Space").On(func() {
		if s.config.Emitter.Space == chirashi.EmitterSpaceWorld {
			s.config.Emitter.Space = chirashi.EmitterSpaceLocal
		} else {
			s.config.Emitter.Space = chirashi.EmitterSpaceWorld
		}
		s.applyChange(applyModeLive)
	})
	ctx.SetGridLayout([]int{-1}, nil)

	s.drawEmitterShapeControls(ctx)
}

func (s *ParticleEditorScene) drawVisualFeatureControls(ctx *debugui.Context) {
	ctx.Text("Visual Features")
	ctx.SetGridLayout([]int{180, 180}, nil)
	colorState := "Color Window: OFF"
	colorButton := "Enable Color"
	if s.config.Animation.Color != nil {
		colorState = "Color Window: ON"
		colorButton = "Disable Color"
	}
	ctx.Text(colorState)
	ctx.Button(colorButton).On(func() {
		if s.config.Animation.Color == nil {
			s.config.Animation.Color = &chirashi.ColorConfig{
				StartR: 1.0, StartG: 1.0, StartB: 1.0,
				EndR: 1.0, EndG: 0.2, EndB: 0.0,
				Easing: "Linear",
			}
		} else {
			s.config.Animation.Color = nil
		}
		s.applyChange(applyModeLive)
	})
	ctx.SetGridLayout([]int{-1}, nil)

	dragLabel := "Drag Emitter: OFF"
	if s.dragEmitter {
		dragLabel = "Drag Emitter: ON"
	}
	ctx.Button(dragLabel).On(func() {
		s.dragEmitter = !s.dragEmitter
	})
}

func (s *ParticleEditorScene) drawShaderControls(ctx *debugui.Context) {
	s.sliderControl(ctx, "Glitch Intensity", &s.glitchIntensity, 0.0, 1.0, 0.01)

	vsyncLabel := "VSync: OFF"
	if s.vsyncEnabled {
		vsyncLabel = "VSync: ON"
	}
	ctx.Button(vsyncLabel).On(func() {
		s.vsyncEnabled = !s.vsyncEnabled
		ebiten.SetVsyncEnabled(s.vsyncEnabled)
	})

	mode := "Default"
	if s.useBlurShader {
		mode = "Blur"
	}
	ctx.SetGridLayout([]int{180, 180}, nil)
	ctx.Text("Particle Shader: " + mode)
	ctx.Button("Toggle Blur").On(func() {
		s.useBlurShader = !s.useBlurShader
		if s.useBlurShader {
			s.shader = s.blurShader
		} else {
			s.shader = s.defaultShader
		}
		s.applyChange(applyModeRecreate)
	})
	ctx.SetGridLayout([]int{-1}, nil)
}

func (s *ParticleEditorScene) drawEmitterShapeControls(ctx *debugui.Context) {
	shape := &s.config.Emitter.Shape
	shapeType := shape.Type
	if shapeType == "" {
		shapeType = "point"
	}

	ctx.Text("Emitter Shape: " + shapeType)
	ctx.SetGridLayout([]int{90, 90, 90, 90}, nil)
	ctx.Button("Point").On(func() { s.setEmitterShapeType("point") })
	ctx.Button("Circle").On(func() { s.setEmitterShapeType("circle") })
	ctx.Button("Box").On(func() { s.setEmitterShapeType("box") })
	ctx.Button("Line").On(func() { s.setEmitterShapeType("line") })
	ctx.SetGridLayout([]int{-1}, nil)

	switch shapeType {
	case "circle":
		if shape.Radius == nil {
			shape.Radius = &chirashi.RangeFloat{Min: 0, Max: 100}
		}
		s.rangeControl(ctx, "Radius", shape.Radius, 0, 300, 5)
		s.sliderControl32(ctx, "Start Angle", &shape.StartAngle, -6.2831855, 6.2831855, 0.1)
		s.sliderControl32(ctx, "End Angle", &shape.EndAngle, -6.2831855, 6.2831855, 0.1)
		ctx.Button("Full Arc").On(func() {
			shape.StartAngle = 0
			shape.EndAngle = 6.2831855
			s.applyChange(applyModeLive)
		})
		edgeLabel := "Area"
		if shape.FromEdge {
			edgeLabel = "Edge"
		}
		ctx.Button("Sample: " + edgeLabel).On(func() {
			shape.FromEdge = !shape.FromEdge
			s.applyChange(applyModeRecreate)
		})
	case "box":
		s.sliderControl32(ctx, "Width", &shape.Width, 0, 400, 5)
		s.sliderControl32(ctx, "Height", &shape.Height, 0, 400, 5)
		s.sliderControl32(ctx, "Rotation##shape", &shape.Rotation, -3.14, 3.14, 0.1)
		boxSample := "Area"
		if shape.FromEdge {
			boxSample = "Edge"
		}
		ctx.Button("Sample: " + boxSample).On(func() {
			shape.FromEdge = !shape.FromEdge
			s.applyChange(applyModeRecreate)
		})
	case "line":
		s.sliderControl32(ctx, "Length", &shape.Length, 0, 400, 5)
		s.sliderControl32(ctx, "Rotation##line", &shape.Rotation, -3.14, 3.14, 0.1)
	default:
		ctx.Text("Single point emission")
	}
}

func (s *ParticleEditorScene) setEmitterShapeType(shapeType string) {
	shape := &s.config.Emitter.Shape
	shape.Type = shapeType
	switch shapeType {
	case "circle":
		if shape.Radius == nil {
			shape.Radius = &chirashi.RangeFloat{Min: 0, Max: 100}
		}
		if shape.EndAngle == 0 {
			shape.EndAngle = 6.2831855
		}
	case "box":
		if shape.Width == 0 {
			shape.Width = 160
		}
		if shape.Height == 0 {
			shape.Height = 80
		}
	case "line":
		if shape.Length == 0 {
			shape.Length = 200
		}
	default:
		shape.Type = "point"
	}
	s.applyChange(applyModeRecreate)
}

func (s *ParticleEditorScene) setPositionMode(mode string) {
	if mode == "" {
		mode = "cartesian"
	}
	s.config.Animation.Position.Type = mode
	switch mode {
	case "polar":
		if s.config.Animation.Position.Angle == nil {
			s.config.Animation.Position.Angle = &chirashi.RangeFloat{Min: 0, Max: 6.283185}
		}
		if s.config.Animation.Position.Distance == nil {
			s.config.Animation.Position.Distance = &chirashi.RangeFloat{Min: 50, Max: 150}
		}
	case "attractor":
		if s.config.Animation.Position.ControlX == nil {
			s.config.Animation.Position.ControlX = &chirashi.RangeFloat{Min: -100, Max: 100}
		}
		if s.config.Animation.Position.ControlY == nil {
			s.config.Animation.Position.ControlY = &chirashi.RangeFloat{Min: -200, Max: -50}
		}
	default:
		s.config.Animation.Position.Type = "cartesian"
		if s.config.Animation.Position.EndX == nil {
			s.config.Animation.Position.EndX = &chirashi.RangeFloat{Min: -100, Max: 100}
		}
		if s.config.Animation.Position.EndY == nil {
			s.config.Animation.Position.EndY = &chirashi.RangeFloat{Min: -100, Max: 100}
		}
	}
	s.applyChange(applyModeRecreate)
}

func (s *ParticleEditorScene) drawMotionContents(ctx *debugui.Context) {
	s.drawDurationSection(ctx)
	ctx.Text("----------------")
	s.drawPositionSection(ctx)
	ctx.Text("----------------")
	s.drawFlowSection(ctx)
}

func (s *ParticleEditorScene) drawDurationSection(ctx *debugui.Context) {
	if s.config.Animation.Duration.Range != nil {
		ctx.SetGridLayout([]int{180, 180}, nil)
		ctx.Text("Duration")
		ctx.Button("Mode: Range").On(func() {
			s.config.Animation.Duration.Range = nil
			if s.config.Animation.Duration.Value <= 0 {
				s.config.Animation.Duration.Value = 1.0
			}
			s.applyChange(applyModeLive)
		})
		ctx.SetGridLayout([]int{-1}, nil)
		s.rangeControlCompact(ctx, "Duration", s.config.Animation.Duration.Range, 0.1, 10.0, 0.1)
		return
	}

	ctx.SetGridLayout([]int{180, 180}, nil)
	ctx.Text("Duration")
	ctx.Button("Mode: Fixed").On(func() {
		base := s.config.Animation.Duration.Value
		if base <= 0 {
			base = 1.0
		}
		minVal := base * 0.5
		if minVal < 0.1 {
			minVal = 0.1
		}
		s.config.Animation.Duration.Range = &chirashi.RangeFloat{Min: minVal, Max: base * 1.5}
		s.applyChange(applyModeLive)
	})
	ctx.SetGridLayout([]int{-1}, nil)

	dv := float64(s.config.Animation.Duration.Value)
	s.sliderControl(ctx, "Value", &dv, 0.1, 10.0, 0.1)
	s.config.Animation.Duration.Value = float32(dv)
}

func (s *ParticleEditorScene) drawPositionSection(ctx *debugui.Context) {
	posType := s.config.Animation.Position.Type
	if posType == "" {
		posType = "cartesian"
	}
	ctx.SetGridLayout([]int{180, 110, 110, 110}, nil)
	ctx.Text("Position: " + posType)
	ctx.Button("Cartesian").On(func() { s.setPositionMode("cartesian") })
	ctx.Button("Polar").On(func() { s.setPositionMode("polar") })
	ctx.Button("Attractor").On(func() { s.setPositionMode("attractor") })
	ctx.SetGridLayout([]int{-1}, nil)

	switch s.config.Animation.Position.Type {
	case "polar":
		s.rangeControl(ctx, "Angle", s.config.Animation.Position.Angle, 0, 6.283185, 0.1)
		s.rangeControl(ctx, "Distance", s.config.Animation.Position.Distance, 0, 500, 10)
	case "attractor":
		s.rangeControl(ctx, "Control X", s.config.Animation.Position.ControlX, -500, 500, 10)
		s.rangeControl(ctx, "Control Y", s.config.Animation.Position.ControlY, -500, 500, 10)
		ctx.Text("Attractor Target")
		ctx.SetGridLayout([]int{180, 60, 60}, nil)
		ctx.Text(fmt.Sprintf("Target X: %.0f", s.attractorX))
		ctx.Button("TX+").On(func() {
			s.attractorX += 10
			s.applyAttractorTarget()
		})
		ctx.Button("TX-").On(func() {
			s.attractorX -= 10
			s.applyAttractorTarget()
		})
		ctx.Text(fmt.Sprintf("Target Y: %.0f", s.attractorY))
		ctx.Button("TY+").On(func() {
			s.attractorY += 10
			s.applyAttractorTarget()
		})
		ctx.Button("TY-").On(func() {
			s.attractorY -= 10
			s.applyAttractorTarget()
		})
		ctx.SetGridLayout([]int{-1}, nil)
	default:
		s.rangeControl(ctx, "End X", s.config.Animation.Position.EndX, -500, 500, 10)
		s.rangeControl(ctx, "End Y", s.config.Animation.Position.EndY, -500, 500, 10)
	}

	ctx.SetGridLayout([]int{200, 180}, nil)
	ctx.Text("Easing: " + s.config.Animation.Position.Easing)
	ctx.Button("Cycle Easing").On(func() {
		s.config.Animation.Position.Easing = s.cycleEasing(s.config.Animation.Position.Easing)
		s.applyChange(applyModeLive)
	})
	ctx.SetGridLayout([]int{-1}, nil)
}

func (s *ParticleEditorScene) drawFlowSection(ctx *debugui.Context) {
	ctx.SetGridLayout([]int{200, 180}, nil)
	if s.config.Animation.Position.Flow == nil {
		ctx.Text("Flow: OFF")
		ctx.Button("Enable Flow").On(func() {
			s.config.Animation.Position.Flow = &chirashi.FlowConfig{
				Type:        "curl",
				Strength:    &chirashi.RangeFloat{Min: 8, Max: 18},
				Scale:       180,
				Octaves:     2,
				Persistence: 0.5,
				TimeScale:   0.2,
				Drag:        0.96,
				Space:       "local",
			}
			s.applyChange(applyModeLive)
		})
		ctx.SetGridLayout([]int{-1}, nil)
		ctx.Text("Enable flow to show curl drift controls.")
		return
	}

	ctx.Text("Flow: ON")
	ctx.Button("Disable Flow").On(func() {
		s.config.Animation.Position.Flow = nil
		s.applyChange(applyModeLive)
	})
	ctx.SetGridLayout([]int{-1}, nil)

	flow := s.config.Animation.Position.Flow
	if flow.Strength == nil {
		flow.Strength = &chirashi.RangeFloat{Min: 8, Max: 18}
	}
	ctx.Text("Type: curl")
	s.rangeControl(ctx, "Flow Strength", flow.Strength, 0, 80, 1)
	s.sliderControl32(ctx, "Flow Scale", &flow.Scale, 32, 480, 4)
	octaves := float64(flow.Octaves)
	s.sliderControl(ctx, "Flow Octaves", &octaves, 1, 3, 1)
	flow.Octaves = int(octaves)
	s.sliderControl32(ctx, "Flow Persistence", &flow.Persistence, 0, 1, 0.05)
	s.sliderControl32(ctx, "Flow Time Scale", &flow.TimeScale, 0, 2, 0.05)
	s.sliderControl32(ctx, "Flow Drag", &flow.Drag, 0.80, 1.0, 0.01)
	s.sliderControl32(ctx, "Flow Bounds", &flow.BoundRadius, 0, 1200, 10)
	ctx.SetGridLayout([]int{200, 180}, nil)
	ctx.Text("Flow Space: " + flow.Space)
	ctx.Button("Toggle Flow Space").On(func() {
		if flow.Space == "world" {
			flow.Space = "local"
		} else {
			flow.Space = "world"
		}
		s.applyChange(applyModeLive)
	})
	respawnLabel := "Respawn On Escape: OFF"
	if flow.RespawnOnEscape {
		respawnLabel = "Respawn On Escape: ON"
	}
	ctx.Button(respawnLabel).On(func() {
		flow.RespawnOnEscape = !flow.RespawnOnEscape
		s.applyChange(applyModeLive)
	})
	ctx.SetGridLayout([]int{-1}, nil)
}

func (s *ParticleEditorScene) drawPropertyWindow(ctx *debugui.Context, label string, config *chirashi.PropertyConfig, min, max, step float64, idSuffix string) {
	s.propertyModeToggle(ctx, label, config)
	if config.IsSequence() {
		s.sequenceControls(ctx, label, config, min, max, step)
		return
	}

	start := float64(config.Start)
	end := float64(config.End)
	s.sliderControl(ctx, "Start##"+idSuffix, &start, min, max, step)
	s.sliderControl(ctx, "End##"+idSuffix, &end, min, max, step)
	config.Start = float32(start)
	config.End = float32(end)
	ctx.SetGridLayout([]int{200, 180}, nil)
	ctx.Text("Easing: " + config.Easing)
	ctx.Button("Cycle Easing##" + idSuffix).On(func() {
		config.Easing = s.cycleEasing(config.Easing)
		s.applyChange(applyModeLive)
	})
	ctx.SetGridLayout([]int{-1}, nil)
}

func (s *ParticleEditorScene) drawColorControls(ctx *debugui.Context) {
	if s.config.Animation.Color == nil {
		return
	}

	ctx.Button("Disable Color").On(func() {
		s.config.Animation.Color = nil
		s.applyChange(applyModeLive)
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
	ctx.SetGridLayout([]int{200, 180}, nil)
	ctx.Text("Easing: " + s.config.Animation.Color.Easing)
	ctx.Button("Cycle Easing##color").On(func() {
		s.config.Animation.Color.Easing = s.cycleEasing(s.config.Animation.Color.Easing)
		s.applyChange(applyModeLive)
	})
	ctx.SetGridLayout([]int{-1}, nil)
}

func (s *ParticleEditorScene) drawDebugContents(ctx *debugui.Context) {
	fps := ebiten.ActualFPS()
	tps := ebiten.ActualTPS()
	ctx.Text(fmt.Sprintf("FPS: %.2f", fps))
	ctx.Text(fmt.Sprintf("TPS: %.2f", tps))
	ctx.Text(fmt.Sprintf("VSync: %v", s.vsyncEnabled))

	// Collect metrics from all particle systems
	var activeCount, totalSpawned, totalDeactivated int
	var updateTimeUs, drawTimeUs int64
	s.forEachParticleSystem(func(entry *donburi.Entry) {
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
}

func (s *ParticleEditorScene) drawFileContents(ctx *debugui.Context) {
	// Save
	ctx.Button("Save " + s.config.Name + ".yaml").On(func() {
		path := filepath.Join("assets", "particles", s.config.Name+".yaml")
		err := s.loader.SaveConfig(path, s.config)
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
		err := s.loader.SaveConfig(path, s.config)
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
				s.loader.ClearCache()

				cfg, err := s.loader.LoadConfig(filePath)
				if err != nil {
					log.Println("Load error:", err)
				} else {
					s.config = cfg
					// Reset attractor target to screen center for new configs
					s.attractorX = editorCenterX
					s.attractorY = editorCenterY
					s.applyChange(applyModeRecreate)
					log.Printf("Loaded %s", name)
				}
			})
		})
	}
}

func (s *ParticleEditorScene) applyAttractorTarget() {
	s.forEachParticleSystem(func(entry *donburi.Entry) {
		data := chirashi.Component.Get(entry)
		data.AttractorX = s.attractorX
		data.AttractorY = s.attractorY
	})
}

func (s *ParticleEditorScene) updateEmitterFromCursor() {
	x, y := ebiten.CursorPosition()
	nextX := float32(x - editorCenterX)
	nextY := float32(y - editorCenterY)
	if s.config.Emitter.X == nextX && s.config.Emitter.Y == nextY {
		return
	}
	s.config.Emitter.X = nextX
	s.config.Emitter.Y = nextY
	s.applyChange(applyModeLive)
}

func (s *ParticleEditorScene) refreshFileList() {
	files, err := s.loader.ListConfigs("assets/particles/*.yaml")
	if err != nil {
		log.Println("Failed to list files:", err)
		return
	}
	s.fileList = files
}

func (s *ParticleEditorScene) sliderControl(ctx *debugui.Context, label string, value *float64, min, max, step float64) {
	s.sliderControlWithMode(ctx, label, value, min, max, step, applyModeLive)
}

func (s *ParticleEditorScene) sliderControlWithMode(ctx *debugui.Context, label string, value *float64, min, max, step float64, mode applyMode) {
	s.numericControl(ctx, label, value, min, max, step, 2, mode)
}

func (s *ParticleEditorScene) sliderControl32(ctx *debugui.Context, label string, value *float32, min, max, step float64) {
	ctx.IDScope(label, func() {
		floatVal := float64(*value)
		s.numericControl(ctx, label, &floatVal, min, max, step, 2, applyModeLive)
		*value = float32(floatVal)
	})
}

func (s *ParticleEditorScene) numericControl(ctx *debugui.Context, label string, value *float64, min, max, step float64, precision int, mode applyMode) {
	ctx.IDScope(label, func() {
		ctx.SetGridLayout([]int{140, 30, -1, 30}, nil)
		ctx.Text(fmt.Sprintf("%s: %.*f", label, precision, *value))

		ctx.Button("-").On(func() {
			*value -= step
			if *value < min {
				*value = min
			}
			s.applyChange(mode)
		})

		ctx.SliderF(value, min, max, step, precision).On(func() {
			s.applyChange(mode)
		})

		ctx.Button("+").On(func() {
			*value += step
			if *value > max {
				*value = max
			}
			s.applyChange(mode)
		})

		ctx.SetGridLayout([]int{-1}, nil)
	})
}

func (s *ParticleEditorScene) rangeControl(ctx *debugui.Context, label string, r *chirashi.RangeFloat, min, max, step float64) {
	s.rangeControlWithMode(ctx, label, r, min, max, step, 1, applyModeLive)
}

func (s *ParticleEditorScene) rangeControlCompact(ctx *debugui.Context, label string, r *chirashi.RangeFloat, min, max, step float64) {
	s.rangeControlWithMode(ctx, label+"_compact", r, min, max, step, 2, applyModeLive)
}

func (s *ParticleEditorScene) rangeControlWithMode(ctx *debugui.Context, label string, r *chirashi.RangeFloat, min, max, step float64, precision int, mode applyMode) {
	if r == nil {
		return
	}
	ctx.IDScope(label, func() {
		minVal := float64(r.Min)
		maxVal := float64(r.Max)

		s.rangeBoundControl(ctx, label, "Min", &minVal, min, max, step, precision, mode)
		r.Min = float32(minVal)

		s.rangeBoundControl(ctx, label, "Max", &maxVal, min, max, step, precision, mode)
		r.Max = float32(maxVal)
	})
}

func (s *ParticleEditorScene) rangeBoundControl(ctx *debugui.Context, label, bound string, value *float64, min, max, step float64, precision int, mode applyMode) {
	ctx.IDScope(label+"_"+bound, func() {
		ctx.Text(fmt.Sprintf("%s %s: %.*f", label, bound, precision, *value))
		ctx.SetGridLayout([]int{30, -1, 30}, nil)
		ctx.Button("-##" + bound).On(func() {
			*value -= step
			s.applyChange(mode)
		})
		ctx.SliderF(value, min, max, step, precision).On(func() {
			s.applyChange(mode)
		})
		ctx.Button("+##" + bound).On(func() {
			*value += step
			s.applyChange(mode)
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
					s.applyChange(applyModeLive)
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
					s.applyChange(applyModeLive)
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
					s.applyChange(applyModeLive)
				})

				ctx.Button("  Remove Step").On(func() {
					config.Steps = append(config.Steps[:i], config.Steps[i+1:]...)
					if len(config.Steps) == 0 {
						config.Type = ""
					}
					s.applyChange(applyModeLive)
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
			s.applyChange(applyModeLive)
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
		s.applyChange(applyModeLive)
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
