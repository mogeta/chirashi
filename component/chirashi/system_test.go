package chirashi

import (
	"math"
	"testing"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
)

func TestSpawnRespectsMaxParticles(t *testing.T) {
	sys := &System{cnt: 0}
	data := &SystemData{
		ParticlePool:      make([]Instance, 3),
		ActiveIndices:     make([]int, 0, 3),
		FreeIndices:       []int{2, 1, 0},
		SpawnInterval:     1,
		ParticlesPerSpawn: 10,
		MaxParticles:      3,
		IsLoop:            true,
		EmitterX:          100,
		EmitterY:          200,
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1.0},
			Appearance: AppearanceParams{
				StartScale:     1.0,
				EndScale:       1.0,
				AlphaEasing:    EasingLinear,
				ScaleEasing:    EasingLinear,
				RotationEasing: EasingLinear,
			},
			Color: ColorParams{
				StartR: 1, StartG: 1, StartB: 1,
				EndR: 1, EndG: 1, EndB: 1,
				Easing: EasingLinear,
			},
			Position: PositionParams{Easing: EasingLinear},
		},
	}

	sys.spawn(data)

	if got := data.ActiveCount; got != 3 {
		t.Fatalf("active count got %d, want 3", got)
	}
	if got := len(data.ActiveIndices); got != 3 {
		t.Fatalf("active indices length got %d, want 3", got)
	}
	if got := len(data.FreeIndices); got != 0 {
		t.Fatalf("free indices length got %d, want 0", got)
	}
	if got := data.Metrics.SpawnCount; got != 3 {
		t.Fatalf("spawn count got %d, want 3", got)
	}
}

func TestUpdateParticlesDeactivatesExpired(t *testing.T) {
	sys := &System{}
	data := &SystemData{
		ParticlePool: []Instance{
			{Active: true, SpawnTime: 0, Duration: 0.5},
			{Active: true, SpawnTime: 1.5, Duration: 1.0},
		},
		ActiveIndices: []int{0, 1},
		FreeIndices:   []int{},
		ActiveCount:   2,
		CurrentTime:   1.0,
	}

	sys.updateParticles(data, 1.0/60.0)

	if got := data.ActiveCount; got != 1 {
		t.Fatalf("active count got %d, want 1", got)
	}
	if got := len(data.ActiveIndices); got != 1 {
		t.Fatalf("active indices length got %d, want 1", got)
	}
	if got := len(data.FreeIndices); got != 1 || data.FreeIndices[0] != 0 {
		t.Fatalf("free indices got %v, want [0]", data.FreeIndices)
	}
	if got := data.Metrics.DeactivateCount; got != 1 {
		t.Fatalf("deactivate count got %d, want 1", got)
	}
}

func TestBuildAnimationParamsIncludesFlowDefaults(t *testing.T) {
	cfg := &ParticleConfig{
		Animation: AnimationConfig{
			Duration: DurationConfig{Value: 1},
			Position: PositionConfig{
				Easing: "Linear",
				Flow:   &FlowConfig{Type: "curl", Strength: &RangeFloat{Min: 4, Max: 10}},
			},
			Alpha:    PropertyConfig{Start: 1, End: 0, Easing: "Linear"},
			Scale:    PropertyConfig{Start: 1, End: 1, Easing: "Linear"},
			Rotation: PropertyConfig{Start: 0, End: 0, Easing: "Linear"},
		},
	}

	pos := buildAnimationParams(cfg).Position
	if !pos.HasFlow {
		t.Fatal("expected flow to be enabled")
	}
	if pos.FlowStrengthMin != 4 || pos.FlowStrengthMax != 10 {
		t.Fatalf("unexpected flow strength range: [%v,%v]", pos.FlowStrengthMin, pos.FlowStrengthMax)
	}
	if pos.FlowScale != 160 || pos.FlowOctaves != 2 || pos.FlowPersistence != 0.5 || pos.FlowTimeScale != 0.2 || pos.FlowDrag != 0.96 {
		t.Fatalf("unexpected flow defaults: %+v", pos)
	}
	if !pos.FlowLocalSpace {
		t.Fatal("expected local flow space by default")
	}
}

