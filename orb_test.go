package main

import "testing"

func TestSampleInsideAndOutside(t *testing.T) {
	for _, s := range states {
		if _, i, d := sample(0, 0, 1, 1, s.p); d >= 1 || i <= 0 {
			t.Errorf("%s: centre should be lit inside the orb, got brightness %.2f at distance %.2f", s.name, i, d)
		}
		if _, i, _ := sample(5, 5, 1, 1, s.p); i > 0.001 {
			t.Errorf("%s: far outside should be dark, got %.3f", s.name, i)
		}
	}
}

func TestEaseReachesTarget(t *testing.T) {
	a, b := states[0].p, states[6].p
	if got := ease(a, b, 1); got != b {
		t.Errorf("ease(k=1) = %+v, want %+v", got, b)
	}
	if got := ease(a, b, 0); got != a {
		t.Errorf("ease(k=0) = %+v, want %+v", got, a)
	}
}

func TestDitherInRange(t *testing.T) {
	for x := 0; x < 64; x++ {
		for y := 0; y < 64; y++ {
			if v := dither(x, y); v < 0 || v >= 1 {
				t.Fatalf("dither(%d, %d) = %v, want [0, 1)", x, y, v)
			}
		}
	}
}
