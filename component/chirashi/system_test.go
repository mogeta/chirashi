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

	sys.updateParticles(data)

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

func TestSpawnInitializesTurbulenceState(t *testing.T) {
	sys := &System{cnt: 0}
	data := &SystemData{
		ParticlePool:      make([]Instance, 8),
		ActiveIndices:     make([]int, 0, 8),
		FreeIndices:       []int{7, 6, 5, 4, 3, 2, 1, 0},
		SpawnInterval:     1,
		ParticlesPerSpawn: 8,
		MaxParticles:      8,
		IsLoop:            true,
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1.0},
			Appearance: AppearanceParams{
				StartScale: 1.0, EndScale: 1.0,
				AlphaEasing: EasingLinear, ScaleEasing: EasingLinear, RotationEasing: EasingLinear,
			},
			Color: ColorParams{StartR: 1, StartG: 1, StartB: 1, EndR: 1, EndG: 1, EndB: 1, Easing: EasingLinear},
			Position: PositionParams{
				Easing:                EasingLinear,
				HasTurbulence:         true,
				TurbulenceStrengthMin: 4,
				TurbulenceStrengthMax: 8,
			},
		},
	}

	sys.spawn(data)

	for _, idx := range data.ActiveIndices {
		p := data.ParticlePool[idx]
		if p.TurbulenceGain < 4 || p.TurbulenceGain > 8 {
			t.Fatalf("turbulence gain got %v, want within [4,8]", p.TurbulenceGain)
		}
	}
}

func TestApplyTurbulenceLocalVsWorldSpace(t *testing.T) {
	p := &Instance{TurbulenceGain: 10}
	localData := &SystemData{
		EmitterX: 100,
		EmitterY: 200,
		AnimParams: AnimationParams{
			Position: PositionParams{
				HasTurbulence:         true,
				TurbulenceScale:       80,
				TurbulenceOctaves:     2,
				TurbulencePersistence: 0.5,
				TurbulenceTimeScale:   1,
				TurbulenceLocalSpace:  true,
				TurbulenceEnvStart:    1,
				TurbulenceEnvEnd:      1,
				TurbulenceEnvEasing:   EasingLinear,
			},
		},
	}
	worldData := &SystemData{
		EmitterX: 100,
		EmitterY: 200,
		AnimParams: AnimationParams{
			Position: PositionParams{
				HasTurbulence:         true,
				TurbulenceScale:       80,
				TurbulenceOctaves:     2,
				TurbulencePersistence: 0.5,
				TurbulenceTimeScale:   1,
				TurbulenceLocalSpace:  false,
				TurbulenceEnvStart:    1,
				TurbulenceEnvEnd:      1,
				TurbulenceEnvEasing:   EasingLinear,
			},
		},
	}

	localX, localY := applyTurbulence(localData, p, 120, 220, 0.3, 0.5)
	worldX, worldY := applyTurbulence(worldData, p, 120, 220, 0.3, 0.5)
	if almostEqualFloat32Turb(localX, worldX, 1e-4) && almostEqualFloat32Turb(localY, worldY, 1e-4) {
		t.Fatal("expected local and world turbulence samples to differ")
	}
}

func TestApplyTurbulenceEnvelopeZeroesOffset(t *testing.T) {
	data := &SystemData{
		AnimParams: AnimationParams{
			Position: PositionParams{
				HasTurbulence:         true,
				TurbulenceScale:       64,
				TurbulenceOctaves:     1,
				TurbulencePersistence: 0.5,
				TurbulenceTimeScale:   1,
				TurbulenceLocalSpace:  true,
				TurbulenceEnvStart:    0,
				TurbulenceEnvEnd:      0,
				TurbulenceEnvEasing:   EasingLinear,
			},
		},
	}
	p := &Instance{TurbulenceGain: 12}

	x, y := applyTurbulence(data, p, 10, 20, 0.4, 0.5)
	if !almostEqualFloat32Turb(x, 10, 1e-4) || !almostEqualFloat32Turb(y, 20, 1e-4) {
		t.Fatalf("turbulence envelope should zero offset, got (%v, %v)", x, y)
	}
}