func TestUpdateParticlesAppliesCurlFlow(t *testing.T) {
	sys := &System{}
	data := &SystemData{
		ParticlePool: []Instance{
			{
				Active:         true,
				SpawnTime:      0,
				Duration:       10,
				StartX:         0,
				EndX:           0,
				StartY:         0,
				EndY:           0,
				PositionEasing: EasingLinear,
				HasFlow:        true,
				FlowGain:       24,
				FlowSeedX:      0.6,
				FlowSeedY:      -0.4,
			},
		},
		ActiveIndices: []int{0},
		ActiveCount:   1,
		CurrentTime:   0.5,
		AnimParams: AnimationParams{
			Position: PositionParams{
				Easing:          EasingLinear,
				HasFlow:         true,
				FlowStrengthMin: 24,
				FlowStrengthMax: 24,
				FlowScale:       160,
				FlowOctaves:     2,
				FlowPersistence: 0.5,
				FlowTimeScale:   0.3,
				FlowDrag:        0.96,
				FlowLocalSpace:  true,
			},
		},
	}

	sys.updateParticles(data, 1.0/60.0)

	p := data.ParticlePool[0]
	if p.FlowOffsetX == 0 && p.FlowOffsetY == 0 {
		t.Fatal("expected curl flow to move the particle offset")
	}
	if p.FlowVelX == 0 && p.FlowVelY == 0 {
		t.Fatal("expected curl flow to update particle velocity")
	}
}

func TestUpdateParticlesResetsFlowWhenLeavingBounds(t *testing.T) {
	sys := &System{}
	data := &SystemData{
		EmitterX:      0,
		EmitterY:      0,
		ParticlePool:  []Instance{{Active: true, SpawnTime: 0, Duration: 10, PositionEasing: EasingLinear, HasFlow: true, FlowGain: 12, FlowOffsetX: 30, FlowOffsetY: 0, FlowVelX: 5, FlowVelY: 5}},
		ActiveIndices: []int{0},
		ActiveCount:   1,
		CurrentTime:   0.5,
		AnimParams: AnimationParams{
			Position: PositionParams{
				Easing:              EasingLinear,
				HasFlow:             true,
				FlowStrengthMin:     12,
				FlowStrengthMax:     12,
				FlowScale:           160,
				FlowOctaves:         1,
				FlowPersistence:     0.5,
				FlowTimeScale:       0.2,
				FlowDrag:            0.96,
				FlowLocalSpace:      true,
				FlowBoundRadius:     10,
				FlowRespawnOnEscape: true,
			},
		},
	}

	sys.updateParticles(data, 1.0/60.0)

	p := data.ParticlePool[0]
	if p.FlowOffsetX != 0 || p.FlowOffsetY != 0 || p.FlowVelX != 0 || p.FlowVelY != 0 {
		t.Fatalf("expected flow state to reset after leaving bounds, got offset=(%v,%v) vel=(%v,%v)", p.FlowOffsetX, p.FlowOffsetY, p.FlowVelX, p.FlowVelY)
	}
}

func TestUpdateRemovesExpiredOneShotEntity(t *testing.T) {
	world := donburi.NewWorld()
	gameECS := ecs.NewECS(world)
	sys := NewSystem()

	entity := world.Create(Component)
	entry := world.Entry(entity)
	donburi.SetValue(entry, Component, SystemData{
		IsLoop:        false,
		LifeTime:      1,
		SpawnInterval: 0,
	})

	sys.Update(gameECS)

	if world.Valid(entity) {
		t.Fatalf("expected entity to be removed when one-shot lifetime expires")
	}
}

