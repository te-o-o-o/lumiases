package main

import "math"

// Forms are the shapes the agent can take besides the orb. Each is drawn by
// a sample function with the orb's contract: colour, brightness, and a
// distance that is 1 on the silhouette's edge (braille draws a rim there)
// and below 1 where no rim belongs.
var forms = []string{"orb", "tendrils", "eye", "hood"}

// form is set from main between frames, read by the row goroutines.
var form = "orb"

// agentParams drive the tendrils: how far they reach out, how much they
// curl, how taut they pull, and how strongly their tips bend toward a work
// point at height ay. glow lights the core from within.
type agentParams struct {
	reach, curl, taut, aim, ay, glow float64
}

var stateAgent = map[string]agentParams{
	"idle":     {reach: 0.35, curl: 0.4},
	"thinking": {reach: 0.12, curl: 1, glow: 1},             // folded in, lit inside
	"reading":  {reach: 1, curl: 0.15},                      // out to the edge and back
	"editing":  {reach: 0.6, curl: 0.25, aim: 0.9, ay: 0.9}, // shaping something below
	"running":  {reach: 0.8, taut: 1},                       // pulled straight, humming
	"waiting":  {reach: 0.6, aim: 0.7, ay: -1.3},            // raised toward you
	"error":    {reach: 0.05, curl: 1},                      // snatched back
	"done":     {reach: 1, glow: 1},                         // spread wide
}

func (a agentParams) ease(b agentParams, k float64) agentParams {
	return agentParams{
		reach: lerp(a.reach, b.reach, k), curl: lerp(a.curl, b.curl, k), taut: lerp(a.taut, b.taut, k),
		aim: lerp(a.aim, b.aim, k), ay: lerp(a.ay, b.ay, k), glow: lerp(a.glow, b.glow, k),
	}
}

// tendrils is a small orb, eyes and all, with limbs that do the work.
func tendrils(x, y, t, ph float64, p params) (rgb, float64, float64) {
	a := p.agent
	core := p
	core.size *= 0.55
	core.amp *= 0.6
	cc, ci, cd := orb(x, y, t, ph, core)
	ci = math.Min(1, ci*(1+0.6*a.glow))
	x += p.shake * math.Sin(t*53)
	y += p.shake * 0.6 * math.Sin(t*41)

	// Each limb is a polyline from the core's edge; keep the closest one,
	// as a distance in its own widths, and where along it (0 root, 1 tip).
	const n, seg = 7, 10
	best, at := math.Inf(1), 0.0
	for k := 0; k < n; k++ {
		fk := float64(k)
		base := 2*math.Pi*fk/n + 0.15*ph + 0.4*hash(k, 7)
		// Limbs reach out and back in turn, never all together.
		reach := a.reach * (0.65 + 0.35*math.Sin(ph*1.3+fk*2.1))
		l := 0.15 + 1.1*reach*(0.8+0.4*hash(k, 8))
		var px, py float64
		for j := 0; j <= seg; j++ {
			s := float64(j) / seg
			th := base + a.curl*2.2*s*s*math.Sin(ph*0.9+fk) +
				(1-a.taut)*0.35*s*math.Sin(5*s-ph*2+fk) + a.taut*0.04*math.Sin(40*s+t*35+fk)
			r := core.size*0.8 + l*s
			qx, qy := r*math.Cos(th), r*math.Sin(th)
			bend := a.aim * s * s
			qx, qy = lerp(qx, 0.25*math.Cos(base), bend), lerp(qy, a.ay, bend)
			if j > 0 {
				dx, dy := qx-px, qy-py
				u := math.Max(0, math.Min(1, ((x-px)*dx+(y-py)*dy)/(dx*dx+dy*dy+1e-9)))
				sm := (float64(j-1) + u) / seg
				w := 0.09*(1-0.75*sm) + 0.015
				if dn := math.Hypot(x-px-u*dx, y-py-u*dy) / w; dn < best {
					best, at = dn, sm
				}
			}
			px, py = qx, qy
		}
	}
	ti := smoothstep(1, 0.6, best)
	if ti == 0 {
		return cc, ci, cd
	}
	// Pulses run out along the limbs, carrying the core's light.
	flow := 0.75 + 0.25*math.Sin(18*at-ph*5)
	tex := fbm(x*6, y*6-ph*0.5)
	ti *= (1.1 - 0.4*at) * flow * (0.8 + 0.4*(tex-0.5)) * (1 + 0.3*a.glow)
	if ti <= ci {
		return cc, ci, cd
	}
	return lerpC(lerpC(p.center, p.mid, 0.5), p.edge, at), math.Min(1, ti), 0.5
}

