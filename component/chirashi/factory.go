package chirashi

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/donburi"
)

const fullCircleEpsilon = float32(0.01)

const (
	defaultEmitterVectorCurveSteps = 12
	defaultFlowScale               = float32(160)
	defaultFlowOctaves             = 2
	defaultFlowPersistence         = float32(0.5)
	defaultFlowTimeScale           = float32(0.2)
	defaultFlowDrag                = float32(0.96)
)

var (
	// Global configuration loader instance
	configLoader = NewConfigLoader()
)

// NewParticlesFromConfig creates a GPU particle system from a configuration struct
func NewParticlesFromConfig(w donburi.World, shader *ebiten.Shader, image *ebiten.Image, config *ParticleConfig, x, y float32) error {
	return createParticlesFromConfig(w, shader, image, config, x, y)
}

// NewParticlesFromFile creates a GPU particle system from a configuration file path
func NewParticlesFromFile(w donburi.World, shader *ebiten.Shader, image *ebiten.Image, configPath string, x, y float32) error {
	config, err := configLoader.LoadConfig(configPath)
	if err != nil {
		return err
	}

	return createParticlesFromConfig(w, shader, image, config, x, y)
}

// createParticlesFromConfig creates particles from a loaded configuration
func createParticlesFromConfig(w donburi.World, shader *ebiten.Shader, image *ebiten.Image, config *ParticleConfig, x, y float32) error {
	_, err := createParticleEntityFromConfig(w, shader, image, config, x, y)
	return err
}

func createParticleEntityFromConfig(w donburi.World, shader *ebiten.Shader, image *ebiten.Image, config *ParticleConfig, x, y float32) (donburi.Entity, error) {
	normalizeParticleConfig(config)

	entity := w.Create(Component)
	entry := w.Entry(entity)
	systemData := buildSystemDataFromConfig(shader, image, config, x, y)

	// Apply sequence configurations if present
	buildSequenceConfigs(config, &systemData)

	donburi.SetValue(entry, Component, systemData)
	return entity, nil
}

func buildSystemDataFromConfig(shader *ebiten.Shader, image *ebiten.Image, config *ParticleConfig, x, y float32) SystemData {
	emitterX := x + config.Emitter.X
	emitterY := y + config.Emitter.Y
	animParams := buildAnimationParams(config)

	freeIndices := make([]int, config.Spawn.MaxParticles)
	for i := range freeIndices {
		freeIndices[i] = config.Spawn.MaxParticles - 1 - i
	}

	maxVertices := config.Spawn.MaxParticles * 4
	maxIndices := config.Spawn.MaxParticles * 6

	var imgWidth, imgHeight float32
	if image != nil {
		bounds := image.Bounds()
		imgWidth = float32(bounds.Dx())
		imgHeight = float32(bounds.Dy())
	}

	data := SystemData{
		ParticlePool:      make([]Instance, config.Spawn.MaxParticles),
		ActiveIndices:     make([]int, 0, config.Spawn.MaxParticles),
		FreeIndices:       freeIndices,
		Vertices:          make([]ebiten.Vertex, 0, maxVertices),
		Indices:           make([]uint16, 0, maxIndices),
		Shader:            shader,
		CurrentTime:       0,
		EmitterX:          emitterX,
		EmitterY:          emitterY,
		EmitterShape:      buildEmitterShapeParams(config.Emitter.Shape),
		EmitterVector:     buildEmitterVectorParams(config.Emitter.Vector),
		EmitterLocalSpace: config.Emitter.Space != EmitterSpaceWorld,
		SpawnInterval:     config.Spawn.Interval,
		ParticlesPerSpawn: config.Spawn.ParticlesPerSpawn,
		MaxParticles:      config.Spawn.MaxParticles,
		SourceImage:       image,
		ImageWidth:        imgWidth,
		ImageHeight:       imgHeight,
		Trail:             buildTrailData(config.Trail),
		ActiveCount:       0,
		IsLoop:            config.Spawn.IsLoop,
		LifeTime:          config.Spawn.LifeTime,
		AnimParams:        animParams,
	}
	if data.Trail.Params.Enabled && data.Trail.Params.Mode == "particle" {
		maxTrailVertices := config.Spawn.MaxParticles * data.Trail.Params.MaxPoints * 2
		maxTrailIndices := config.Spawn.MaxParticles * (data.Trail.Params.MaxPoints - 1) * 6
		data.Trail.Runtime.Vertices = make([]ebiten.Vertex, 0, maxTrailVertices)
		data.Trail.Runtime.Indices = make([]uint16, 0, maxTrailIndices)
		for i := range data.ParticlePool {
			data.ParticlePool[i].TrailPoints = make([]TrailPoint, 0, data.Trail.Params.MaxPoints)
		}
	}
	return data
}

