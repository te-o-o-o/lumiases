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
// runs a band of light across, rise turns the top of the head into smoke,
// look turns the head left and right (radians), spin turns it all the way
// round (radians a second), blast > 0 makes it burst once in place.
// angle (where the spin has got to) and the blast's progress are kept by
// main, not eased like the rest.
type statueParams struct {
	roll, glitch, ripple, sweep, rise, look, spin, angle, blast float64
}

var stateStatue = map[string]statueParams{
	"idle":     {look: 0.3},
	"thinking": {roll: 0.08, rise: 1},
	"reading":  {sweep: 1, look: 0.55},
	"editing":  {glitch: 0.4},
	"running":  {ripple: 1},
	"waiting":  {},
	"error":    {glitch: 1},
	"done":     {sweep: 0.8, blast: 1},
}

func (a statueParams) ease(b statueParams, k float64) statueParams {
	return statueParams{
		roll: lerp(a.roll, b.roll, k), glitch: lerp(a.glitch, b.glitch, k), ripple: lerp(a.ripple, b.ripple, k),
		sweep: lerp(a.sweep, b.sweep, k), rise: lerp(a.rise, b.rise, k), look: lerp(a.look, b.look, k),
		spin: lerp(a.spin, b.spin, k), angle: a.angle, blast: a.blast,
	}
}

// faceImg is a statue photo the matter flows over (statue mode, -face).
type faceImg struct {
	w, h      int
	lum, mask []float64
	// axis and half are, per row, the middle and half-width of the statue as
	// fractions of the picture's width: the bust turns as a stack of
	// cylinders, one per row, so the neck stays narrow and the hair wide.
	axis, half []float64
}

// face is the head being drawn, nil for the orb; main switches it between
// frames.
var face *faceImg

// loadFace reads a photo, keeps the crop box (fractions of the picture),
// shrinks it to at most 360 px and tells the statue from its background:
// by transparency when the picture has some (a cut-out PNG), else by a
// colour key that only knows warm stone against greenery (red above green).
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
	opaque, ok := img.(interface{ Opaque() bool })
	cutOut := ok && !opaque.Opaque()
	step := max(1, max(x1-x0, y1-y0)/360)
	fi := &faceImg{w: (x1 - x0) / step, h: (y1 - y0) / step}
	fi.lum, fi.mask = make([]float64, fi.w*fi.h), make([]float64, fi.w*fi.h)
	var stone []float64
	for j := 0; j < fi.h; j++ {
		for i := 0; i < fi.w; i++ {
			var r, g, bl, al float64 // box sums of the step×step block
			for dy := 0; dy < step; dy++ {
				for dx := 0; dx < step; dx++ {
					cr, cg, cb, ca := img.At(x0+i*step+dx, y0+j*step+dy).RGBA()
					r, g, bl, al = r+float64(cr), g+float64(cg), bl+float64(cb), al+float64(ca)
				}
			}
			k := j*fi.w + i
			if cutOut {
				fi.mask[k] = al / (float64(step*step) * 65535)
				if al > 0 { // RGBA() is premultiplied: divide the alpha back out
					r, g, bl = r/al, g/al, bl/al
				}
			} else {
				n := float64(step*step) * 65535
				r, g, bl = r/n, g/n, bl/n
				fi.mask[k] = smoothstep(0.04, 0.12, r-g)
			}
			fi.lum[k] = 0.299*r + 0.587*g + 0.114*bl
			if fi.mask[k] > 0.5 {
				stone = append(stone, fi.lum[k])
			}
		}
	}
	if len(stone) < 100 {
		return nil, fmt.Errorf("%s: no statue found (give a PNG with the background removed, or warm stone on a green background)", path)
	}
	// Stretch the stone's own range to 0..1, ignoring the 2% extremes.
	sort.Float64s(stone)
	lo, hi := stone[len(stone)/50], stone[len(stone)*49/50]
	for k, l := range fi.lum {
		fi.lum[k] = math.Max(0, math.Min(1, (l-lo)/(hi-lo+1e-9)))
	}
	// Unsharp mask: a terminal has few cells for a face, so the features
	// (eyes, nose, lips) need their local contrast pushed or they melt away.
	blur := boxBlur(fi.lum, fi.w, fi.h, 7)
	for k, l := range fi.lum {
		l = math.Max(0, math.Min(1, l+2.2*(l-blur[k])))
		fi.lum[k] = l * l * (3 - 2*l) // S-curve: sockets and nostrils go dark
	}
	// Measure each row's width, smoothed over a few rows so a stray speck
	// of warm background doesn't make one cylinder jut out.
	fi.axis, fi.half = make([]float64, fi.h), make([]float64, fi.h)
	for j := 0; j < fi.h; j++ {
		lo, hi := -1, -1
		for i := 0; i < fi.w; i++ {
			if fi.mask[j*fi.w+i] > 0.5 {
				if lo < 0 {
					lo = i
				}
				hi = i
			}
		}
		fi.axis[j] = 0.5
		if lo >= 0 {
			fi.axis[j] = float64(lo+hi) / 2 / float64(fi.w-1)
			fi.half[j] = float64(hi-lo) / 2 / float64(fi.w-1)
		}
	}
	fi.axis, fi.half = boxBlur(fi.axis, 1, fi.h, 4), boxBlur(fi.half, 1, fi.h, 4)
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

