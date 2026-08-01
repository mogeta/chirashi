package chirashi

import "testing"

func newFlowBenchData(particleCount int) (*System, *SystemData) {
	sys := &System{}
	data := &SystemData{
		ParticlePool: make([]Instance, particleCount),
		MaxParticles: particleCount,
		IsLoop:       true,
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1e9},
			Position: PositionParams{
				HasFlow:         true,
				FlowStrengthMin: 24,
				FlowStrengthMax: 24,
				FlowScale:       160,
				FlowOctaves:     2,
				FlowPersistence: 0.5,
				FlowTimeScale:   0.3,
				FlowDrag:        0.96,
				Easing:          EasingLinear,
			},
		},
	}
	for i := range data.ParticlePool {
		p := &data.ParticlePool[i]
		p.Active = true
		p.Duration = 1e9
		p.StartX = float32(i % 100)
		p.StartY = float32(i / 100)
		p.EndX = p.StartX + 100
		p.EndY = p.StartY + 100
		p.HasFlow = true
		p.FlowGain = 24
		p.FlowSeedX = float32(i) * 0.13
		p.FlowSeedY = float32(-i) * 0.07
	}
	data.ActiveCount = particleCount
	return sys, data
}

func BenchmarkUpdateParticlesFlow1000(b *testing.B) {
	sys, data := newFlowBenchData(1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.CurrentTime += defaultDeltaTime
		sys.updateParticles(data, defaultDeltaTime)
	}
}

func BenchmarkUpdateParticlesFlow10000(b *testing.B) {
	sys, data := newFlowBenchData(10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.CurrentTime += defaultDeltaTime
		sys.updateParticles(data, defaultDeltaTime)
	}
}

func BenchmarkSpawnWithSequences(b *testing.B) {
	seq := NewSequenceConfig([]SequenceStep{
		{ToBase: 100, ToRange: 20, Duration: 0.5, Easing: EasingOutQuad},
		{FromBase: 100, ToBase: 0, Duration: 0.5, Easing: EasingInQuad},
	})
	sys := &System{}
	data := &SystemData{
		ParticlePool:      make([]Instance, 256),
		MaxParticles:      256,
		SpawnInterval:     1,
		ParticlesPerSpawn: 256,
		EmissionScale:     1,
		IsLoop:            true,
		PosXSeq:           seq,
		PosYSeq:           seq,
		ScaleSeq:          seq,
		RotSeq:            seq,
		AlphaSeq:          seq,
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 1},
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.ActiveCount = 0
		sys.cnt = 0
		sys.spawn(data)
	}
}

func BenchmarkEvaluateSequenceManySteps(b *testing.B) {
	steps := make([]SequenceStep, 16)
	for i := range steps {
		steps[i] = SequenceStep{ToBase: float32(i), Duration: 0.1, Easing: EasingLinear}
	}
	seq := NewSequenceConfig(steps)
	snap := GenerateSnapshot(seq, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Late elapsed forces a scan deep into the step list.
		EvaluateSequence(seq, &snap, 1.55)
	}
}

// BenchmarkChurnFlow exercises steady-state spawn/expire churn, where pool
// fragmentation used to scatter particle memory accesses.
func BenchmarkChurnFlow4000(b *testing.B) {
	const maxParticles = 4000
	sys := &System{}
	data := &SystemData{
		ParticlePool:      make([]Instance, maxParticles),
		MaxParticles:      maxParticles,
		SpawnInterval:     1,
		ParticlesPerSpawn: 60,
		EmissionScale:     1,
		IsLoop:            true,
		AnimParams: AnimationParams{
			Duration: DurationParams{Base: 0.9, Range: 0.5},
			Position: PositionParams{
				UsePolar:         true,
				UsePolarVelocity: true,
				AngleMax:         6.28,
				SpeedMin:         40,
				SpeedMax:         120,
				HasFlow:          true,
				FlowStrengthMin:  24,
				FlowStrengthMax:  24,
				FlowScale:        160,
				FlowOctaves:      2,
				FlowPersistence:  0.5,
				FlowTimeScale:    0.3,
				FlowDrag:         0.96,
				Easing:           EasingLinear,
			},
			Appearance: AppearanceParams{StartScale: 1, EndScale: 1},
		},
	}
	// Reach steady state (spawn/expire equilibrium) before measuring.
	for i := 0; i < 240; i++ {
		sys.cnt++
		data.CurrentTime += defaultDeltaTime
		sys.spawn(data)
		sys.updateParticles(data, defaultDeltaTime)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sys.cnt++
		data.CurrentTime += defaultDeltaTime
		sys.spawn(data)
		sys.updateParticles(data, defaultDeltaTime)
	}
}