func TestEvaluateParticlePositionAndVelocityIncludesTurbulence(t *testing.T) {
	data := &SystemData{
		AnimParams: AnimationParams{
			Position: PositionParams{
				Easing:                EasingLinear,
				HasTurbulence:         true,
				TurbulenceScale:       64,
				TurbulenceOctaves:     2,
				TurbulencePersistence: 0.5,
				TurbulenceTimeScale:   1,
				TurbulenceLocalSpace:  true,
				TurbulenceEnvStart:    1,
				TurbulenceEnvEnd:      1,
				TurbulenceEnvEasing:   EasingLinear,
			},
		},
	}
	p := &Instance{
		Duration:          1,
		StartX:            0,
		EndX:              100,
		StartY:            0,
		EndY:              0,
		PositionEasing:    EasingLinear,
		TurbulenceGain:    12,
		TurbulenceOffsetX: 0.35,
		TurbulenceOffsetY: -0.2,
	}

	x, y, vx, vy := evaluateParticlePositionAndVelocity(data, p, 0.5, 0.5)
	if almostEqualFloat32Turb(x, 50, 1e-4) && almostEqualFloat32Turb(y, 0, 1e-4) {
		t.Fatal("expected turbulence-adjusted position to differ from base path")
	}
	if almostEqualFloat32Turb(vx, 100, 1e-2) && almostEqualFloat32Turb(vy, 0, 1e-2) {
		t.Fatal("expected turbulence-adjusted velocity to differ from base path velocity")
	}
}

func TestBuildAnimationParamsIncludesTurbulenceDefaults(t *testing.T) {
	cfg := &ParticleConfig{
		Animation: AnimationConfig{
			Duration: DurationConfig{Value: 1},
			Position: PositionConfig{
				Easing: "Linear",
				Turbulence: &TurbulenceConfig{
					Strength: &RangeFloat{Min: 2, Max: 6},
				},
			},
			Alpha:    PropertyConfig{Start: 1, End: 0, Easing: "Linear"},
			Scale:    PropertyConfig{Start: 1, End: 1, Easing: "Linear"},
			Rotation: PropertyConfig{Start: 0, End: 0, Easing: "Linear"},
		},
	}

	pos := buildAnimationParams(cfg).Position
	if !pos.HasTurbulence {
		t.Fatal("expected turbulence to be enabled")
	}
	if pos.TurbulenceScale != 96 || pos.TurbulenceOctaves != 1 || pos.TurbulencePersistence != 0.5 || pos.TurbulenceTimeScale != 1 {
		t.Fatalf("unexpected turbulence defaults: %+v", pos)
	}
	if !pos.TurbulenceLocalSpace {
		t.Fatal("expected default turbulence space to be local")
	}
}

func TestBuildAnimationParamsIncludesPositionNoise(t *testing.T) {
	cfg := &ParticleConfig{
		Animation: AnimationConfig{
			Duration: DurationConfig{Value: 1},
			Position: PositionConfig{
				Easing: "Linear",
				NoiseX: &NoiseConfig{Amplitude: 8, Frequency: 0.6, Octaves: 2, Seed: 1},
				NoiseY: &NoiseConfig{Amplitude: 4, Frequency: 1.1, Octaves: 1, Seed: 2},
			},
			Alpha:    PropertyConfig{Start: 1, End: 0, Easing: "Linear"},
			Scale:    PropertyConfig{Start: 1, End: 1, Easing: "Linear"},
			Rotation: PropertyConfig{Start: 0, End: 0, Easing: "Linear"},
		},
	}

	pos := buildAnimationParams(cfg).Position
	if !pos.PositionNoiseX.Enabled || !pos.PositionNoiseY.Enabled {
		t.Fatal("expected position noise to be enabled")
	}
	if pos.PositionNoiseX.Amplitude != 8 || pos.PositionNoiseY.Frequency != 1.1 {
		t.Fatalf("unexpected position noise params: %+v", pos)
	}
}