func TestUpdateTrailTracksEmitterHistory(t *testing.T) {
	data := &SystemData{
		CurrentTime: 0,
		EmitterX:    10,
		EmitterY:    20,
		Trail: TrailData{
			Params: TrailParams{
				Enabled:          true,
				Mode:             "emitter",
				MaxPoints:        3,
				MinPointDistance: 5,
				MaxPointAge:      0.25,
			},
		},
	}

	updateTrail(data)
	if got := len(data.Trail.Runtime.Points); got != 1 {
		t.Fatalf("initial point count got %d, want 1", got)
	}

	data.CurrentTime = 0.05
	data.EmitterX = 12
	data.EmitterY = 22
	updateTrail(data)
	if got := len(data.Trail.Runtime.Points); got != 1 {
		t.Fatalf("short move should update head in place, got %d points", got)
	}
	if data.Trail.Runtime.Points[0].X != 12 || data.Trail.Runtime.Points[0].CapturedAt != 0.05 {
		t.Fatalf("expected updated head point, got %+v", data.Trail.Runtime.Points[0])
	}

	data.CurrentTime = 0.10
	data.EmitterX = 20
	data.EmitterY = 22
	updateTrail(data)
	if got := len(data.Trail.Runtime.Points); got != 2 {
		t.Fatalf("large move should append point, got %d", got)
	}

	data.CurrentTime = 0.40
	data.EmitterX = 30
	data.EmitterY = 22
	updateTrail(data)
	if got := len(data.Trail.Runtime.Points); got != 1 {
		t.Fatalf("expired points should be pruned before appending, got %d", got)
	}
	if data.Trail.Runtime.Points[0].X != 30 {
		t.Fatalf("expected newest point to remain after prune, got %+v", data.Trail.Runtime.Points[0])
	}
}

func TestBuildTrailDataDefaults(t *testing.T) {
	trail := buildTrailData(&TrailConfig{Enabled: true})
	if !trail.Params.Enabled {
		t.Fatal("expected trail to be enabled")
	}
	if trail.Params.MaxPoints != defaultTrailMaxPoints || trail.Params.MinPointDistance != defaultTrailMinPointDistance || trail.Params.MaxPointAge != defaultTrailMaxPointAge {
		t.Fatalf("unexpected trail defaults: %+v", trail)
	}
	if trail.Params.Mode != "" {
		t.Fatalf("expected empty trail mode to preserve emitter default behavior, got %q", trail.Params.Mode)
	}
	if trail.Params.WidthStart != 0 || trail.Params.AlphaStart != 0 {
		t.Fatalf("expected zero width/alpha when omitted, got width=%v alpha=%v", trail.Params.WidthStart, trail.Params.AlphaStart)
	}
}

func TestUpdateTrailTracksParticleHistory(t *testing.T) {
	data := &SystemData{
		CurrentTime:   0.10,
		ActiveIndices: []int{0},
		ParticlePool: []Instance{
			{
				Active:         true,
				SpawnTime:      0,
				Duration:       1,
				StartX:         0,
				EndX:           20,
				StartY:         0,
				EndY:           0,
				PositionEasing: EasingLinear,
				TrailPoints:    make([]TrailPoint, 0, 4),
			},
		},
		Trail: TrailData{
			Params: TrailParams{
				Enabled:          true,
				Mode:             "particle",
				MaxPoints:        4,
				MinPointDistance: 3,
				MaxPointAge:      0.3,
				WidthStart:       8,
				AlphaStart:       1,
			},
		},
	}

	updateTrail(data)
	if got := len(data.ParticlePool[0].TrailPoints); got != 1 {
		t.Fatalf("expected first particle trail sample, got %d", got)
	}

	data.CurrentTime = 0.30
	updateTrail(data)
	if got := len(data.ParticlePool[0].TrailPoints); got != 2 {
		t.Fatalf("expected second particle trail sample after movement, got %d", got)
	}

	data.CurrentTime = 0.70
	updateTrail(data)
	if got := len(data.ParticlePool[0].TrailPoints); got != 1 {
		t.Fatalf("expected old particle trail samples to prune, got %d", got)
	}
}

