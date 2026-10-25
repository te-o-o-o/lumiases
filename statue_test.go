package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writePNG draws a w×h picture, pixel by pixel from paint, into a temp file.
func writePNG(t *testing.T, w, h int, paint func(x, y int) color.Color) string {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, paint(x, y))
		}
	}
	path := filepath.Join(t.TempDir(), "statue.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadFaceKeysWarmStoneOnGreen(t *testing.T) {
	green, stone := color.RGBA{60, 140, 50, 255}, color.RGBA{210, 140, 100, 255}
	path := writePNG(t, 200, 200, func(x, y int) color.Color {
		if (x-100)*(x-100)+(y-100)*(y-100) < 60*60 {
			return stone
		}
		return green
	})
	f, err := loadFace(path, [4]float64{0, 0, 1, 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, m := f.at(0.5, 0.5); m < 0.9 {
		t.Errorf("centre should be statue, mask = %.2f", m)
	}
	if _, m := f.at(0.05, 0.05); m > 0.1 {
		t.Errorf("corner should be background, mask = %.2f", m)
	}
}

func TestLoadFaceRefusesAPictureWithoutStatue(t *testing.T) {
	path := writePNG(t, 64, 64, func(x, y int) color.Color { return color.RGBA{60, 140, 50, 255} })
	if _, err := loadFace(path, [4]float64{0, 0, 1, 1}); err == nil {
		t.Error("an all-green picture should be refused")
	}
	if _, err := loadFace(filepath.Join(t.TempDir(), "missing.png"), [4]float64{0, 0, 1, 1}); err == nil {
		t.Error("a missing file should be refused")
	}
}

func TestEveryStateHasAStatueEffect(t *testing.T) {
	for _, s := range states {
		if _, ok := stateStatue[s.name]; !ok {
			t.Errorf("state %q has no entry in stateStatue", s.name)
		}
	}
}
