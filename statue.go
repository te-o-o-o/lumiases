package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"sort"
)

// statueParams set the statue's effect for a state: roll tilts the head,
// glitch slides slices sideways, ripple sends waves from the centre, sweep
// runs a band of light across, rise turns the top of the head into smoke.
type statueParams struct {
	roll, glitch, ripple, sweep, rise float64
}

var stateStatue = map[string]statueParams{
	"idle":     {},
	"thinking": {roll: 0.08, rise: 1},
	"reading":  {sweep: 1},
	"editing":  {glitch: 0.4},
	"running":  {ripple: 1},
	"waiting":  {},
	"error":    {glitch: 1},
	"done":     {sweep: 0.8},
}

func (a statueParams) ease(b statueParams, k float64) statueParams {
	return statueParams{
		roll: lerp(a.roll, b.roll, k), glitch: lerp(a.glitch, b.glitch, k), ripple: lerp(a.ripple, b.ripple, k),
		sweep: lerp(a.sweep, b.sweep, k), rise: lerp(a.rise, b.rise, k),
	}
}

// faceImg is a statue photo the matter flows over (statue mode, -face).
type faceImg struct {
	w, h      int
	lum, mask []float64
}

// face is set once at start in statue mode, nil otherwise.
var face *faceImg

// loadFace reads a photo, keeps the crop box (fractions of the picture),
// shrinks it to at most 360 px and keys the statue out of its background.
// ponytail: the key is warm stone against greenery (red above green); a
// statue on another background needs another key or a mask file.
func loadFace(path string, crop [4]float64) (*faceImg, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	b := img.Bounds()
	x0, y0 := b.Min.X+int(crop[0]*float64(b.Dx())), b.Min.Y+int(crop[1]*float64(b.Dy()))
	x1, y1 := b.Min.X+int(crop[2]*float64(b.Dx())), b.Min.Y+int(crop[3]*float64(b.Dy()))
	if x1-x0 < 8 || y1-y0 < 8 {
		return nil, fmt.Errorf("%s: the crop box keeps too little of the picture", path)
	}
	step := max(1, max(x1-x0, y1-y0)/360)
	fi := &faceImg{w: (x1 - x0) / step, h: (y1 - y0) / step}
	fi.lum, fi.mask = make([]float64, fi.w*fi.h), make([]float64, fi.w*fi.h)
	var stone []float64
	for j := 0; j < fi.h; j++ {
		for i := 0; i < fi.w; i++ {
			var r, g, bl float64 // box average of the step×step block
			for dy := 0; dy < step; dy++ {
				for dx := 0; dx < step; dx++ {
					cr, cg, cb, _ := img.At(x0+i*step+dx, y0+j*step+dy).RGBA()
					r, g, bl = r+float64(cr), g+float64(cg), bl+float64(cb)
				}
			}
			n := float64(step*step) * 65535
			r, g, bl = r/n, g/n, bl/n
			k := j*fi.w + i
			fi.lum[k] = 0.299*r + 0.587*g + 0.114*bl
			fi.mask[k] = smoothstep(0.04, 0.12, r-g)
			if fi.mask[k] > 0.5 {
				stone = append(stone, fi.lum[k])
			}
		}
	}
	if len(stone) < 100 {
		return nil, fmt.Errorf("%s: no statue found (the key looks for warm stone on a green background)", path)
	}
	// Stretch the stone's own range to 0..1, ignoring the 2% extremes.
	sort.Float64s(stone)
	lo, hi := stone[len(stone)/50], stone[len(stone)*49/50]
	for k, l := range fi.lum {
		fi.lum[k] = math.Max(0, math.Min(1, (l-lo)/(hi-lo+1e-9)))
	}
	// Unsharp mask: a terminal has few cells for a face, so the features
	// (eyes, nose, lips) need their local contrast pushed or they melt away.
	blur := boxBlur(fi.lum, fi.w, fi.h, 5)
	for k, l := range fi.lum {
		fi.lum[k] = math.Max(0, math.Min(1, l+1.3*(l-blur[k])))
	}
	return fi, nil
}

// boxBlur averages each pixel over a (2r+1)² square, clamped at the borders.
func boxBlur(s []float64, w, h, r int) []float64 {
	pass := func(src []float64, dx, dy int) []float64 {
		out := make([]float64, len(src))
		for j := 0; j < h; j++ {
			for i := 0; i < w; i++ {
				sum, n := 0.0, 0.0
				for k := -r; k <= r; k++ {
					x, y := i+k*dx, j+k*dy
					if x >= 0 && x < w && y >= 0 && y < h {
						sum, n = sum+src[y*w+x], n+1
					}
				}
				out[j*w+i] = sum / n
			}
		}
		return out
	}
	return pass(pass(s, 1, 0), 0, 1)
}