func TestExpiredParticleTrailBecomesGhostUntilFadeCompletes(t *testing.T) {
	sys := &System{}
	data := &SystemData{
		CurrentTime: 0.6,
		ParticlePool: []Instance{
			{
				Active:      true,
				SpawnTime:   0,
				Duration:    0.5,
				TrailPoints: []TrailPoint{{X: 0, Y: 0, CapturedAt: 0.2}, {X: 10, Y: 0, CapturedAt: 0.5}},
			},
		},
		ActiveIndices: []int{0},
		ActiveCount:   1,
		Trail: TrailData{
			Params: TrailParams{
				Enabled:     true,
				Mode:        "particle",
				MaxPoints:   4,
				MaxPointAge: 0.4,
			},
		},
	}

	sys.updateParticles(data, 1.0/60.0)
	if data.ActiveCount != 0 {
		t.Fatalf("expected particle to deactivate, got activeCount=%d", data.ActiveCount)
	}
	if got := len(data.Trail.Runtime.Ghosts); got != 1 {
		t.Fatalf("expected one detached trail ghost, got %d", got)
	}
	if !trailHasVisiblePoints(data) {
		t.Fatal("expected detached trail ghost to remain visible after particle expiry")
	}

	data.CurrentTime = 1.0
	updateTrail(data)
	if trailHasVisiblePoints(data) {
		t.Fatal("expected detached trail ghost to disappear after max_point_age")
	}
}

func TestSpawnCircleEmitterSamplesInsideRadius(t *testing.T) {
	sys := &System{cnt: 0}
	data := &SystemData{
		ParticlePool:      make([]Instance, 32),
		ActiveIndices:     make([]int, 0, 32),
		FreeIndices:       make([]int, 32),
		SpawnInterval:     1,
		ParticlesPerSpawn: 32,
		MaxParticles:      32,
		IsLoop:            true,
		EmitterX:          100,
		EmitterY:          200,
		EmitterShape: EmitterShapeParams{
			Type:      EmitterShapeCircle,
			RadiusMin: 10,
			RadiusMax: 20,
		},
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1.0},
			Appearance: AppearanceParams{
				StartScale:     1.0,
				EndScale:       1.0,
				AlphaEasing:    EasingLinear,
				ScaleEasing:    EasingLinear,
				RotationEasing: EasingLinear,
			},
			Color: ColorParams{
				StartR: 1, StartG: 1, StartB: 1,
				EndR: 1, EndG: 1, EndB: 1,
				Easing: EasingLinear,
			},
			Position: PositionParams{Easing: EasingLinear},
		},
	}
	for i := range data.FreeIndices {
		data.FreeIndices[i] = len(data.FreeIndices) - 1 - i
	}

	sys.spawn(data)

	for _, idx := range data.ActiveIndices {
		p := data.ParticlePool[idx]
		dx := p.StartX - data.EmitterX
		dy := p.StartY - data.EmitterY
		dist := float32(math.Hypot(float64(dx), float64(dy)))
		if dist < 10 || dist > 20 {
			t.Fatalf("particle spawned at distance %v, want within [10,20]", dist)
		}
	}
}

func TestSpawnLineEmitterRespectsRotation(t *testing.T) {
	sys := &System{cnt: 0}
	data := &SystemData{
		ParticlePool:      make([]Instance, 16),
		ActiveIndices:     make([]int, 0, 16),
		FreeIndices:       make([]int, 16),
		SpawnInterval:     1,
		ParticlesPerSpawn: 16,
		MaxParticles:      16,
		IsLoop:            true,
		EmitterX:          50,
		EmitterY:          80,
		EmitterShape: EmitterShapeParams{
			Type:     EmitterShapeLine,
			Length:   60,
			Rotation: math.Pi / 2,
		},
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1.0},
			Appearance: AppearanceParams{
				StartScale:     1.0,
				EndScale:       1.0,
				AlphaEasing:    EasingLinear,
				ScaleEasing:    EasingLinear,
				RotationEasing: EasingLinear,
			},
			Color: ColorParams{
				StartR: 1, StartG: 1, StartB: 1,
				EndR: 1, EndG: 1, EndB: 1,
				Easing: EasingLinear,
			},
			Position: PositionParams{Easing: EasingLinear},
		},
	}
	for i := range data.FreeIndices {
		data.FreeIndices[i] = len(data.FreeIndices) - 1 - i
	}

	sys.spawn(data)

	for _, idx := range data.ActiveIndices {
		p := data.ParticlePool[idx]
		if math.Abs(float64(p.StartX-data.EmitterX)) > 0.001 {
			t.Fatalf("line emitter with vertical rotation should keep x constant, got startX=%v emitterX=%v", p.StartX, data.EmitterX)
		}
		if p.StartY < data.EmitterY-30 || p.StartY > data.EmitterY+30 {
			t.Fatalf("line emitter startY=%v out of range", p.StartY)
		}
	}
}