func normalizeParticleConfig(config *ParticleConfig) {
	normalizeEmitterShapeConfig(&config.Emitter.Shape)
}

func normalizeEmitterShapeConfig(shape *EmitterShapeConfig) {
	if shape.Type != "circle" {
		return
	}

	tau := float32(2 * math.Pi)
	if shape.StartAngle == 0 && shape.EndAngle == 0 {
		shape.EndAngle = tau
		return
	}

	diff := shape.EndAngle - shape.StartAngle
	if math.Abs(float64(diff-tau)) <= float64(fullCircleEpsilon) ||
		math.Abs(float64(diff+tau)) <= float64(fullCircleEpsilon) {
		shape.EndAngle = shape.StartAngle + tau
	}
}

func buildEmitterShapeParams(config EmitterShapeConfig) EmitterShapeParams {
	shape := EmitterShapeParams{
		Type:       parseEmitterShapeType(config.Type),
		StartAngle: config.StartAngle,
		EndAngle:   config.EndAngle,
		Width:      config.Width,
		Height:     config.Height,
		Length:     config.Length,
		Rotation:   config.Rotation,
		FromEdge:   config.FromEdge,
	}
	if config.Radius != nil {
		shape.RadiusMin = config.Radius.Min
		shape.RadiusMax = config.Radius.Max
	}
	return shape
}

func buildEmitterVectorParams(config *EmitterVectorConfig) EmitterVectorParams {
	if config == nil {
		return EmitterVectorParams{}
	}

	params := EmitterVectorParams{
		Enabled:   true,
		Type:      parseEmitterVectorType(config.Type),
		Placement: parseEmitterVectorPlacement(config.Placement),
	}
	if params.Type == EmitterVectorPolyline && config.Placement == "" {
		params.Placement = EmitterVectorSurface
	}
	if config.Rect != nil {
		params.Rect = EmitterVectorRectParams{
			Width:    config.Rect.Width,
			Height:   config.Rect.Height,
			Rotation: config.Rect.Rotation,
		}
	}
	if config.Polyline != nil {
		params.Polyline = buildEmitterVectorPolylineParams(config.Polyline)
	}
	return params
}

func buildEmitterVectorPolylineParams(config *EmitterVectorPolylineConfig) EmitterVectorPolylineParams {
	params := EmitterVectorPolylineParams{
		Closed:         config.Closed,
		Interpolation:  emitterVectorPolylineInterpolation(config),
		CurveSteps:     emitterVectorPolylineCurveSteps(config),
		SegmentLengths: nil,
	}
	compiledPoints := CompileEmitterVectorPolylinePoints(config)
	params.Points = make([]EmitterVectorPointParams, len(compiledPoints))
	for i, point := range compiledPoints {
		params.Points[i] = EmitterVectorPointParams(point)
	}

	segmentCount := len(params.Points) - 1
	if params.Closed && len(params.Points) > 1 {
		segmentCount = len(params.Points)
	}
	if segmentCount <= 0 {
		params.SegmentLengths = nil
		return params
	}

	params.SegmentLengths = make([]float32, 0, segmentCount)
	for i := 0; i < len(params.Points)-1; i++ {
		length := vectorSegmentLength(params.Points[i], params.Points[i+1])
		params.SegmentLengths = append(params.SegmentLengths, length)
		params.TotalLength += length
	}
	if params.Closed {
		length := vectorSegmentLength(params.Points[len(params.Points)-1], params.Points[0])
		params.SegmentLengths = append(params.SegmentLengths, length)
		params.TotalLength += length
	}
	return params
}

