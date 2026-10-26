package main

import "math"

// forms are the shapes the agent can take, drawn from code alone, no
// picture: the orb, a swarm of dots, and line drawings in the manner of
// scientific plots. The scenes read the state through p.mood, each state's
// weight eased like every other parameter, so a change of state morphs.

// Indexes into params.mood, in the order of states.
const (
	mIdle = iota
	mThinking
	mReading
	mEditing
	mRunning
	mWaiting
	mError
	mDone
)

var forms = []string{"orb", "swarm", "pulsar", "scope", "field", "topo", "waves", "spectrum", "orbits", "flow"}

// form is set from main between frames, read by the row goroutines.
var form = "orb"

func sq(x float64) float64 { return x * x }

// swarm is a flock of dots riding one slow flow, drawing a figure per
// state: a cloud, a spiral, a stream, a ring.
func swarm(x, y, t, ph float64, p params) (rgb, float64, float64) {
	m := p.mood
	s := p.size / 0.8
	x, y = x/s, y/s
	r, a := math.Hypot(x, y), math.Atan2(y, x)

	// What the flock draws, as a density in 0..1, blended by state.
	cloud := smoothstep(1, 0.3, math.Hypot(x/0.95, y/0.6)+0.35*(fbm(x*2+ph*0.3, y*2)-0.5))
	spiral := math.Pow(0.5+0.5*math.Cos(2*(a-math.Log(r+0.05)*2.5-ph*0.8)), 3) * smoothstep(1.1, 0.15, r)
	stream := math.Exp(-sq(y/0.22)) * smoothstep(1.4, 0.9, math.Abs(x))
	ring := math.Exp(-sq((r - 0.6) / 0.12))
	dash := smoothstep(1, 0.3, math.Hypot(x/1.2, y/0.28))
	pulse := math.Exp(-sq((r - 0.55*(1+0.15*math.Sin(4*t))) / 0.1))
	scatter := 0.22 * smoothstep(1.5, 0.4, r)
	R := 0.3 + 1.2*p.statue.blast
	burst := math.Exp(-sq((r-R)/0.15)) * (1 - 0.5*p.statue.blast)
	dens := m[mIdle]*cloud + m[mThinking]*spiral + m[mReading]*stream + m[mEditing]*ring +
		m[mRunning]*dash + m[mWaiting]*pulse + m[mError]*scatter + m[mDone]*burst

	// The dots live in a frame that turns, slides and billows, so they
	// move as one flock instead of each on its own.
	rot := ph * (0.08 + 0.5*m[mThinking])
	c, sn := math.Cos(rot), math.Sin(rot)
	qx, qy := c*x+sn*y-ph*(0.6*m[mReading]+0.9*m[mRunning]), -sn*x+c*y
	stir := 0.25 * (1 + 2*m[mError])
	qx, qy = qx+stir*(fbm(qx*1.3+ph*0.2, qy*1.3)-0.5), qy+stir*(fbm(qx*1.3-4, qy*1.3+ph*0.2)-0.5)

	// One dot per cell of that frame, fluttering inside it, shown when its
	// own draw falls under the density.
	const g = 0.045
	ci, cj := int(math.Floor(qx/g)), int(math.Floor(qy/g))
	h1, h2 := hash(ci, cj+31), hash(ci+57, cj)
	ox := 0.5 + 0.28*math.Sin(t*(1+h1)+h2*6)
	oy := 0.5 + 0.28*math.Cos(t*(1+h2)+h1*6)
	d := math.Hypot(qx/g-float64(ci)-ox, qy/g-float64(cj)-oy) * g
	v := 0.12 * dens // a faint haze, so the figure reads between the dots
	if hash(ci, cj+999) < dens {
		v = math.Max(v, smoothstep(0.02, 0.007, d)*(0.7+0.3*dens))
	}
	return lerpC(p.mid, p.center, dens), math.Min(1, v), 0.5
}