func TestSpawnRectVectorFillDistributesAcrossArea(t *testing.T) {
	sys := &System{cnt: 0}
	data := &SystemData{
		ParticlePool:      make([]Instance, 9),
		ActiveIndices:     make([]int, 0, 9),
		FreeIndices:       make([]int, 9),
		SpawnInterval:     1,
		ParticlesPerSpawn: 9,
		MaxParticles:      9,
		IsLoop:            true,
		EmitterX:          100,
		EmitterY:          200,
		EmitterVector: EmitterVectorParams{
			Enabled:   true,
			Type:      EmitterVectorRect,
			Placement: EmitterVectorFill,
			Rect:      EmitterVectorRectParams{Width: 90, Height: 60},
		},
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1.0},
			Appearance: AppearanceParams{
				StartScale:     1.0,
				EndScale:       1.0,
				AlphaEasing:    EasingLinear,
				ScaleEasing:    EasingLinear,
				RotationEasing: EasingLinear,
			},
			Color:    ColorParams{StartR: 1, StartG: 1, StartB: 1, EndR: 1, EndG: 1, EndB: 1, Easing: EasingLinear},
			Position: PositionParams{Easing: EasingLinear},
		},
	}
	for i := range data.FreeIndices {
		data.FreeIndices[i] = len(data.FreeIndices) - 1 - i
	}

	sys.spawn(data)

	seenCols := map[int]bool{}
	seenRows := map[int]bool{}
	for _, idx := range data.ActiveIndices {
		p := data.ParticlePool[idx]
		dx := p.StartX - data.EmitterX
		dy := p.StartY - data.EmitterY
		if dx < -45 || dx > 45 || dy < -30 || dy > 30 {
			t.Fatalf("vector fill spawn out of bounds: (%v,%v)", dx, dy)
		}
		col := int(math.Floor(float64((dx + 45) / 30)))
		row := int(math.Floor(float64((dy + 30) / 20)))
		seenCols[col] = true
		seenRows[row] = true
	}
	if len(seenCols) < 3 || len(seenRows) < 3 {
		t.Fatalf("expected fill placement to cover multiple rows/cols, got cols=%d rows=%d", len(seenCols), len(seenRows))
	}
}

func TestSpawnRectVectorSurfaceStaysOnPerimeter(t *testing.T) {
	sys := &System{cnt: 0}
	data := &SystemData{
		ParticlePool:      make([]Instance, 8),
		ActiveIndices:     make([]int, 0, 8),
		FreeIndices:       make([]int, 8),
		SpawnInterval:     1,
		ParticlesPerSpawn: 8,
		MaxParticles:      8,
		IsLoop:            true,
		EmitterX:          0,
		EmitterY:          0,
		EmitterVector: EmitterVectorParams{
			Enabled:   true,
			Type:      EmitterVectorRect,
			Placement: EmitterVectorSurface,
			Rect:      EmitterVectorRectParams{Width: 80, Height: 40},
		},
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1.0},
			Appearance: AppearanceParams{
				StartScale:     1.0,
				EndScale:       1.0,
				AlphaEasing:    EasingLinear,
				ScaleEasing:    EasingLinear,
				RotationEasing: EasingLinear,
			},
			Color:    ColorParams{StartR: 1, StartG: 1, StartB: 1, EndR: 1, EndG: 1, EndB: 1, Easing: EasingLinear},
			Position: PositionParams{Easing: EasingLinear},
		},
	}
	for i := range data.FreeIndices {
		data.FreeIndices[i] = len(data.FreeIndices) - 1 - i
	}

	sys.spawn(data)

	for _, idx := range data.ActiveIndices {
		p := data.ParticlePool[idx]
		x := p.StartX
		y := p.StartY
		onVertical := math.Abs(float64(math.Abs(float64(x))-40)) < 0.001 && y >= -20.001 && y <= 20.001
		onHorizontal := math.Abs(float64(math.Abs(float64(y))-20)) < 0.001 && x >= -40.001 && x <= 40.001
		if !onVertical && !onHorizontal {
			t.Fatalf("vector surface spawn should stay on perimeter, got (%v,%v)", x, y)
		}
	}
}

