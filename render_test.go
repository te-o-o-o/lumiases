package main

import (
	"strings"
	"testing"
)

func TestDitherInRange(t *testing.T) {
	for x := 0; x < 64; x++ {
		for y := 0; y < 64; y++ {
			if v := dither(x, y); v < 0 || v >= 1 {
				t.Fatalf("dither(%d, %d) = %v, want [0, 1)", x, y, v)
			}
		}
	}
}

func TestBestGlyphFollowsShape(t *testing.T) {
	cases := []struct {
		name string
		lit  [8]float64 // left, right per row, top row first
		want string     // any of these glyphs
	}{
		{"top lit", [8]float64{1, 1, 0, 0, 0, 0, 0, 0}, `"'^` + "`"},
		{"bottom lit", [8]float64{0, 0, 0, 0, 0, 0, 1, 1}, "_.,"},
		{"left lit", [8]float64{1, 0, 1, 0, 1, 0, 1, 0}, "([bL"},
		{"right lit", [8]float64{0, 1, 0, 1, 0, 1, 0, 1}, ")]dJ"},
		{"rising diagonal", [8]float64{0, 1, 0, 1, 1, 0, 1, 0}, "/"},
		{"empty", [8]float64{}, " "},
		{"full", [8]float64{1, 1, 1, 1, 1, 1, 1, 1}, "@"},
	}
	for _, c := range cases {
		if g := bestGlyph(c.lit); !strings.ContainsRune(c.want, g) {
			t.Errorf("%s: got %q, want one of %q", c.name, g, c.want)
		}
	}
}
