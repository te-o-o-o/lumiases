package main

import (
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadClipCompositesPatchesAndTimesFrames(t *testing.T) {
	pal := color.Palette{color.Transparent, color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}}
	full := image.NewPaletted(image.Rect(0, 0, 4, 4), pal)
	for i := range full.Pix {
		full.Pix[i] = 1 // all red
	}
	patch := image.NewPaletted(image.Rect(0, 0, 2, 2), pal)
	for i := range patch.Pix {
		patch.Pix[i] = 2 // blue top-left corner only
	}
	path := filepath.Join(t.TempDir(), "a.gif")
	f, _ := os.Create(path)
	err := gif.EncodeAll(f, &gif.GIF{Image: []*image.Paletted{full, patch}, Delay: []int{0, 50},
		Config: image.Config{ColorModel: pal, Width: 4, Height: 4}})
	f.Close()
	if err != nil {
		t.Fatal(err)
	}

	c, err := loadClip(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.ends[0] != 0.1 || c.ends[1] != 0.6 {
		t.Errorf("ends = %v, want [0.1 0.6] (a 0 delay plays as 0.1 s)", c.ends)
	}
	second := c.frameAt(0.3)
	if second[0] != 0 || second[2] != 255 {
		t.Errorf("patched corner should be blue, got %v", second[:4])
	}
	if px := second[(3*4+3)*4:]; px[0] != 255 {
		t.Errorf("the rest should keep the first frame's red, got %v", px[:4])
	}
	if c.length() < 2.5 {
		t.Errorf("a short clip should loop to at least 2.5 s, got %.2f", c.length())
	}
}