func TestSpawnPolylineVectorSurfaceStaysOnSegments(t *testing.T) {
	sys := &System{cnt: 0}
	data := &SystemData{
		ParticlePool:      make([]Instance, 6),
		ActiveIndices:     make([]int, 0, 6),
		FreeIndices:       make([]int, 6),
		SpawnInterval:     1,
		ParticlesPerSpawn: 6,
		MaxParticles:      6,
		IsLoop:            true,
		EmitterX:          0,
		EmitterY:          0,
		EmitterVector: EmitterVectorParams{
			Enabled:   true,
			Type:      EmitterVectorPolyline,
			Placement: EmitterVectorSurface,
			Polyline: buildEmitterVectorPolylineParams(&EmitterVectorPolylineConfig{
				Points: []EmitterVectorPoint{
					{X: -30, Y: 0},
					{X: 0, Y: 0},
					{X: 0, Y: 30},
				},
			}),
		},
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1.0},
			Appearance: AppearanceParams{
				StartScale:     1.0,
				EndScale:       1.0,
				AlphaEasing:    EasingLinear,
				ScaleEasing:    EasingLinear,
				RotationEasing: EasingLinear,
			},
			Color:    ColorParams{StartR: 1, StartG: 1, StartB: 1, EndR: 1, EndG: 1, EndB: 1, Easing: EasingLinear},
			Position: PositionParams{Easing: EasingLinear},
		},
	}
	for i := range data.FreeIndices {
		data.FreeIndices[i] = len(data.FreeIndices) - 1 - i
	}

	sys.spawn(data)

	for _, idx := range data.ActiveIndices {
		p := data.ParticlePool[idx]
		onHorizontal := math.Abs(float64(p.StartY)) < 0.001 && p.StartX >= -30.001 && p.StartX <= 0.001
		onVertical := math.Abs(float64(p.StartX)) < 0.001 && p.StartY >= -0.001 && p.StartY <= 30.001
		if !onHorizontal && !onVertical {
			t.Fatalf("vector polyline spawn should stay on configured segments, got (%v,%v)", p.StartX, p.StartY)
		}
	}
}

func TestBuildEmitterVectorPolylineParamsQuadraticCompilesCurve(t *testing.T) {
	params := buildEmitterVectorPolylineParams(&EmitterVectorPolylineConfig{
		Interpolation: "quadratic",
		CurveSteps:    6,
		Points: []EmitterVectorPoint{
			{X: -40, Y: 0},
			{X: 0, Y: -40},
			{X: 40, Y: 0},
		},
	})

	if len(params.Points) != 7 {
		t.Fatalf("expected 7 compiled points, got %d", len(params.Points))
	}
	if params.TotalLength <= 0 {
		t.Fatalf("expected quadratic polyline length to be positive")
	}
	if params.Points[0].X != -40 || params.Points[len(params.Points)-1].X != 40 {
		t.Fatalf("compiled curve should preserve endpoints, got start=%+v end=%+v", params.Points[0], params.Points[len(params.Points)-1])
	}
}

func TestSpawnCircleEmitterArcLimitsAngle(t *testing.T) {
	sys := &System{cnt: 0}
	data := &SystemData{
		ParticlePool:      make([]Instance, 16),
		ActiveIndices:     make([]int, 0, 16),
		FreeIndices:       make([]int, 16),
		SpawnInterval:     1,
		ParticlesPerSpawn: 16,
		MaxParticles:      16,
		IsLoop:            true,
		EmitterShape: EmitterShapeParams{
			Type:       EmitterShapeCircle,
			RadiusMin:  10,
			RadiusMax:  10,
			StartAngle: 0,
			EndAngle:   math.Pi / 2,
			FromEdge:   true,
		},
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1.0},
			Appearance: AppearanceParams{
				StartScale:     1.0,
				EndScale:       1.0,
				AlphaEasing:    EasingLinear,
				ScaleEasing:    EasingLinear,
				RotationEasing: EasingLinear,
			},
			Color: ColorParams{
				StartR: 1, StartG: 1, StartB: 1,
				EndR: 1, EndG: 1, EndB: 1,
				Easing: EasingLinear,
			},
			Position: PositionParams{Easing: EasingLinear},
		},
	}
	for i := range data.FreeIndices {
		data.FreeIndices[i] = len(data.FreeIndices) - 1 - i
	}

	sys.spawn(data)

	for _, idx := range data.ActiveIndices {
		p := data.ParticlePool[idx]
		if p.StartX < -0.001 || p.StartY < -0.001 {
			t.Fatalf("arc emitter spawned outside first quadrant: (%v, %v)", p.StartX, p.StartY)
		}
	}
}

