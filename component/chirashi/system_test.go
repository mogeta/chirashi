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
