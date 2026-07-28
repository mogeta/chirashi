package chirashi

import "math/rand/v2"

// SequenceStep defines one step in a multi-step animation
type SequenceStep struct {
	FromBase  float32
	FromRange float32 // ±randomization applied at spawn
	ToBase    float32
	ToRange   float32 // ±randomization applied at spawn
	Duration  float32
	Easing    EasingType
}

// SequenceConfig defines a multi-step animation sequence (shared across particles)
type SequenceConfig struct {
	Steps         []SequenceStep
	TotalDuration float32 // Sum of step durations (precomputed)
}

// SequenceSnapshot stores per-particle randomized from/to values
// Layout: [from0, to0, from1, to1, ...]
type SequenceSnapshot struct {
	Values []float32

	// Step cursor: elapsed time is monotonic within a particle's life, so
	// evaluation resumes from the last step instead of scanning from step 0.
	stepIdx   int
	stepStart float32 // Accumulated duration of steps before stepIdx
}

// GenerateSnapshot creates a randomized snapshot for one particle
func GenerateSnapshot(config *SequenceConfig, baseValue float32) SequenceSnapshot {
	var snap SequenceSnapshot
	FillSnapshot(config, &snap, baseValue)
	return snap
}

// FillSnapshot randomizes snap in place for one particle, reusing the existing
// Values slice when possible so respawning does not allocate.
func FillSnapshot(config *SequenceConfig, snap *SequenceSnapshot, baseValue float32) {
	need := len(config.Steps) * 2
	if cap(snap.Values) < need {
		snap.Values = make([]float32, need)
	} else {
		snap.Values = snap.Values[:need]
	}
	snap.stepIdx = 0
	snap.stepStart = 0
	currentBase := baseValue

	for i, step := range config.Steps {
		from := currentBase + step.FromBase
		if step.FromRange > 0 {
			from += (rand.Float32()*2 - 1) * step.FromRange
		}

		to := currentBase + step.ToBase
		if step.ToRange > 0 {
			to += (rand.Float32()*2 - 1) * step.ToRange
		}

		snap.Values[i*2] = from
		snap.Values[i*2+1] = to

		// Next step's base starts from this step's to value
		currentBase = to
	}
}

// EvaluateSequence evaluates a multi-step sequence at the given elapsed time
// Returns the interpolated value for the current step
func EvaluateSequence(config *SequenceConfig, snap *SequenceSnapshot, elapsed float32) float32 {
	if len(config.Steps) == 0 {
		return 0
	}

	// Clamp elapsed to total duration
	if elapsed >= config.TotalDuration {
		// Return final value
		lastIdx := len(config.Steps) - 1
		return snap.Values[lastIdx*2+1]
	}
	if elapsed <= 0 {
		return snap.Values[0]
	}

	// Resume from the cached step cursor; reset if it is stale (config swapped
	// via live update, or time moved backwards).
	i := snap.stepIdx
	accumulated := snap.stepStart
	if i >= len(config.Steps) || i*2+1 >= len(snap.Values) || elapsed < accumulated {
		i = 0
		accumulated = 0
	}
	for ; i < len(config.Steps); i++ {
		step := &config.Steps[i]
		if elapsed < accumulated+step.Duration {
			snap.stepIdx = i
			snap.stepStart = accumulated

			stepElapsed := elapsed - accumulated
			normalizedT := stepElapsed / step.Duration
			easedT := ApplyEasing(normalizedT, step.Easing)

			from := snap.Values[i*2]
			to := snap.Values[i*2+1]
			return from + (to-from)*easedT
		}
		accumulated += step.Duration
	}

	// Shouldn't reach here, but return final value
	lastIdx := len(config.Steps) - 1
	return snap.Values[lastIdx*2+1]
}

// NewSequenceConfig creates a SequenceConfig from steps, precomputing TotalDuration
func NewSequenceConfig(steps []SequenceStep) *SequenceConfig {
	total := float32(0)
	for _, s := range steps {
		total += s.Duration
	}
	return &SequenceConfig{
		Steps:         steps,
		TotalDuration: total,
	}
}
