package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"math"
	"os"
	"path/filepath"
	"sort"
)

// gifClip is an animated GIF, each frame already composited over the ones
// before (a GIF frame is often only the patch that changed) and shrunk.
type gifClip struct {
	w, h   int
	frames [][]uint8 // RGBA, w*h*4 each
	ends   []float64 // when each frame ends, seconds from the start
}

// clipSide caps the stored size: a pane is a few hundred pixels across
// in a terminal's image, and a long GIF at full size would take hundreds
// of MB.
const clipSide = 480

func loadClip(path string) (*gifClip, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		return nil, err
	}
	W, H := g.Config.Width, g.Config.Height
	step := (max(W, H) + clipSide - 1) / clipSide
	c := &gifClip{w: W / step, h: H / step}
	canvas := image.NewRGBA(image.Rect(0, 0, W, H))
	end := 0.0
	for i, fr := range g.Image {
		var prev *image.RGBA
		if g.Disposal[i] == gif.DisposalPrevious {
			prev = image.NewRGBA(canvas.Rect)
			copy(prev.Pix, canvas.Pix)
		}
		draw.Draw(canvas, fr.Bounds(), fr, fr.Bounds().Min, draw.Over)
		px := make([]uint8, c.w*c.h*4)
		for y := 0; y < c.h; y++ {
			for x := 0; x < c.w; x++ {
				copy(px[(y*c.w+x)*4:], canvas.Pix[canvas.PixOffset(x*step, y*step):][:4])
			}
		}
		// Browsers play a delay of 0 or 1 (hundredths) as 0.1 s; so do we.
		d := g.Delay[i]
		if d < 2 {
			d = 10
		}
		end += float64(d) / 100
		c.frames, c.ends = append(c.frames, px), append(c.ends, end)
		switch g.Disposal[i] {
		case gif.DisposalBackground:
			draw.Draw(canvas, fr.Bounds(), image.Transparent, image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			canvas = prev
		}
	}
	return c, nil
}

// introClips loads the GIFs in dir, in name order; a missing dir is no intro.
func introClips(dir string) ([]*gifClip, error) {
	names, _ := filepath.Glob(filepath.Join(dir, "*.gif"))
	sort.Strings(names)
	var out []*gifClip
	for _, n := range names {
		c, err := loadClip(n)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// introDir is where the intro GIFs live unless -intro says otherwise.
func introDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "lumiases", "intro")
}

// length is how long the clip plays in the intro: once, looped up to at
// least 2.5 s so a short GIF still registers.
func (c *gifClip) length() float64 {
	d := c.ends[len(c.ends)-1]
	return math.Ceil(2.5/d) * d
}

// frameAt is the frame showing t seconds in, looping.
func (c *gifClip) frameAt(t float64) []uint8 {
	t = math.Mod(t, c.ends[len(c.ends)-1])
	i := sort.SearchFloat64s(c.ends, t)
	return c.frames[min(i, len(c.frames)-1)]
}

// show draws one frame as a real picture through the kitty graphics
// protocol (Ghostty, kitty, WezTerm), fitted and centred in the w×h pane,
// through the fx filter.
func (c *gifClip) show(px []uint8, w, h int, fx string) string {
	// A cell is about twice as tall as it is wide.
	a := float64(c.w) / float64(c.h)
	cols, rows := int(math.Round(float64(h)*2*a)), h
	if cols > w {
		cols, rows = w, max(1, int(math.Round(float64(w)/(2*a))))
	}
	return fmt.Sprintf("\x1b[?2026h\x1b[2J\x1b[%d;%dH", (h-rows)/2+1, (w-cols)/2+1) +
		kittyImage(filter(px, c.w, c.h, fx), c.w, c.h, cols, rows) + "\x1b[?2026l"
}

// filters are the looks a clip can take: contrast pushes the tones apart
// and the colours up; digital adds posterized, dithered colour, scanlines
// and a red/blue split, like a cheap screen.
var filters = []string{"contrast", "digital", "none"}

// bayer is a 4×4 ordered dither, thresholds in 0..1.
var bayer = [16]float64{0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5}

func filter(px []uint8, w, h int, fx string) []uint8 {
	out := make([]uint8, len(px))
	copy(out, px)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			o := (y*w + x) * 4
			r, g, b := float64(px[o])/255, float64(px[o+1])/255, float64(px[o+2])/255
			switch fx {
			case "contrast":
				l := 0.3*r + 0.59*g + 0.11*b
				// The curve acts on light alone and the colour rides along:
				// curving each channel apart oversaturates a dithered GIF.
				k := lerp(l, smoothstep(0.05, 0.95, l), 0.8) / (l + 1e-6)
				r, g, b = k*(l+1.15*(r-l)), k*(l+1.15*(g-l)), k*(l+1.15*(b-l))
			case "digital":
				// The split: red read a pixel to the left, blue to the right.
				r = float64(px[(y*w+max(0, x-1))*4]) / 255
				b = float64(px[(y*w+min(w-1, x+1))*4+2]) / 255
				d := (bayer[(y%4)*4+x%4]+0.5)/16 - 0.5
				const lv = 3 // steps per channel above black
				r, g, b = math.Round(r*lv+d)/lv, math.Round(g*lv+d)/lv, math.Round(b*lv+d)/lv
				if y%2 == 1 {
					r, g, b = r*0.55, g*0.55, b*0.55
				}
			}
			to8 := func(v float64) byte { return byte(math.Round(255 * math.Max(0, math.Min(1, v)))) }
			out[o], out[o+1], out[o+2] = to8(r), to8(g), to8(b)
		}
	}
	return out
}