// pulsar is a stack of ridge lines, each hiding the ones behind it, like
// the plot of a radio pulsar: calm when idle, a wave rolling back while
// thinking, a scanned bump while reading, a ripple when done.
func pulsar(x, y, t, ph float64, p params) (rgb, float64, float64) {
	s := p.size / 0.8
	x, y = x/s+p.shake*math.Sin(t*53), y/s
	if math.Abs(x) > 1.05 {
		return p.edge, 0, 0.5
	}
	const n, hmax, th = 30, 0.8, 0.018
	// Front (bottom) line first: a point under a ridge is hidden by it.
	for k := n - 1; k >= 0; k-- {
		base := -0.95 + 2.1*float64(k)/(n-1)
		if base-hmax > y+th { // this ridge never rises up to the point
			continue
		}
		if base < y-th { // the point is under this ridge, whatever its height
			return p.edge, 0, 0.5
		}
		h := ridge(x, float64(k), n, t, ph, p)
		yk := base - h
		if d := math.Abs(y - yk); d < th*1.6 {
			v := smoothstep(th*1.6, th*0.4, d) * smoothstep(1.03, 0.95, math.Abs(x)) * (0.7 + 0.3*math.Min(1, h*3))
			return lerpC(p.mid, p.center, math.Min(1, h*2.5)), v, 0.5
		}
		if y > yk {
			return p.edge, 0, 0.5
		}
	}
	return p.edge, 0, 0.5
}

// ridge is the height of line k at x.
func ridge(x, k float64, n int, t, ph float64, p params) float64 {
	m := p.mood
	cx := m[mReading] * 0.55 * math.Sin(t*1.2)
	env := math.Exp(-sq((x - cx) / 0.38))
	h := env * math.Pow(fbm(x*5+k*1.7, k*0.37+ph*0.6), 2.2) * (0.3 + 0.25*m[mEditing] + 0.15*m[mRunning])
	h += 0.012 * (fbm(x*20+k, ph) - 0.5)
	// Thinking: a crest rolls from the front line to the back one.
	crest := float64(n-1) * (1 - math.Mod(ph*0.25, 1))
	h += m[mThinking] * 0.35 * env * math.Exp(-sq((k-crest)/2.5))
	h *= 1 + m[mWaiting]*0.6*math.Sin(4*t)
	h += m[mError] * 0.12 * env * (noise(x*14, k+t*25) - 0.5)
	rho := math.Hypot(x, (k-float64(n)/2)/float64(n)*2)
	h += m[mDone] * 0.35 * math.Exp(-sq((rho-1.5*p.statue.blast)/0.12)) * (1 - 0.6*p.statue.blast)
	return math.Max(0, h)
}

// lw is the half-width of a drawn line, the pulsar's thickness.
const lw = 0.016

// stroke lights a line from its distance d, lw wide unless w says otherwise.
func stroke(d, w float64) float64 { return smoothstep(w*1.6, w*0.4, d) }

// iso is the distance to the nearest isoline of a field f drawn every sp,
// given the length of its gradient: lines keep one width however steep.
func iso(f, sp, grad float64) float64 {
	k := f / sp
	return math.Abs(k-math.Round(k)) * sp / (grad + 1e-9)
}

// dominant is the state with the most weight, for scenes that cannot
// blend two figures and switch between them instead.
func dominant(m [8]float64) int {
	best := 0
	for i := range m {
		if m[i] > m[best] {
			best = i
		}
	}
	return best
}

