package chirashi

import (
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
		EmitterX:          100,
		EmitterY:          200,
		AnimParams: AnimationParams{
			DurationBase: 1.0,
			StartScale:   1.0,
			EndScale:     1.0,
			StartR:       1, StartG: 1, StartB: 1,
			EndR: 1, EndG: 1, EndB: 1,
			PositionEasing: EasingLinear,
			AlphaEasing:    EasingLinear,
			ScaleEasing:    EasingLinear,
			RotationEasing: EasingLinear,
			ColorEasing:    EasingLinear,
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