// CompileEmitterVectorPolylinePoints expands a configured polyline into sampled points.
func CompileEmitterVectorPolylinePoints(config *EmitterVectorPolylineConfig) []EmitterVectorPoint {
	if config == nil || len(config.Points) == 0 {
		return nil
	}
	if emitterVectorPolylineInterpolation(config) != "quadratic" {
		points := make([]EmitterVectorPoint, len(config.Points))
		copy(points, config.Points)
		return points
	}

	curveSteps := emitterVectorPolylineCurveSteps(config)
	if curveSteps < 1 {
		curveSteps = defaultEmitterVectorCurveSteps
	}
	points := make([]EmitterVectorPoint, 0, ((len(config.Points)-1)/2)*curveSteps+1)
	start := config.Points[0]
	points = append(points, start)
	for i := 0; i+2 < len(config.Points); i += 2 {
		a := EmitterVectorPointParams(config.Points[i])
		control := EmitterVectorPointParams(config.Points[i+1])
		b := EmitterVectorPointParams(config.Points[i+2])
		for step := 1; step <= curveSteps; step++ {
			t := float32(step) / float32(curveSteps)
			points = append(points, EmitterVectorPoint(quadraticBezierPoint(a, control, b, t)))
		}
	}
	return points
}

func emitterVectorPolylineInterpolation(config *EmitterVectorPolylineConfig) string {
	if config == nil || config.Interpolation == "" {
		return "linear"
	}
	return config.Interpolation
}

func emitterVectorPolylineCurveSteps(config *EmitterVectorPolylineConfig) int {
	if config == nil || config.CurveSteps <= 0 {
		return defaultEmitterVectorCurveSteps
	}
	return config.CurveSteps
}

func quadraticBezierPoint(a, control, b EmitterVectorPointParams, t float32) EmitterVectorPointParams {
	u := 1 - t
	return EmitterVectorPointParams{
		X: u*u*a.X + 2*u*t*control.X + t*t*b.X,
		Y: u*u*a.Y + 2*u*t*control.Y + t*t*b.Y,
	}
}

func vectorSegmentLength(a, b EmitterVectorPointParams) float32 {
	dx := b.X - a.X
	dy := b.Y - a.Y
	return float32(math.Hypot(float64(dx), float64(dy)))
}

func parseEmitterShapeType(shapeType string) EmitterShapeType {
	switch shapeType {
	case "", "point":
		return EmitterShapePoint
	case "circle":
		return EmitterShapeCircle
	case "box":
		return EmitterShapeBox
	case "line":
		return EmitterShapeLine
	default:
		return EmitterShapePoint
	}
}

func parseEmitterVectorType(vectorType string) EmitterVectorType {
	switch vectorType {
	case "rect":
		return EmitterVectorRect
	case "polyline":
		return EmitterVectorPolyline
	default:
		return EmitterVectorNone
	}
}

func parseEmitterVectorPlacement(placement string) EmitterVectorPlacement {
	switch placement {
	case "surface":
		return EmitterVectorSurface
	default:
		return EmitterVectorFill
	}
}