// field draws the field lines between two poles: they close in while
// thinking, one sweeps while reading, the lines flow round while running,
// break up on error and fan out from one pole when done.
func field(x, y, t, ph float64, p params) (rgb, float64, float64) {
	m := p.mood
	s := p.size / 0.8
	x, y = x/s+p.shake*math.Sin(t*53), y/s
	r := math.Hypot(x, y)
	if r > 1.3 {
		return p.edge, 0, 0.5
	}
	sep := 0.45 - 0.27*m[mThinking] + 0.12*m[mWaiting]*math.Sin(4*t) + 0.1*m[mEditing]*math.Sin(ph*0.7)
	rot := ph * (0.12 + 0.6*m[mThinking])
	c, sn := math.Cos(rot)*sep, math.Sin(rot)*sep
	x1, y1 := c+m[mReading]*0.5*math.Sin(t*1.2), sn
	x2, y2 := -c, -sn
	q2 := -(1 - m[mDone])
	// The lines are isolines of the summed angles round each pole; the
	// gradient is exact, so the cut where atan2 jumps never shows.
	d1x, d1y, d2x, d2y := x-x1, y-y1, x-x2, y-y2
	r1, r2 := d1x*d1x+d1y*d1y, d2x*d2x+d2y*d2y
	psi := math.Atan2(d1y, d1x) + q2*math.Atan2(d2y, d2x) + ph*0.6*m[mRunning]
	gx, gy := -d1y/r1-q2*d2y/r2, d1x/r1+q2*d2x/r2
	v := stroke(iso(psi, 2*math.Pi/14, math.Hypot(gx, gy)), lw*0.7)
	v *= 1 - m[mError]*(1-smoothstep(0.4, 0.5, noise(x*9+t*20, y*9)))
	v *= smoothstep(1.25, 0.75, r)
	pole := math.Exp(-r1/0.004) + math.Abs(q2)*math.Exp(-r2/0.004)
	v = math.Max(v*0.85, pole)
	return lerpC(p.mid, p.center, math.Min(1, pole+0.4*math.Exp(-math.Min(r1, r2)*6))), math.Min(1, v), 0.5
}

// topo is a contour map of a breathing relief, every fifth line heavier:
// a peak rises while thinking and the lines climb it, a bump runs across
// while reading, a crater opens on error, rings spread when done.
func topo(x, y, t, ph float64, p params) (rgb, float64, float64) {
	m := p.mood
	s := p.size / 0.8
	x, y = x/s+p.shake*math.Sin(t*53), y/s
	r := math.Hypot(x, y)
	if r > 1.3 {
		return p.edge, 0, 0.5
	}
	bx := 0.7 * math.Sin(t*1.2)
	h := func(x, y float64) float64 {
		r2 := x*x + y*y
		// Two broad octaves only: fbm's fine ones crowd the map with islets.
		xo := x + ph*0.04*(1+4*m[mEditing])
		v := (0.65*noise(xo*1.1+5, y*1.1) + 0.35*noise(xo*2.3, y*2.3+3)) * (1 + 0.15*math.Sin(t*0.8) + 0.3*m[mWaiting]*math.Sin(4*t))
		v += m[mThinking]*0.7*math.Exp(-r2/0.12) + m[mReading]*0.5*math.Exp(-((x-bx)*(x-bx)+y*y)/0.05)
		v += m[mError] * (0.35*math.Exp(-sq((math.Sqrt(r2)-0.35)/0.1)) - 0.6*math.Exp(-r2/0.06))
		return lerp(v, 0.5+0.12*math.Cos(math.Sqrt(r2)*14-ph*4), m[mDone])
	}
	const e = 0.01
	h0 := h(x, y)
	g := math.Hypot(h(x+e, y)-h0, h(x, y+e)-h0) / e
	f := h0 - ph*0.04*(m[mThinking]+m[mRunning]) // contours climb
	const sp = 0.05
	w := lw * 0.6
	if int(math.Round(f/sp))%5 == 0 {
		w = lw
	}
	v := stroke(iso(f, sp, g), w) * smoothstep(1.2, 0.85, r+0.2*(fbm(x*2, y*2)-0.5))
	return lerpC(p.edge, p.center, math.Min(1, h0*1.2)), v, 0.5
}

