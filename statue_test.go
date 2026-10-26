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
	f, err := loadFace(path, [4]float64{0, 0, 1, 1}, nil)
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

func TestLoadFaceKeysAPlainBackgroundAndInvertsInk(t *testing.T) {
	white, ink := color.RGBA{250, 250, 250, 255}, color.RGBA{30, 30, 30, 255}
	path := writePNG(t, 200, 200, func(x, y int) color.Color {
		if y > 185 { // a watermark band along one edge
			return ink
		}
		if (x-100)*(x-100)+(y-100)*(y-100) < 50*50 {
			return ink
		}
		return white
	})
	f, err := loadFace(path, [4]float64{0, 0, 1, 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if l, m := f.at(0.5, 0.5); m < 0.9 || l < 0.5 {
		t.Errorf("the ink should be the subject, and lit: mask %.2f, light %.2f", m, l)
	}
	if _, m := f.at(0.05, 0.05); m > 0.1 {
		t.Errorf("the white page should be background, mask = %.2f", m)
	}
}

func TestLoadFaceRefusesAPictureWithoutStatue(t *testing.T) {
	path := writePNG(t, 64, 64, func(x, y int) color.Color { return color.RGBA{60, 140, 50, 255} })
	if _, err := loadFace(path, [4]float64{0, 0, 1, 1}, nil); err == nil {
		t.Error("an all-green picture should be refused")
	}
	if _, err := loadFace(filepath.Join(t.TempDir(), "missing.png"), [4]float64{0, 0, 1, 1}, nil); err == nil {
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

func TestLoadFaceMeasuresEachRow(t *testing.T) {
	path := writePNG(t, 200, 200, func(x, y int) color.Color {
		if (x-100)*(x-100)+(y-100)*(y-100) < 60*60 {
			return color.RGBA{210, 140, 100, 255}
		}
		return color.RGBA{60, 140, 50, 255}
	})
	f, err := loadFace(path, [4]float64{0, 0, 1, 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	mid := f.h / 2
	if a := f.axis[mid]; a < 0.45 || a > 0.55 {
		t.Errorf("axis of the middle row = %.2f, want about 0.5", a)
	}
	if h := f.half[mid]; h < 0.25 || h > 0.35 {
		t.Errorf("half-width of the middle row = %.2f, want about 0.3", h)
	}
}

func TestHalfTurnShowsTheMirror(t *testing.T) {
	// A disc of stone, bright on its left half, dark on its right.
	path := writePNG(t, 200, 200, func(x, y int) color.Color {
		switch {
		case (x-100)*(x-100)+(y-100)*(y-100) >= 80*80:
			return color.RGBA{60, 140, 50, 255}
		case x < 100:
			return color.RGBA{230, 160, 120, 255}
		default:
			return color.RGBA{110, 50, 30, 255}
		}
	})
	f, err := loadFace(path, [4]float64{0, 0, 1, 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	face = &head{photo: f}
	defer func() { face = nil }()
	var p params
	p.mid, p.center = rgb{200, 200, 200}, rgb{255, 255, 255}
	_, front, _ := sampleFace(-0.4, 0, 0, 0, p)
	p.statue.angle = 3.14159
	_, back, _ := sampleFace(-0.4, 0, 0, 0, p)
	if front < 0.5 || back > front/2 {
		t.Errorf("left of the screen: %.2f facing, %.2f after a half turn; the mirror should put the dark side there", front, back)
	}
}

func TestLoadFaceUsesTransparency(t *testing.T) {
	// Grey marble on a transparent background: the colour key would find
	// nothing, the alpha channel finds it all.
	path := writePNG(t, 200, 200, func(x, y int) color.Color {
		if (x-100)*(x-100)+(y-100)*(y-100) < 60*60 {
			return color.NRGBA{uint8(150 + x/4), uint8(150 + x/4), uint8(150 + x/4), 255}
		}
		return color.NRGBA{0, 0, 0, 0}
	})
	f, err := loadFace(path, [4]float64{0, 0, 1, 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, m := f.at(0.5, 0.5); m < 0.99 {
		t.Errorf("centre should be statue, mask = %.2f", m)
	}
	if _, m := f.at(0.05, 0.05); m > 0.01 {
		t.Errorf("corner should be background, mask = %.2f", m)
	}
}
