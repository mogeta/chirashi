package chirashi

import "math/rand"

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
}

// GenerateSnapshot creates a randomized snapshot for one particle
func GenerateSnapshot(config *SequenceConfig, baseValue float32) SequenceSnapshot {
	values := make([]float32, len(config.Steps)*2)
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

		values[i*2] = from
		values[i*2+1] = to

		// Next step's base starts from this step's to value
		currentBase = to
	}

	return SequenceSnapshot{Values: values}
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

	// Find which step we're in
	accumulated := float32(0)
	for i, step := range config.Steps {
		if elapsed < accumulated+step.Duration {
			// We're in this step
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