// scope is an oscilloscope tracing a Lissajous figure with a fading
// afterglow: a circle when idle, knots that turn over while thinking, a
// fast wave while reading, a chopped signal on error.
func scope(x, y, t, ph float64, p params) (rgb, float64, float64) {
	m := p.mood
	s := p.size / 0.8
	x, y = x/s, y/s+m[mError]*0.1*(noise(x*30, t*40)-0.5)
	if math.Abs(x) > 1.15 || math.Abs(y) > 1.15 {
		return p.edge, 0, 0.5
	}
	figs := [8][2]float64{{1, 1}, {3, 2}, {1, 4}, {2, 3}, {5, 4}, {1, 1}, {1, 1}, {1, 2}}
	ab := figs[dominant(m)]
	a, b := ab[0], ab[1]
	A := 0.85 * (1 + 0.12*m[mWaiting]*math.Sin(4*t) + 0.1*p.statue.blast)
	dl := math.Pi/2 + ph*(0.05+0.5*m[mThinking]+0.3*m[mRunning]+0.4*m[mReading])
	// Closest point of the curve: the parameters where it crosses x, then
	// two Newton steps on the squared distance from there.
	best, bs := math.Inf(1), 0.0
	u := math.Asin(math.Max(-1, math.Min(1, x/A)))
	for k := 0.0; k < a; k++ {
		for _, s0 := range [2]float64{(u - dl + 2*math.Pi*k) / a, (math.Pi - u - dl + 2*math.Pi*k) / a} {
			for i := 0; i < 2; i++ {
				X, Y := A*math.Sin(a*s0+dl)-x, A*math.Sin(b*s0)-y
				X1, Y1 := A*a*math.Cos(a*s0+dl), A*b*math.Cos(b*s0)
				X2, Y2 := -a*a*(X+x), -b*b*(Y+y)
				if d2 := X1*X1 + Y1*Y1 + X*X2 + Y*Y2; d2 > 1e-6 {
					s0 -= (X*X1 + Y*Y1) / d2
				}
			}
			if d := math.Hypot(A*math.Sin(a*s0+dl)-x, A*math.Sin(b*s0)-y); d < best {
				best, bs = d, s0
			}
		}
	}
	behind := math.Mod(math.Mod(ph*1.5-bs, 2*math.Pi)+2*math.Pi, 2*math.Pi)
	glow := 0.55 + 0.45*math.Exp(-behind*1.2)
	v := math.Max(stroke(best, lw*1.2), 0.35*math.Exp(-best/0.025)) * glow
	// A faint graticule of dots, as on the glass.
	gx, gy := x/0.25-math.Round(x/0.25), y/0.25-math.Round(y/0.25)
	if math.Abs(x) < 1.01 && math.Abs(y) < 1.01 {
		v = math.Max(v, 0.22*smoothstep(0.1, 0.05, math.Hypot(gx, gy)))
	}
	return lerpC(p.mid, p.center, glow*smoothstep(0.05, 0, best)), math.Min(1, v), 0.5
}

// waves are the crests of ripples from a few sources, crossing into
// interference fringes: three sources turning while thinking, one moving
// while reading, a scatter on error, one wide ring when done.
func waves(x, y, t, ph float64, p params) (rgb, float64, float64) {
	m := p.mood
	s := p.size / 0.8
	x, y = x/s+p.shake*math.Sin(t*53), y/s
	r := math.Hypot(x, y)
	if r > 1.3 {
		return p.edge, 0, 0.5
	}
	k, om := 18+10*m[mThinking], ph*3*(1+m[mRunning])
	sum, ws := 0.0, 1e-9
	add := func(sx, sy, w float64) {
		if w > 0.01 {
			sum += w * math.Cos(k*math.Hypot(x-sx, y-sy)-om)
			ws += w
		}
	}
	add(0.35, 0, m[mIdle]+m[mRunning])
	add(-0.35, 0, m[mIdle]+m[mRunning])
	for i := 0.0; i < 3; i++ {
		a := ph*0.6 + i*2*math.Pi/3
		add(0.3*math.Cos(a), 0.3*math.Sin(a), m[mThinking])
	}
	add(0.6*math.Sin(t*1.2), 0, m[mReading])
	add(0.15, 0, m[mEditing])
	add(-0.15, 0, m[mEditing])
	add(0, 0, (m[mWaiting]*(0.3+0.7*math.Max(0, math.Sin(2*t))))+m[mDone])
	tick := int(t * 5)
	for i := 0; i < 4; i++ {
		add(hash(i, tick)*1.2-0.6, hash(i+9, tick)*1.2-0.6, m[mError])
	}
	v := math.Pow(0.5+0.5*sum/ws, 6)
	v = math.Max(v, m[mDone]*math.Exp(-sq((r-1.2*p.statue.blast)/0.05))*(1-0.5*p.statue.blast))
	v *= smoothstep(1.25, 0.8, r)
	return lerpC(p.mid, p.center, v), v, 0.5
}

