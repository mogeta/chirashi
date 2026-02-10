package chirashi

import "testing"

func TestNewSequenceConfigComputesTotalDuration(t *testing.T) {
	cfg := NewSequenceConfig([]SequenceStep{
		{Duration: 0.4},
		{Duration: 0.6},
		{Duration: 1.0},
	})

	if !almostEqualFloat32(cfg.TotalDuration, 2.0, 1e-6) {
		t.Fatalf("total duration got %.4f, want 2.0", cfg.TotalDuration)
	}
}

func TestGenerateSnapshotDeterministicWithoutRanges(t *testing.T) {
	cfg := NewSequenceConfig([]SequenceStep{
		{FromBase: 1, ToBase: 3, Duration: 1, Easing: EasingLinear},
		{FromBase: 3, ToBase: 4, Duration: 1, Easing: EasingLinear},
	})

	snap := GenerateSnapshot(cfg, 0)
	if len(snap.Values) != 4 {
		t.Fatalf("snapshot value length got %d, want 4", len(snap.Values))
	}

	want := []float32{1, 3, 6, 7}
	for i := range want {
		if !almostEqualFloat32(snap.Values[i], want[i], 1e-6) {
			t.Fatalf("snapshot[%d] got %.4f, want %.4f", i, snap.Values[i], want[i])
		}
	}
}

func TestEvaluateSequenceBoundaries(t *testing.T) {
	cfg := NewSequenceConfig([]SequenceStep{
		{FromBase: 0, ToBase: 10, Duration: 1.0, Easing: EasingLinear},
		{FromBase: 10, ToBase: 20, Duration: 1.0, Easing: EasingLinear},
	})
	snap := SequenceSnapshot{Values: []float32{0, 10, 20, 30}}

	if got := EvaluateSequence(cfg, &snap, -0.1); !almostEqualFloat32(got, 0, 1e-6) {
		t.Fatalf("elapsed<0 got %.4f, want 0", got)
	}
	if got := EvaluateSequence(cfg, &snap, 0.5); !almostEqualFloat32(got, 5, 1e-6) {
		t.Fatalf("elapsed=0.5 got %.4f, want 5", got)
	}
	if got := EvaluateSequence(cfg, &snap, 1.5); !almostEqualFloat32(got, 25, 1e-6) {
		t.Fatalf("elapsed=1.5 got %.4f, want 25", got)
	}
	if got := EvaluateSequence(cfg, &snap, 2.1); !almostEqualFloat32(got, 30, 1e-6) {
		t.Fatalf("elapsed>total got %.4f, want 30", got)
	}
}