func TestBuildAnimationParamsIncludesPropertyNoise(t *testing.T) {
	cfg := &ParticleConfig{
		Animation: AnimationConfig{
			Duration: DurationConfig{Value: 1},
			Position: PositionConfig{Easing: "Linear"},
			Alpha:    PropertyConfig{Start: 1, End: 0, Easing: "Linear", Noise: &NoiseConfig{Amplitude: 0.2, Frequency: 1.5, Octaves: 2, Seed: 3}},
			Scale:    PropertyConfig{Start: 1, End: 1, Easing: "Linear", Noise: &NoiseConfig{Amplitude: 0.5, Frequency: 0.75, Octaves: 3, Seed: 5}},
			Rotation: PropertyConfig{Start: 0, End: 1, Easing: "Linear", Noise: &NoiseConfig{Amplitude: 0.3, Frequency: 0.9, Octaves: 2, Seed: 7}},
		},
	}

	app := buildAnimationParams(cfg).Appearance
	if !app.AlphaNoise.Enabled || !app.ScaleNoise.Enabled || !app.RotationNoise.Enabled {
		t.Fatal("expected property noise to be enabled")
	}
	if app.ScaleNoise.Octaves != 3 || app.AlphaNoise.Seed != 3 || app.RotationNoise.Frequency != 0.9 {
		t.Fatalf("unexpected property noise params: %+v", app)
	}
}

func TestResolveParticleScaleAppliesContinuousNoise(t *testing.T) {
	data := &SystemData{
		AnimParams: AnimationParams{
			Appearance: AppearanceParams{
				StartScale: 1,
				EndScale:   1,
				ScaleNoise: NoiseParams{Enabled: true, Amplitude: 0.5, Frequency: 1, Octaves: 2},
			},
		},
	}
	p := &Instance{
		StartScale:      1,
		EndScale:        1,
		ScaleEasing:     EasingLinear,
		NoisePhaseScale: 0.3,
	}

	got0 := resolveParticleScale(data, p, 0.1, 0.1)
	got1 := resolveParticleScale(data, p, 0.2, 0.2)
	if almostEqualFloat32Turb(got0, 1, 1e-4) && almostEqualFloat32Turb(got1, 1, 1e-4) {
		t.Fatal("expected noise-adjusted scale to differ from base value")
	}
	if almostEqualFloat32Turb(got0, got1, 1e-4) {
		t.Fatal("expected continuous noise to vary over time")
	}
}

func TestResolveParticleAlphaNoiseWorksWithSequence(t *testing.T) {
	data := &SystemData{
		AlphaSeq: NewSequenceConfig([]SequenceStep{{FromBase: 0.5, ToBase: 0.5, Duration: 1, Easing: EasingLinear}}),
		AnimParams: AnimationParams{
			Appearance: AppearanceParams{
				AlphaNoise: NoiseParams{Enabled: true, Amplitude: 0.25, Frequency: 1, Octaves: 1},
			},
		},
	}
	p := &Instance{
		HasAlphaSeq:     true,
		AlphaSnap:       SequenceSnapshot{Values: []float32{0.5, 0.5}},
		NoisePhaseAlpha: 1.2,
	}

	got := resolveParticleAlpha(data, p, 0.25, 0.25)
	if almostEqualFloat32Turb(got, 0.5, 1e-4) {
		t.Fatal("expected alpha sequence value to receive noise overlay")
	}
}

func TestEvaluateParticlePositionAppliesPositionNoise(t *testing.T) {
	data := &SystemData{
		AnimParams: AnimationParams{
			Position: PositionParams{
				Easing:         EasingLinear,
				PositionNoiseX: NoiseParams{Enabled: true, Amplitude: 10, Frequency: 1, Octaves: 2},
				PositionNoiseY: NoiseParams{Enabled: true, Amplitude: 6, Frequency: 0.7, Octaves: 1},
			},
		},
	}
	p := &Instance{
		Duration:       1,
		StartX:         0,
		EndX:           100,
		StartY:         0,
		EndY:           0,
		PositionEasing: EasingLinear,
		NoisePhasePosX: 0.8,
		NoisePhasePosY: 1.4,
	}

	x, y := evaluateParticlePosition(data, p, 0.5, 0.5)
	if almostEqualFloat32Turb(x, 50, 1e-4) && almostEqualFloat32Turb(y, 0, 1e-4) {
		t.Fatal("expected position noise to alter base path")
	}
}

func almostEqualFloat32Turb(a, b, epsilon float32) bool {
	return float32(math.Abs(float64(a-b))) <= epsilon
}
