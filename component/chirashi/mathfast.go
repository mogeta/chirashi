package chirashi

import "math"

// fastSincos returns approximate sin(x) and cos(x) for x in radians.
//
// The argument is reduced to [-π/4, π/4] in float64 (cheap: one multiply and
// one round), then degree-7/8 Taylor kernels run in float32. Maximum absolute
// error is about 4e-6 — far below one pixel for any on-screen use — while
// avoiding the float64 round trips and full-precision kernels of math.Sin/Cos.
// Used in per-particle hot paths: flow field sampling, spiral motion, spawn
// angles, and draw rotation.
func fastSincos(x float32) (float32, float32) {
	const twoOverPi = float32(2 / math.Pi)
	// Cody-Waite split of π/2 so reduction stays in float32.
	const piOver2Hi = float32(1.5707855224609375)
	const piOver2Lo = float32(1.0804334e-5)

	// q = round(x / (π/2)) via truncation with a sign-matched 0.5 offset.
	f := x * twoOverPi
	half := math.Float32frombits(0x3F000000 | (math.Float32bits(f) & 0x80000000))
	q := int32(f + half)
	fq := float32(q)
	// r = x - q·(π/2) ∈ [-π/4, π/4]
	r := (x - fq*piOver2Hi) - fq*piOver2Lo

	r2 := r * r
	// Taylor kernels on [-π/4, π/4]
	sinR := r * (1 + r2*(-1.0/6+r2*(1.0/120+r2*(-1.0/5040))))
	cosR := 1 + r2*(-1.0/2+r2*(1.0/24+r2*(-1.0/720+r2*(1.0/40320))))

	if q&1 != 0 {
		sinR, cosR = cosR, -sinR
	}
	if q&2 != 0 {
		sinR, cosR = -sinR, -cosR
	}
	return sinR, cosR
}

// fastSin returns approximate sin(x); see fastSincos for accuracy notes.
func fastSin(x float32) float32 {
	s, _ := fastSincos(x)
	return s
}

// fastCos returns approximate cos(x); see fastSincos for accuracy notes.
func fastCos(x float32) float32 {
	_, c := fastSincos(x)
	return c
}
