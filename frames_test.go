package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// writeFrame draws a grey disc of the given brightness on transparency.
func writeFrame(t *testing.T, dir, name string, grey uint8) {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			if (x-32)*(x-32)+(y-32)*(y-32) < 24*24 {
				img.Set(x, y, color.NRGBA{grey, grey, grey, 255})
			}
		}
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestFramesBlendBetweenAngles(t *testing.T) {
	dir := t.TempDir()
	writeFrame(t, dir, "neutral-y000-p0.png", 230)
	writeFrame(t, dir, "neutral-y180-p0.png", 40)
	writeFrame(t, dir, "frown-y000-p0.png", 40)
	writeFrame(t, dir, "frown-y180-p0.png", 40)
	f, err := loadFrames(dir)
	if err != nil {
		t.Fatal(err)
	}
	front, _ := f.at(0.5, 0.5, 0, 0, 0, 0)
	back, _ := f.at(0.5, 0.5, math.Pi, 0, 0, 0)
	side, _ := f.at(0.5, 0.5, math.Pi/2, 0, 0, 0)
	if front <= back {
		t.Fatalf("front %.2f should be brighter than back %.2f", front, back)
	}
	if mid := (front + back) / 2; math.Abs(side-mid) > 0.02 {
		t.Errorf("a quarter turn should blend the two frames: got %.2f, want %.2f", side, mid)
	}
	if frowned, _ := f.at(0.5, 0.5, 0, 0, 1, 0); math.Abs(frowned-back) > 0.02 {
		t.Errorf("a full frown should read the frown frames: got %.2f, want %.2f", frowned, back)
	}
}

func TestFramesRefuseAnIncompleteExpression(t *testing.T) {
	dir := t.TempDir()
	writeFrame(t, dir, "neutral-y000-p0.png", 200)
	writeFrame(t, dir, "neutral-y180-p0.png", 200)
	writeFrame(t, dir, "frown-y000-p0.png", 200)
	if _, err := loadFrames(dir); err == nil {
		t.Error("a frown missing its y180 frame should be refused")
	}
}
