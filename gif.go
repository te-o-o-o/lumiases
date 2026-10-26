package main

import (
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

// clip, when set, is played instead of the orb, clipT seconds in. Both
// are set from main between frames, read by the row goroutines.
var (
	clip  *gifClip
	clipT float64
)

// clipSide caps the stored size: a terminal shows a few hundred dots
// across at most, and a long GIF at full size would take hundreds of MB.
const clipSide = 320

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

// sampleClip shows the clip fitted to the view like a photo, its colours
// as they are: the brightest channel is the light, the rest the hue.
func sampleClip(x, y float64) (rgb, float64, float64) {
	a := float64(clip.w) / float64(clip.h)
	H := 3 / math.Max(1, a)
	u, v := x/(H*a)+0.5, y/H+0.5
	if u < 0 || u >= 1 || v < 0 || v >= 1 {
		return rgb{}, 0, 2
	}
	px := clip.frameAt(clipT)[(int(v*float64(clip.h))*clip.w+int(u*float64(clip.w)))*4:]
	c := rgb{float64(px[0]), float64(px[1]), float64(px[2])}
	return vivid(c), math.Max(c.r, math.Max(c.g, c.b)) / 255 * float64(px[3]) / 255, 0.5
}