// burst is how far the bust has blown apart, 0..1: fully in profile,
// where the photo has nothing true to show, and once in place when a blast
// runs (its progress goes 0 to 1, the burst peaking halfway).
func burst(t float64, st statueParams) float64 {
	yaw := st.look*math.Sin(t*0.35) + st.angle
	turn := math.Pow(smoothstep(0.7, 0.98, math.Abs(math.Sin(yaw))), 1.5)
	return math.Max(turn, 4*st.blast*(1-st.blast))
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
	// Burst: the screen is cut into shards, each reading the photo from
	// nearer the centre so its piece flies outward by its own amount, and
	// more of them are gone the closer the head is to profile.
	b := burst(t, st)
	if b > 0 {
		ci, cj := int(math.Floor(sx/0.14)), int(math.Floor(sy/0.14))
		if hash(ci+5000, cj) < 0.25*b {
			return p.edge, 0, 2
		}
		r := math.Hypot(x, y) + 1e-6
		push := b * (0.08 + 0.22*hash(ci, cj+7000))
		x, y = x-push*x/r, y-push*y/r
	}
	// Turn: each row of the photo is wrapped round its own vertical cylinder.
	// The back half shows the photo mirrored, which is what makes the seams
	// at ±90° meet column for column: past a quarter turn the far side comes
	// round as the face again, like a Janus bust, since no photo shows the
	// back of the head. Mirrored or not, the column read is R·sin(θ).
	shade, yaw := 1.0, st.look*math.Sin(t*0.35)+st.angle // the same yaw as burst
	if j := int((y/H + 0.5) * float64(face.h-1)); j >= 0 && j < face.h && face.half[j] > 0 {
		W := H * a
		X, R := (face.axis[j]-0.5)*W, face.half[j]*W*1.05
		if math.Abs(x-X) >= R {
			return p.edge, 0, 2
		}
		ts := math.Asin((x - X) / R)
		th := math.Remainder(ts-yaw, 2*math.Pi)
		// In profile the column read would be the photo's own rim, nearly
		// empty, and the head would look hollow: read further in as the
		// turn nears a quarter, back to the true column face on.
		x = X + R*math.Sin(th)*(1-0.2*math.Abs(math.Sin(yaw)))
		shade = 0.75 + 0.25*math.Cos(ts) // the sides of the cylinder turn away from the light
	}

	// Noise nudges where the photo is read: the stone seems to flow while
	// the features stay put.
	qx, qy := fbm(x*2+ph*0.3, y*2), fbm(x*2-5.2, y*2+ph*0.25)
	u := (x+p.warp*0.1*(qx-0.5))/(H*a) + 0.5
	v := (y+p.warp*0.1*(qy-0.5))/H + 0.5
	l, m := face.at(u, v)
	l *= shade
	// Sweep: a diagonal band of light crossing the face, then a pause.
	if st.sweep > 0 {
		pos := math.Mod(ph*0.6, 4) - 2
		l = math.Min(1, l+st.sweep*0.7*smoothstep(0.12, 0, math.Abs(sx+0.4*sy-pos)))
	}
	// The silhouette crumbles into dust instead of a clean cut-out, and the
	// shards of a burst crumble further, glowing with the blast.
	m = smoothstep(0.3, 0.7, m+0.3*(fbm(u*9, v*9-ph*0.4)-0.5)-0.4*b*fbm(u*14+ph, v*14))
	l = math.Min(1, l+0.15*b)
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
