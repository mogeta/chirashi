package chirashi

import "testing"

func TestNewBloomEffectCompilesShaders(t *testing.T) {
	bloom, err := NewBloomEffect()
	if err != nil {
		t.Fatalf("NewBloomEffect: %v", err)
	}
	if bloom.Threshold <= 0 || bloom.Intensity <= 0 || bloom.Passes < 1 {
		t.Errorf("unexpected defaults: threshold=%v intensity=%v passes=%d", bloom.Threshold, bloom.Intensity, bloom.Passes)
	}
}