func TestSpawnBoxEmitterFromEdgeStaysOnPerimeter(t *testing.T) {
	sys := &System{cnt: 0}
	data := &SystemData{
		ParticlePool:      make([]Instance, 16),
		ActiveIndices:     make([]int, 0, 16),
		FreeIndices:       make([]int, 16),
		SpawnInterval:     1,
		ParticlesPerSpawn: 16,
		MaxParticles:      16,
		IsLoop:            true,
		EmitterX:          10,
		EmitterY:          20,
		EmitterShape: EmitterShapeParams{
			Type:     EmitterShapeBox,
			Width:    40,
			Height:   20,
			FromEdge: true,
		},
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1.0},
			Appearance: AppearanceParams{
				StartScale:     1.0,
				EndScale:       1.0,
				AlphaEasing:    EasingLinear,
				ScaleEasing:    EasingLinear,
				RotationEasing: EasingLinear,
			},
			Color: ColorParams{
				StartR: 1, StartG: 1, StartB: 1,
				EndR: 1, EndG: 1, EndB: 1,
				Easing: EasingLinear,
			},
			Position: PositionParams{Easing: EasingLinear},
		},
	}
	for i := range data.FreeIndices {
		data.FreeIndices[i] = len(data.FreeIndices) - 1 - i
	}

	sys.spawn(data)

	for _, idx := range data.ActiveIndices {
		p := data.ParticlePool[idx]
		dx := p.StartX - data.EmitterX
		dy := p.StartY - data.EmitterY
		onVertical := math.Abs(math.Abs(float64(dx))-20) < 0.001 && math.Abs(float64(dy)) <= 10.001
		onHorizontal := math.Abs(math.Abs(float64(dy))-10) < 0.001 && math.Abs(float64(dx)) <= 20.001
		if !onVertical && !onHorizontal {
			t.Fatalf("box edge emitter spawned inside area: (%v, %v)", dx, dy)
		}
	}
}

func TestSpawnCircleEmitterFullCircleWithTwoPiEndAngle(t *testing.T) {
	sys := &System{cnt: 0}
	data := &SystemData{
		ParticlePool:      make([]Instance, 128),
		ActiveIndices:     make([]int, 0, 128),
		FreeIndices:       make([]int, 128),
		SpawnInterval:     1,
		ParticlesPerSpawn: 128,
		MaxParticles:      128,
		IsLoop:            true,
		EmitterShape: EmitterShapeParams{
			Type:       EmitterShapeCircle,
			RadiusMin:  10,
			RadiusMax:  10,
			StartAngle: 0,
			EndAngle:   float32(2 * math.Pi),
			FromEdge:   true,
		},
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1.0},
			Appearance: AppearanceParams{
				StartScale:     1.0,
				EndScale:       1.0,
				AlphaEasing:    EasingLinear,
				ScaleEasing:    EasingLinear,
				RotationEasing: EasingLinear,
			},
			Color: ColorParams{
				StartR: 1, StartG: 1, StartB: 1,
				EndR: 1, EndG: 1, EndB: 1,
				Easing: EasingLinear,
			},
			Position: PositionParams{Easing: EasingLinear},
		},
	}
	for i := range data.FreeIndices {
		data.FreeIndices[i] = len(data.FreeIndices) - 1 - i
	}

	sys.spawn(data)

	var hasNegX, hasPosX, hasNegY, hasPosY bool
	for _, idx := range data.ActiveIndices {
		p := data.ParticlePool[idx]
		if p.StartX < 0 {
			hasNegX = true
		}
		if p.StartX > 0 {
			hasPosX = true
		}
		if p.StartY < 0 {
			hasNegY = true
		}
		if p.StartY > 0 {
			hasPosY = true
		}
	}
	if !(hasNegX && hasPosX && hasNegY && hasPosY) {
		t.Fatalf("full circle sampling did not cover all quadrants: negX=%v posX=%v negY=%v posY=%v", hasNegX, hasPosX, hasNegY, hasPosY)
	}
}

