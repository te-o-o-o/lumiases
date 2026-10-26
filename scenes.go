package main

import "math"

// The scenes below are drawn from code alone, no picture: swarm, black hole,
// pulsar and mask. They read the state through p.mood, each state's weight
// eased like every other parameter, so a change of state morphs.

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

// blackHole is a disk of light seen nearly edge on, wrapped round a black
// shadow: the far side of the disk shows bent over the top, as lensing
// does, and the side turning toward us burns brighter.
func blackHole(x, y, t, ph float64, p params) (rgb, float64, float64) {
	m := p.mood
	s := p.size / 0.8 * (1 + p.pulse*math.Sin(4*t))
	x, y = x/s+p.shake*math.Sin(t*53), y/s+p.shake*0.6*math.Sin(t*41)
	const rs, tilt = 0.26, 0.22

	dv := y / tilt
	r, a := math.Hypot(x, dv), math.Atan2(dv, x)
	disk := 0.0
	if r > rs*1.3 && r < 1.4 {
		spin := ph * 1.2 / math.Pow(r, 1.5) // the inside turns faster
		swirl := fbm(math.Cos(a+spin)*r*3+7, math.Sin(a+spin)*r*3)
		swirl = math.Pow(swirl, 1+m[mEditing]) // streams of matter falling in
		doppler := 1 + 0.6*(-x/r)
		disk = smoothstep(rs*1.3, rs*2.2, r) * smoothstep(1.4, 0.5, r) * (0.4 + 0.9*swirl) * doppler * (1.1 - 0.5*r)
		// Reading: a hot spot orbits, scanning the disk.
		sa := ph * 2
		disk += m[mReading] * math.Exp(-(sq(x-0.7*math.Cos(sa))+sq(dv-0.7*math.Sin(sa)))/0.012)
	}
	rc := math.Hypot(x, y)
	var v float64
	switch {
	case rc < rs && y < 0: // the far disk is behind the shadow
	case rc < rs:
		v = disk // the near disk passes in front of it
	default:
		ac := math.Atan2(y, x)
		lens := math.Exp(-3*sq((rc-rs*1.5)/(rs*0.7))) * (0.5 + 0.6*fbm(math.Cos(ac+ph*0.8)*2, math.Sin(ac+ph*0.8)*2))
		lens *= 0.6 + 0.4*math.Abs(math.Cos(ac)) // brightest where it meets the disk
		ring := math.Exp(-sq((rc - rs*1.04) / 0.02))
		v = math.Max(disk, math.Max(1.1*lens, ring))
	}
	v *= 1 + 0.4*m[mThinking]
	if m[mError] > 0.01 { // the light stutters
		v *= 1 - 0.5*m[mError]*hash(int(t*20), 3)
	}
	// Done: jets shoot out of the poles.
	jet := m[mDone] * math.Exp(-math.Abs(x)/0.025) * smoothstep(rs, rs*1.6, math.Abs(y)) *
		math.Exp(-math.Abs(y)*0.9) * (0.5 + 0.5*fbm(y*6-t*8, x*20))
	v = math.Max(v, jet)
	return lerpC(lerpC(p.edge, p.mid, smoothstep(1.4, 0.4, r)), p.center, math.Min(1, v)), math.Min(1, v), 0.5
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

type v3 struct{ x, y, z float64 }

func (a v3) add(b v3) v3      { return v3{a.x + b.x, a.y + b.y, a.z + b.z} }
func (a v3) sub(b v3) v3      { return v3{a.x - b.x, a.y - b.y, a.z - b.z} }
func (a v3) mul(k float64) v3 { return v3{a.x * k, a.y * k, a.z * k} }
func (a v3) dot(b v3) float64 { return a.x*b.x + a.y*b.y + a.z*b.z }
func (a v3) length() float64  { return math.Sqrt(a.dot(a)) }
func (a v3) norm() v3         { return a.mul(1 / (a.length() + 1e-9)) }
func (a v3) div(b v3) v3      { return v3{a.x / b.x, a.y / b.y, a.z / b.z} }
func smin(a, b, k float64) float64 {
	h := math.Max(0, math.Min(1, 0.5+0.5*(b-a)/k))
	return lerp(b, a, h) - k*h*(1-h)
}
func smax(a, b, k float64) float64 { return -smin(-a, -b, k) }

// ellipsoid is the usual approximate distance to an ellipsoid at c.
func ellipsoid(q, c, r v3) float64 {
	d := q.sub(c)
	k0, k1 := d.div(r).length(), d.div(v3{r.x * r.x, r.y * r.y, r.z * r.z}).length()
	return k0 * (k0 - 1) / (k1 + 1e-9)
}

func capsule(q, a, b v3, r float64) float64 {
	pa, ba := q.sub(a), b.sub(a)
	h := math.Max(0, math.Min(1, pa.dot(ba)/ba.dot(ba)))
	return pa.sub(ba.mul(h)).length() - r
}

// maskSDF is a smooth, stylised head, facing +z with y down: soft shapes
// joined by smooth unions, sockets and mouth carved out. frown pulls the
// brows' inner ends down, smile bends the mouth's corners up.
func maskSDF(q v3, frown, smile float64) float64 {
	d := ellipsoid(q, v3{0, -0.18, -0.02}, v3{0.6, 0.75, 0.68})
	d = smin(d, ellipsoid(q, v3{0, 0.38, 0.12}, v3{0.4, 0.42, 0.48}), 0.3)
	d = smin(d, ellipsoid(q, v3{0, 0.68, 0.36}, v3{0.15, 0.1, 0.12}), 0.12)
	d = smin(d, capsule(q, v3{0, 0.6, -0.1}, v3{0, 1.3, -0.15}, 0.28), 0.15)
	d = smin(d, capsule(q, v3{0, -0.02, 0.66}, v3{0, 0.27, 0.76}, 0.055), 0.1)
	d = smin(d, ellipsoid(q, v3{0, 0.47, 0.58}, v3{0.17, 0.06, 0.08}), 0.06)
	for _, s := range []float64{-1, 1} {
		d = smin(d, ellipsoid(q, v3{s * 0.3, 0.18, 0.42}, v3{0.2, 0.18, 0.2}), 0.15)
		d = smin(d, ellipsoid(q, v3{s * 0.6, 0.02, -0.05}, v3{0.07, 0.17, 0.11}), 0.05)
		d = smin(d, capsule(q, v3{s * 0.38, -0.16, 0.55}, v3{s * 0.1, -0.14 + 0.07*frown, 0.63}, 0.05), 0.08)
		d = smax(d, -ellipsoid(q, v3{s * 0.22, 0, 0.68}, v3{0.14, 0.09, 0.15}), 0.04)
	}
	mq := q
	mq.y += smile * 1.2 * q.x * q.x
	return smax(d, -ellipsoid(mq, v3{0, 0.47, 0.6}, v3{0.16, 0.018, 0.12}), 0.03)
}

// mask is that head ray-marched in true 3D: it turns, nods, frowns and
// smiles with the statue parameters, under a fixed light, and its eyes
// glow from the back of the sockets.
func mask(x, y, t, ph float64, p params) (rgb, float64, float64) {
	st, e := p.statue, p.eyes
	s := p.size / 0.8 * 1.3
	x, y = x/s, y/s
	if math.Abs(x) > 0.95 || y < -1.05 || y > 1.8 {
		return p.edge, 0, 2
	}
	yaw := st.look*math.Sin(t*0.35) + st.angle + p.shake*8*math.Sin(t*25)
	pitch := (st.nod*math.Sin(t*2.2) - 9*p.mood[mThinking]) * math.Pi / 180
	cy, sy, cp, sp := math.Cos(yaw), math.Sin(yaw), math.Cos(pitch), math.Sin(pitch)
	// World to head: undo the yaw, then the pitch.
	toHead := func(v v3) v3 {
		v = v3{cy*v.x - sy*v.z, v.y, sy*v.x + cy*v.z}
		return v3{v.x, cp*v.y + sp*v.z, -sp*v.y + cp*v.z}
	}
	o, dir := toHead(v3{x, y, 3}), toHead(v3{0, 0, -1})
	f := func(q v3) float64 { return maskSDF(q, st.frown, st.smile) }
	dist, hit := 1.8, false
	var q v3
	for i := 0; i < 56 && dist < 4.4; i++ {
		q = o.add(dir.mul(dist))
		d := f(q)
		if d < 0.002 {
			hit = true
			break
		}
		dist += d
	}
	if !hit {
		return p.edge, 0, 2
	}
	const eps = 0.003
	k1, k2, k3, k4 := v3{1, -1, -1}, v3{-1, -1, 1}, v3{-1, 1, -1}, v3{1, 1, 1}
	n := k1.mul(f(q.add(k1.mul(eps)))).add(k2.mul(f(q.add(k2.mul(eps))))).
		add(k3.mul(f(q.add(k3.mul(eps))))).add(k4.mul(f(q.add(k4.mul(eps))))).norm()
	// The key light comes from the side, so the relief casts shade: lit
	// head on, every surface turns the same grey and the features vanish.
	light, view := toHead(v3{-0.75, -0.45, 0.5}.norm()), toHead(v3{0, 0, 1})
	ao := math.Max(0, math.Min(1, f(q.add(n.mul(0.06)))/0.06))
	rim := math.Pow(1-math.Abs(n.dot(view)), 3) * 0.3
	// The matter flows over the stone, as on the statues.
	tex := fbm(q.x*4+ph*0.3, q.y*4-ph*0.4)
	v := (0.05 + 0.95*math.Max(0, n.dot(light))*ao + rim) * (0.85 + 0.35*(tex-0.5))
	v = v * v * (3 - 2*v) // deeper shadows, brighter planes

	open := e.open
	if b := math.Mod(t, 4.3); b < 0.18 {
		open *= math.Abs(b-0.09) / 0.09
	}
	glow := 0.0
	for _, sd := range []float64{-1, 1} {
		ec := v3{sd*0.22 + e.gx*0.4, e.gy * 0.4, 0.58}
		glow = math.Max(glow, smoothstep(0.1, 0.035, q.sub(ec).length()))
	}
	glow *= math.Min(1, open) * e.shown
	if glow > v {
		return lerpC(p.mid, rgb{255, 255, 255}, 0.5*glow), glow, 0.5
	}
	c := lerpC(p.edge, p.mid, math.Min(1, v/0.5))
	if v > 0.5 {
		c = lerpC(p.mid, p.center, (v-0.5)/0.5)
	}
	return c, math.Min(1, v), 0.5
}