// bigEye is one eye as big as the orb: the matter is its white, the lids
// open and close with the state, and the iris looks where the eyes would.
func bigEye(x, y, t, ph float64, p params) (rgb, float64, float64) {
	e := p.eyes
	white := p
	white.eyes.shown = 0
	white.amp *= 0.5
	c, a, d := orb(x, y, t, ph, white)
	if d >= 1 {
		return c, a, d
	}
	s := p.size
	open := e.open
	if b := math.Mod(t, 4.3); b < 0.18 {
		open *= math.Abs(b-0.09) / 0.09
	}
	// Lids: an almond whose height follows open; a frown lowers the top
	// lid, a smile raises the bottom one.
	lid := 0.0
	if ux := x / (1.05 * s); math.Abs(ux) < 1 {
		h := s * 0.62 * open * (1 - ux*ux)
		top, bot := h*(1-0.5*e.tilt), h*(1-0.8*e.smile)
		lid = smoothstep(-0.03, 0.03, y+top) * smoothstep(0.03, -0.03, y-bot)
	}
	ix, iy := (e.gx*3+e.scan*3*math.Sin(t*2.5))*s, e.gy*3*s
	ri := math.Hypot(x-ix, y-iy) / (0.36 * s)
	ang := math.Atan2(y-iy, x-ix)
	// Fibres run outward from the pupil; the pupil shrinks with focus and
	// widens with alarm.
	fib := fbm(math.Cos(ang)*4+ri*1.5, math.Sin(ang)*4-ph*0.3)
	pupil := 0.42 * math.Max(0.45, math.Min(1.3, open))
	// The iris glows against a dim white, ringed in black like a real one.
	iris := (0.6 + 0.4*fib) * (1.1 - 0.3*ri) * smoothstep(1, 0.88, ri) * smoothstep(pupil-0.04, pupil+0.04, ri)
	glint := smoothstep(0.06, 0.03, math.Hypot(x-ix+0.12*s, y-iy+0.12*s))
	v, col := a*0.45*smoothstep(1, 1.12, ri), c
	if ri < 1 {
		v, col = math.Max(iris, glint), lerpC(p.mid, p.center, fib)
	}
	return lerpC(lerpC(p.edge, p.mid, 0.4), col, lid), lerp(a*0.12, math.Min(1, v), lid), d
}

// hood is a cloaked figure: cloth catching light from above, a dark face
// opening, and only the eyes alight inside it.
func hood(x, y, t, ph float64, p params) (rgb, float64, float64) {
	x += p.shake * math.Sin(t*53)
	y += p.shake * 0.6 * math.Sin(t*41)
	s := p.size / 0.8
	x, y = x/s, y/s*(1+p.pulse*math.Sin(4*t)*0.3)
	// Hood and shoulders, smoothly joined, the cloth's edge stirring.
	dh := math.Hypot(x/0.52, (y+0.3)/0.68) - 1
	db := math.Hypot(x/1.05, (y-1.15)/0.75) - 1
	k := 0.25
	j := math.Max(0, math.Min(1, 0.5+0.5*(db-dh)/k))
	dd := lerp(db, dh, j) - k*j*(1-j) + p.amp*0.6*(fbm(x*2+ph*0.4, y*2-ph*0.3)-0.5)
	if dd > 0.06 {
		return p.edge, 0, 2
	}
	m := smoothstep(0.06, -0.04, dd)
	fold := fbm(x*3.5, y*0.9-ph*0.15)
	light := 0.25 + 0.6*smoothstep(1.6, -0.9, y)
	v := light * (0.65 + 0.8*(fold-0.5))
	fo := math.Hypot(x/0.3, (y+0.2)/0.4)
	v *= smoothstep(0.8, 1.1, fo)
	v = math.Max(v, 0.55*light*smoothstep(0.12, 0, math.Abs(fo-1.08))) // the opening's lit rim
	ek, eg := eyes(x/0.42, (y+0.2)/0.42, t, p.eyes)
	glow := smoothstep(1.05, 0.9, fo) * math.Max(0.85*ek, eg)
	if glow > v {
		return lerpC(p.mid, p.center, glow), glow, 0.5
	}
	return lerpC(p.edge, p.mid, math.Min(1, v*1.4)), v * m, 0.5
}