func TestSpawnCircleEmitterTreatsSixPointTwoEightAsFullCircle(t *testing.T) {
	sys := &System{cnt: 0}
	data := &SystemData{
		ParticlePool:      make([]Instance, 128),
		ActiveIndices:     make([]int, 0, 128),
		FreeIndices:       make([]int, 128),
		SpawnInterval:     1,
		ParticlesPerSpawn: 128,
		MaxParticles:      128,
		IsLoop:            true,
		EmitterShape: EmitterShapeParams{
			Type:       EmitterShapeCircle,
			RadiusMin:  10,
			RadiusMax:  10,
			StartAngle: 0,
			EndAngle:   6.28,
			FromEdge:   true,
		},
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1.0},
			Appearance: AppearanceParams{
				StartScale:     1.0,
				EndScale:       1.0,
				AlphaEasing:    EasingLinear,
				ScaleEasing:    EasingLinear,
				RotationEasing: EasingLinear,
			},
			Color: ColorParams{
				StartR: 1, StartG: 1, StartB: 1,
				EndR: 1, EndG: 1, EndB: 1,
				Easing: EasingLinear,
			},
			Position: PositionParams{Easing: EasingLinear},
		},
	}
	for i := range data.FreeIndices {
		data.FreeIndices[i] = len(data.FreeIndices) - 1 - i
	}

	sys.spawn(data)

	var hasNegX, hasPosX, hasNegY, hasPosY bool
	for _, idx := range data.ActiveIndices {
		p := data.ParticlePool[idx]
		if p.StartX < 0 {
			hasNegX = true
		}
		if p.StartX > 0 {
			hasPosX = true
		}
		if p.StartY < 0 {
			hasNegY = true
		}
		if p.StartY > 0 {
			hasPosY = true
		}
	}
	if !(hasNegX && hasPosX && hasNegY && hasPosY) {
		t.Fatalf("6.28 full circle sampling did not cover all quadrants: negX=%v posX=%v negY=%v posY=%v", hasNegX, hasPosX, hasNegY, hasPosY)
	}
}

func TestSpawnCircleEmitterWrapArc(t *testing.T) {
	sys := &System{cnt: 0}
	data := &SystemData{
		ParticlePool:      make([]Instance, 64),
		ActiveIndices:     make([]int, 0, 64),
		FreeIndices:       make([]int, 64),
		SpawnInterval:     1,
		ParticlesPerSpawn: 64,
		MaxParticles:      64,
		IsLoop:            true,
		EmitterShape: EmitterShapeParams{
			Type:       EmitterShapeCircle,
			RadiusMin:  10,
			RadiusMax:  10,
			StartAngle: 5.5,
			EndAngle:   0.5,
			FromEdge:   true,
		},
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1.0},
			Appearance: AppearanceParams{
				StartScale:     1.0,
				EndScale:       1.0,
				AlphaEasing:    EasingLinear,
				ScaleEasing:    EasingLinear,
				RotationEasing: EasingLinear,
			},
			Color: ColorParams{
				StartR: 1, StartG: 1, StartB: 1,
				EndR: 1, EndG: 1, EndB: 1,
				Easing: EasingLinear,
			},
			Position: PositionParams{Easing: EasingLinear},
		},
	}
	for i := range data.FreeIndices {
		data.FreeIndices[i] = len(data.FreeIndices) - 1 - i
	}

	sys.spawn(data)

	for _, idx := range data.ActiveIndices {
		p := data.ParticlePool[idx]
		angle := normalizeAngle(float32(math.Atan2(float64(p.StartY), float64(p.StartX))))
		if angle > 0.5 && angle < 5.5 {
			t.Fatalf("wrap arc sampled outside expected range: angle=%v", angle)
		}
	}
}