// buildAnimationParams converts config to runtime animation parameters
func buildAnimationParams(config *ParticleConfig) AnimationParams {
	dur := DurationParams{Base: config.Animation.Duration.Value}
	if config.Animation.Duration.Range != nil {
		dur.Base = (config.Animation.Duration.Range.Max + config.Animation.Duration.Range.Min) / 2
		dur.Range = (config.Animation.Duration.Range.Max - config.Animation.Duration.Range.Min) / 2
	}

	posType := config.Animation.Position.Type
	pos := PositionParams{
		UsePolar:     posType == "polar",
		UseAttractor: posType == "attractor",
		Easing:       ParseEasing(config.Animation.Position.Easing),
	}
	switch {
	case pos.UseAttractor:
		if config.Animation.Position.ControlX != nil {
			pos.ControlXMin = config.Animation.Position.ControlX.Min
			pos.ControlXMax = config.Animation.Position.ControlX.Max
		}
		if config.Animation.Position.ControlY != nil {
			pos.ControlYMin = config.Animation.Position.ControlY.Min
			pos.ControlYMax = config.Animation.Position.ControlY.Max
		}
	case pos.UsePolar:
		if config.Animation.Position.Angle != nil {
			pos.AngleMin = config.Animation.Position.Angle.Min
			pos.AngleMax = config.Animation.Position.Angle.Max
		}
		if config.Animation.Position.Distance != nil {
			pos.DistMin = config.Animation.Position.Distance.Min
			pos.DistMax = config.Animation.Position.Distance.Max
		}
		if config.Animation.Position.Speed != nil {
			pos.SpeedMin = config.Animation.Position.Speed.Min
			pos.SpeedMax = config.Animation.Position.Speed.Max
		}
	default: // cartesian
		if config.Animation.Position.StartX != nil {
			pos.StartXMin = config.Animation.Position.StartX.Min
			pos.StartXMax = config.Animation.Position.StartX.Max
		}
		if config.Animation.Position.EndX != nil {
			pos.EndXMin = config.Animation.Position.EndX.Min
			pos.EndXMax = config.Animation.Position.EndX.Max
		}
		if config.Animation.Position.StartY != nil {
			pos.StartYMin = config.Animation.Position.StartY.Min
			pos.StartYMax = config.Animation.Position.StartY.Max
		}
		if config.Animation.Position.EndY != nil {
			pos.EndYMin = config.Animation.Position.EndY.Min
			pos.EndYMax = config.Animation.Position.EndY.Max
		}
	}
	if flow := config.Animation.Position.Flow; flow != nil {
		pos.HasFlow = true
		if flow.Strength != nil {
			pos.FlowStrengthMin = flow.Strength.Min
			pos.FlowStrengthMax = flow.Strength.Max
		}
		pos.FlowScale = flow.Scale
		if pos.FlowScale <= 0 {
			pos.FlowScale = defaultFlowScale
		}
		pos.FlowOctaves = flow.Octaves
		if pos.FlowOctaves <= 0 {
			pos.FlowOctaves = defaultFlowOctaves
		}
		pos.FlowPersistence = flow.Persistence
		if pos.FlowPersistence == 0 {
			pos.FlowPersistence = defaultFlowPersistence
		}
		pos.FlowTimeScale = flow.TimeScale
		if pos.FlowTimeScale == 0 {
			pos.FlowTimeScale = defaultFlowTimeScale
		}
		pos.FlowDrag = flow.Drag
		if pos.FlowDrag == 0 {
			pos.FlowDrag = defaultFlowDrag
		}
		pos.FlowLocalSpace = flow.Space != "world"
		pos.FlowBoundRadius = flow.BoundRadius
		pos.FlowRespawnOnEscape = flow.RespawnOnEscape
	}

	app := AppearanceParams{
		StartAlpha:     config.Animation.Alpha.Start,
		EndAlpha:       config.Animation.Alpha.End,
		AlphaEasing:    ParseEasing(config.Animation.Alpha.Easing),
		StartScale:     config.Animation.Scale.Start,
		EndScale:       config.Animation.Scale.End,
		ScaleEasing:    ParseEasing(config.Animation.Scale.Easing),
		StartRotation:  config.Animation.Rotation.Start,
		EndRotation:    config.Animation.Rotation.End,
		RotationEasing: ParseEasing(config.Animation.Rotation.Easing),
	}
	if app.StartScale == 0 && app.EndScale == 0 {
		app.StartScale = 1.0
		app.EndScale = 1.0
	}

	var clr ColorParams
	if config.Animation.Color != nil {
		clr = ColorParams{
			Enabled: true,
			StartR:  config.Animation.Color.StartR,
			StartG:  config.Animation.Color.StartG,
			StartB:  config.Animation.Color.StartB,
			EndR:    config.Animation.Color.EndR,
			EndG:    config.Animation.Color.EndG,
			EndB:    config.Animation.Color.EndB,
			Easing:  ParseEasing(config.Animation.Color.Easing),
		}
	} else {
		clr = ColorParams{StartR: 1, StartG: 1, StartB: 1, EndR: 1, EndG: 1, EndB: 1}
	}

	return AnimationParams{
		Duration:   dur,
		Position:   pos,
		Appearance: app,
		Color:      clr,
	}
}

