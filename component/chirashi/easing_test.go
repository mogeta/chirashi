package chirashi

import "testing"

func TestParseEasingFallbackAndCaseInsensitive(t *testing.T) {
	if got := ParseEasing("InOutQuad"); got != EasingInOutQuad {
		t.Fatalf("expected InOutQuad, got %v", got)
	}

	if got := ParseEasing("inoutquad"); got != EasingInOutQuad {
		t.Fatalf("expected inoutquad to parse case-insensitively, got %v", got)
	}

	if got := ParseEasing("not-an-easing"); got != EasingLinear {
		t.Fatalf("expected unknown easing to fallback to Linear, got %v", got)
	}
}

func TestApplyEasingRepresentativeValues(t *testing.T) {
	tests := []struct {
		name   string
		t      float32
		easing EasingType
		want   float32
	}{
		{name: "linear", t: 0.5, easing: EasingLinear, want: 0.5},
		{name: "outquad", t: 0.5, easing: EasingOutQuad, want: 0.75},
		{name: "inquad", t: 0.5, easing: EasingInQuad, want: 0.25},
		{name: "insine at 0", t: 0.0, easing: EasingInSine, want: 0.0},
		{name: "outsine at 1", t: 1.0, easing: EasingOutSine, want: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyEasing(tt.t, tt.easing)
			if !almostEqualFloat32(got, tt.want, 1e-4) {
				t.Fatalf("got %.6f, want %.6f", got, tt.want)
			}
		})
	}
}

func almostEqualFloat32(a, b, eps float32) bool {
	if a > b {
		return a-b <= eps
	}
	return b-a <= eps
}