// spectrum is a mirrored row of level bars: quiet when idle, the middle
// bands pulsing while thinking, chatter while editing and running, a flat
// line with a blip while waiting, clipping on error, a swell when done.
func spectrum(x, y, t, ph float64, p params) (rgb, float64, float64) {
	m := p.mood
	s := p.size / 0.8
	x, y = x/s+p.shake*math.Sin(t*53), y/s
	if math.Abs(x) > 1 || math.Abs(y) > 1 {
		return p.edge, 0, 0.5
	}
	const nb = 25
	fi := (x + 1) / 2 * nb
	i := math.Floor(fi)
	fx := fi - i
	c := i/(nb-1)*2 - 1
	b := p.statue.blast
	lv := m[mIdle]*(0.06+0.08*noise(c*3, ph*0.8)) +
		m[mThinking]*0.6*math.Exp(-c*c/0.12)*(0.6+0.4*math.Sin(ph*3+c*4)) +
		m[mReading]*(0.05+0.55*math.Exp(-sq((c-0.8*math.Sin(t*1.2))/0.2))) +
		m[mEditing]*0.7*math.Pow(noise(c*4, ph*4), 1.5) +
		m[mRunning]*0.8*math.Pow(noise(c*5, ph*6), 1.2) +
		m[mWaiting]*(0.02+0.6*math.Exp(-math.Mod(t, 1.6)*6)*math.Exp(-c*c/0.02)) +
		m[mError]*0.85*math.Floor(hash(int(i), int(t*12))+0.5) +
		m[mDone]*0.8*math.Exp(-sq((math.Abs(c)-b)/0.25))*(1-0.7*b)
	lv *= 0.85
	ay := math.Abs(y)
	v := smoothstep(lv+0.015, lv-0.015, ay) * smoothstep(0.12, 0.22, fx) * smoothstep(0.88, 0.78, fx)
	v = math.Max(v, 0.5*smoothstep(0.012, 0.004, ay))
	return lerpC(p.mid, p.center, math.Min(1, ay/0.6)), v, 0.5
}

