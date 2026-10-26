package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io"
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

func TestBlockCellSplitsLitFromDark(t *testing.T) {
	w, k := rgb{200, 200, 200}, rgb{}
	if got := blockCell([]rgb{w, w, k, k}, quadrants); !strings.HasSuffix(got, "▀") || strings.Contains(got, "48;2") {
		t.Errorf("lit top over dark bottom: got %q, want ▀ on the terminal's background", got)
	}
	if got := blockCell([]rgb{k, w, w, k}, quadrants); !strings.HasSuffix(got, "▞") {
		t.Errorf("lit diagonal: got %q, want ▞", got)
	}
	if got := blockCell([]rgb{k, k, k, k}, quadrants); got != " " {
		t.Errorf("dark cell: got %q, want a space", got)
	}
}

func TestKittyImageRoundTrips(t *testing.T) {
	pix := make([]byte, 64*40*4)
	for i := range pix {
		pix[i] = byte(i * 7)
	}
	out := kittyImage(pix, 64, 40, 16, 5)
	var b64 strings.Builder
	for _, cmd := range strings.Split(out, "\x1b\\")[:strings.Count(out, "\x1b\\")] {
		_, payload, _ := strings.Cut(strings.TrimPrefix(cmd, "\x1b_G"), ";")
		if len(payload) > 4096 {
			t.Fatalf("chunk of %d bytes, the protocol caps them at 4096", len(payload))
		}
		b64.WriteString(payload)
	}
	z, err := base64.StdEncoding.DecodeString(b64.String())
	if err != nil {
		t.Fatal(err)
	}
	r, err := zlib.NewReader(bytes.NewReader(z))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(r)
	if !bytes.Equal(got, pix) {
		t.Error("decoded pixels differ from the ones sent")
	}
	if strings.Count(out, "m=0;") != 1 {
		t.Error("the last chunk must say m=0")
	}
}

func TestOctantsCoverEveryPattern(t *testing.T) {
	seen := map[rune]bool{}
	for _, r := range octants {
		if seen[r] {
			t.Fatalf("%U drawn for two patterns", r)
		}
		seen[r] = true
	}
	if last := octants[254]; last != 0x1CDE5 {
		t.Errorf("the last octant should be U+1CDE5, got %U: the table is off", last)
	}
}
