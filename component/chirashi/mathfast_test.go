package chirashi

import (
	"math"
	"testing"
)

func TestFastSincosAccuracy(t *testing.T) {
	const tolerance = 5e-6
	var maxErr float64
	for x := float32(-200); x <= 200; x += 0.0037 {
		s, c := fastSincos(x)
		refS, refC := math.Sincos(float64(x))
		errS := math.Abs(float64(s) - refS)
		errC := math.Abs(float64(c) - refC)
		if errS > maxErr {
			maxErr = errS
		}
		if errC > maxErr {
			maxErr = errC
		}
		if errS > tolerance || errC > tolerance {
			t.Fatalf("fastSincos(%v) = (%v, %v), want (%v, %v)", x, s, c, refS, refC)
		}
	}
	t.Logf("max abs error: %.3g", maxErr)
}

func TestFastSincosSpecialAngles(t *testing.T) {
	for _, x := range []float32{0, math.Pi / 2, math.Pi, 3 * math.Pi / 2, 2 * math.Pi, -math.Pi / 2, -math.Pi} {
		s, c := fastSincos(x)
		refS, refC := math.Sincos(float64(x))
		if math.Abs(float64(s)-refS) > 5e-6 || math.Abs(float64(c)-refC) > 5e-6 {
			t.Errorf("fastSincos(%v) = (%v, %v), want (%v, %v)", x, s, c, refS, refC)
		}
	}
}

func BenchmarkFastSincos(b *testing.B) {
	var sink float32
	for i := 0; i < b.N; i++ {
		s, c := fastSincos(float32(i) * 0.01)
		sink += s + c
	}
	_ = sink
}

func BenchmarkMathSincos(b *testing.B) {
	var sink float32
	for i := 0; i < b.N; i++ {
		s, c := math.Sincos(float64(i) * 0.01)
		sink += float32(s) + float32(c)
	}
	_ = sink
}