// orbits are bodies on ellipses round a small sun, each trailing its own
// path: faster while thinking, the planes knocked askew on error, all
// bodies falling into line when done.
func orbits(x, y, t, ph float64, p params) (rgb, float64, float64) {
	m := p.mood
	s := p.size / 0.8
	x, y = x/s+p.shake*math.Sin(t*53), y/s
	if math.Hypot(x, y) > 1.2 {
		return p.edge, 0, 0.5
	}
	sun := 0.07 * (1 + 0.5*m[mWaiting]*math.Sin(4*t))
	v := math.Exp(-(x*x + y*y) / (sun * sun))
	col := p.center
	for i := 0; i < 5; i++ {
		fi := float64(i)
		a := 0.25 + 0.17*fi
		b := a * (0.45 + 0.35*m[mEditing]*(0.5+0.5*math.Sin(ph*0.8+fi)))
		rot := -0.35 + 0.1*fi + m[mReading]*0.4*math.Sin(t*1.2) +
			m[mError]*(1.5*(hash(i, 5)-0.5)+0.2*math.Sin(t*23+fi))
		cr, sr := math.Cos(rot), math.Sin(rot)
		u, w := cr*x+sr*y, -sr*x+cr*y
		k := math.Hypot(u/a, w/b)
		g := math.Hypot(u/(a*a*k), w/(b*b*k))
		d := math.Abs(k-1) / (g + 1e-9)
		// Inner bodies run faster, as Kepler has it; done brings them in line.
		tb := ph*(0.5+1.5*m[mThinking]+m[mRunning])/math.Pow(a, 1.5) + hash(i, 3)*2*math.Pi
		tb += m[mDone] * math.Remainder(math.Pi/2-tb, 2*math.Pi)
		th := math.Atan2(w/b, u/a)
		behind := math.Mod(math.Mod(tb-th, 2*math.Pi)+2*math.Pi, 2*math.Pi)
		v = math.Max(v, stroke(d, lw*0.6)*(0.2+0.75*math.Exp(-behind*1.5)))
		if bd := math.Hypot(u-a*math.Cos(tb), w-b*math.Sin(tb)); bd < 0.05 {
			v, col = math.Max(v, smoothstep(0.05, 0.025, bd)), p.center
		}
	}
	return lerpC(p.mid, col, math.Min(1, v)), math.Min(1, v), 0.5
}

// flow is a field of short strokes, one per cell, drifting along a slow
// current: a vortex while thinking, a stream while reading, drawn inward
// while editing, scrambled on error, blown outward when done.
func flow(x, y, t, ph float64, p params) (rgb, float64, float64) {
	m := p.mood
	s := p.size / 0.8
	x, y = x/s, y/s
	r := math.Hypot(x, y)
	if r > 1.25 {
		return p.edge, 0, 0.5
	}
	const g = 0.075
	ci, cj := math.Floor(x/g), math.Floor(y/g)
	ii, jj := int(ci), int(cj)
	cx, cy := (ci+0.5)*g, (cj+0.5)*g
	cr := math.Hypot(cx, cy) + 1e-9
	n := noise(cx*1.3+ph*0.05, cy*1.3) * 4 * math.Pi
	e := hash(ii, jj+int(t*7)) * 2 * math.Pi
	br := math.Sin(2 * t)
	dx := (m[mIdle]+0.5*m[mRunning])*math.Cos(n) - m[mThinking]*cy/cr + m[mReading] + m[mRunning] -
		m[mEditing]*cx/cr + m[mWaiting]*br*cx/cr + m[mError]*math.Cos(e) + m[mDone]*cx/cr
	dy := (m[mIdle]+0.5*m[mRunning])*math.Sin(n) + m[mThinking]*cx/cr + m[mReading]*0.3*math.Sin(cx*4+ph) -
		m[mEditing]*cy/cr + m[mWaiting]*br*cy/cr + m[mError]*math.Sin(e) + m[mDone]*cy/cr
	l := math.Hypot(dx, dy) + 1e-9
	dx, dy = dx/l, dy/l
	// Each stroke slides along its cell's current and fades at both ends of
	// its run, so the whole field seems to stream.
	run := math.Mod(ph*0.5*(1+2*m[mRunning])+hash(ii, jj), 1)
	off := (run - 0.5) * g * 0.3
	px, py := cx+dx*off, cy+dy*off
	const L = g * 0.42
	along := math.Max(-L, math.Min(L, (x-px)*dx+(y-py)*dy))
	d := math.Hypot(x-px-along*dx, y-py-along*dy)
	v := stroke(d, 0.011) * (0.3 + 0.7*math.Sin(math.Pi*run)) * smoothstep(1.2, 0.7, r)
	return lerpC(p.mid, p.center, hash(ii, jj+7)), v, 0.5
}