// buildSequenceConfig converts a PropertyConfig with steps to a SequenceConfig
func buildSequenceConfig(prop *PropertyConfig) *SequenceConfig {
	if prop == nil || !prop.IsSequence() {
		return nil
	}

	steps := make([]SequenceStep, len(prop.Steps))
	for i, s := range prop.Steps {
		step := SequenceStep{
			FromBase: s.From,
			ToBase:   s.To,
			Duration: s.Duration,
			Easing:   ParseEasing(s.Easing),
		}
		if s.FromRange != nil {
			step.FromRange = (s.FromRange.Max - s.FromRange.Min) / 2
			step.FromBase = (s.FromRange.Max + s.FromRange.Min) / 2
		}
		if s.ToRange != nil {
			step.ToRange = (s.ToRange.Max - s.ToRange.Min) / 2
			step.ToBase = (s.ToRange.Max + s.ToRange.Min) / 2
		}
		steps[i] = step
	}

	return NewSequenceConfig(steps)
}

// buildSequenceConfigs extracts sequence configs from the particle config and sets them on SystemData
func buildSequenceConfigs(config *ParticleConfig, data *SystemData) {
	data.PosXSeq = nil
	data.PosYSeq = nil
	data.AlphaSeq = nil
	data.ScaleSeq = nil
	data.RotSeq = nil

	// Position X/Y sequences
	if config.Animation.Position.X != nil && config.Animation.Position.X.IsSequence() {
		data.PosXSeq = buildSequenceConfig(config.Animation.Position.X)
	}
	if config.Animation.Position.Y != nil && config.Animation.Position.Y.IsSequence() {
		data.PosYSeq = buildSequenceConfig(config.Animation.Position.Y)
	}

	// Alpha sequence
	if config.Animation.Alpha.IsSequence() {
		data.AlphaSeq = buildSequenceConfig(&config.Animation.Alpha)
	}

	// Scale sequence
	if config.Animation.Scale.IsSequence() {
		data.ScaleSeq = buildSequenceConfig(&config.Animation.Scale)
	}

	// Rotation sequence
	if config.Animation.Rotation.IsSequence() {
		data.RotSeq = buildSequenceConfig(&config.Animation.Rotation)
	}
}

// ReloadConfig reloads all cached configurations.
//
// Deprecated: Use NewConfigLoader() and pass it explicitly. This function
// operates on a package-level singleton and will be removed in a future version.
func ReloadConfig() {
	configLoader.ClearCache()
}

// GetConfigLoader returns the package-level config loader.
//
// Deprecated: Use NewConfigLoader() and pass it explicitly. This function
// exposes a package-level singleton and will be removed in a future version.
func GetConfigLoader() *ConfigLoader {
	return configLoader
}