// at reads luminance and mask at u, v (0..1 across the picture), bilinear.
func (f *faceImg) at(u, v float64) (float64, float64) {
	x, y := u*float64(f.w-1), v*float64(f.h-1)
	if x < 0 || y < 0 || x >= float64(f.w-1) || y >= float64(f.h-1) {
		return 0, 0
	}
	i, j := int(x), int(y)
	fx, fy := x-float64(i), y-float64(j)
	read := func(s []float64) float64 {
		k := j*f.w + i
		return lerp(lerp(s[k], s[k+1], fx), lerp(s[k+f.w], s[k+f.w+1], fx), fy)
	}
	return read(f.lum), read(f.mask)
}

// sampleFace is sample for statue mode: the features hold still while the
// matter churns over them in the state's colours.
func sampleFace(x, y, t, ph float64, p params) (rgb, float64, float64) {
	const H = 2.6 // picture height in orb units; the view spans about ±1.5, the rest is room to sway
	a := float64(face.w) / float64(face.h)
	sx, sy := x, y // screen position, for effects that stay put (light bands)
	st := p.statue

	// Every effect moves where the photo is read (x, y), never the photo.
	// Life: breathing, and a head that sways around the neck and nods.
	s := 1 + 0.012*math.Sin(t*1.3) + 0.3*p.pulse*math.Sin(4*t)
	x, y = x/s, y/s
	roll := 0.03*math.Sin(t*0.5) + st.roll*math.Sin(t*0.7)
	cr, sr := math.Cos(roll), math.Sin(roll)
	x, y = cr*x-sr*(y-1.2), sr*x+cr*(y-1.2)+1.2 // pivot at the neck
	y += 0.02 * math.Sin(t*0.4)
	x += p.shake * math.Sin(t*53)
	y += p.shake * 0.6 * math.Sin(t*41)
	// Ripple: rings travelling out from the centre.
	if r := math.Hypot(x, y); st.ripple > 0 && r > 0 {
		w := st.ripple * 0.035 * math.Sin(r*12-t*6)
		x, y = x+w*x/r, y+w*y/r
	}
	// Glitch: thin horizontal slices jump sideways, re-rolled 8 times a second.
	if st.glitch > 0 {
		band, tick := int(math.Floor((y+2)*14)), int(t*8)
		if hash(band, tick) > 1-0.35*st.glitch {
			x += (hash(band, tick+977) - 0.5) * 0.35 * st.glitch
		}
	}
	// Rise: above the brow, the head is read from lower down along noisy
	// streaks, so the hair trails upward like smoke.
	if st.rise > 0 && y < -0.3 {
		y += st.rise * (-y - 0.3) * 0.6 * fbm(x*4, y*2+ph*0.8)
	}

	// Noise nudges where the photo is read: the stone seems to flow while
	// the features stay put.
	qx, qy := fbm(x*2+ph*0.3, y*2), fbm(x*2-5.2, y*2+ph*0.25)
	u := (x+p.warp*0.1*(qx-0.5))/(H*a) + 0.5
	v := (y+p.warp*0.1*(qy-0.5))/H + 0.5
	l, m := face.at(u, v)
	// Sweep: a diagonal band of light crossing the face, then a pause.
	if st.sweep > 0 {
		pos := math.Mod(ph*0.6, 4) - 2
		l = math.Min(1, l+st.sweep*0.7*smoothstep(0.12, 0, math.Abs(sx+0.4*sy-pos)))
	}
	// The silhouette crumbles into dust instead of a clean cut-out.
	m = smoothstep(0.3, 0.7, m+0.3*(fbm(u*9, v*9-ph*0.4)-0.5))
	if m < 0.01 {
		return p.edge, 0, 2
	}
	var c rgb
	if l < 0.5 {
		c = lerpC(p.edge, p.mid, l/0.5)
	} else {
		c = lerpC(p.mid, p.center, (l-0.5)/0.5)
	}
	tex := fbm(x*6+qx*2, y*6-ph*0.5)
	// Lighter texture than the orb's: it should flow over the face, not hide it.
	return c, math.Min(1, l*(0.9+0.2*(tex-0.5))*m), 0.5
}
