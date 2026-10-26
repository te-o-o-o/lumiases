package main

import "math"

type rgb struct{ r, g, b float64 }

// params drive the orb; each Claude state is one set, and the orb eases
// from one to the next so a state change morphs instead of jumping.
type params struct {
	amp, speed, warp, shake, pulse, size float64
	center, mid, edge                    rgb
	eyes                                 eyeParams
	statue                               statueParams
	agent                                agentParams
	mood                                 [8]float64 // each state's weight, see scenes.go
}

// states are the moods Claude's actions map to; their order is the 1-8 keys.
// Each state's eyes live in stateEyes and its statue effect in stateStatue.
var states = []struct {
	name string
	p    params
}{
	{"idle", params{amp: 0.08, speed: 0.5, warp: 0.10, size: 0.80, center: rgb{180, 200, 255}, mid: rgb{90, 110, 230}, edge: rgb{40, 30, 120}}},
	{"thinking", params{amp: 0.12, speed: 0.8, warp: 0.30, size: 0.85, center: rgb{255, 220, 255}, mid: rgb{140, 100, 255}, edge: rgb{60, 20, 140}}},
	{"reading", params{amp: 0.10, speed: 1.2, warp: 0.15, size: 0.85, center: rgb{220, 255, 255}, mid: rgb{70, 210, 230}, edge: rgb{10, 70, 120}}},
	{"editing", params{amp: 0.28, speed: 1.8, warp: 0.34, size: 0.90, center: rgb{255, 240, 200}, mid: rgb{255, 140, 60}, edge: rgb{140, 30, 20}}},
	{"running", params{amp: 0.24, speed: 2.8, warp: 0.25, size: 0.90, center: rgb{230, 255, 220}, mid: rgb{60, 220, 130}, edge: rgb{10, 80, 60}}},
	{"waiting", params{amp: 0.06, speed: 0.6, warp: 0.08, pulse: 0.12, size: 0.80, center: rgb{255, 250, 220}, mid: rgb{255, 176, 0}, edge: rgb{120, 60, 0}}},
	{"error", params{amp: 0.16, speed: 2.5, warp: 0.20, shake: 0.025, size: 0.85, center: rgb{255, 230, 220}, mid: rgb{255, 77, 77}, edge: rgb{90, 0, 20}}},
	{"done", params{amp: 0.10, speed: 1.0, warp: 0.12, size: 1.00, center: rgb{240, 255, 230}, mid: rgb{124, 255, 107}, edge: rgb{20, 90, 70}}},
}

func lerp(a, b, k float64) float64 { return a + (b-a)*k }

func lerpC(a, b rgb, k float64) rgb {
	return rgb{lerp(a.r, b.r, k), lerp(a.g, b.g, k), lerp(a.b, b.b, k)}
}

func ease(a, b params, k float64) params {
	out := params{
		amp: lerp(a.amp, b.amp, k), speed: lerp(a.speed, b.speed, k), warp: lerp(a.warp, b.warp, k),
		shake: lerp(a.shake, b.shake, k), pulse: lerp(a.pulse, b.pulse, k), size: lerp(a.size, b.size, k),
		center: lerpC(a.center, b.center, k), mid: lerpC(a.mid, b.mid, k), edge: lerpC(a.edge, b.edge, k),
		eyes: a.eyes.ease(b.eyes, k), statue: a.statue.ease(b.statue, k), agent: a.agent.ease(b.agent, k),
	}
	for i := range out.mood {
		out.mood[i] = lerp(a.mood[i], b.mood[i], k)
	}
	return out
}

func smoothstep(e0, e1, x float64) float64 {
	k := math.Max(0, math.Min(1, (x-e0)/(e1-e0)))
	return k * k * (3 - 2*k)
}

func hash(x, y int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ h>>13) * 1274126177
	return float64(h^h>>16) / math.MaxUint32
}

// noise is smooth value noise in [0, 1].
func noise(x, y float64) float64 {
	xi, yi := math.Floor(x), math.Floor(y)
	fx, fy := x-xi, y-yi
	ux, uy := fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	ix, iy := int(xi), int(yi)
	return lerp(lerp(hash(ix, iy), hash(ix+1, iy), ux), lerp(hash(ix, iy+1), hash(ix+1, iy+1), ux), uy)
}

// fbm stacks 4 octaves of noise: large swirls carrying ever finer detail.
func fbm(x, y float64) float64 {
	s, a := 0.0, 0.5
	for i := 0; i < 4; i++ {
		s += a * noise(x, y)
		x, y = x*2.03+17.1, y*2.03-9.7
		a /= 2
	}
	return s / 0.9375
}

// sample returns the colour, brightness (0..1) and normalised distance to
// the edge (0 centre, 1 edge) at x, y, in square units with the orb about 1
// wide. ph is the accumulated phase (time scaled by speed), so speed changes
// never make the shape jump.
func sample(x, y, t, ph float64, p params) (rgb, float64, float64) {
	if face != nil {
		return sampleFace(x, y, t, ph, p)
	}
	switch form {
	case "tendrils":
		return tendrils(x, y, t, ph, p)
	case "eye":
		return bigEye(x, y, t, ph, p)
	case "hood":
		return hood(x, y, t, ph, p)
	case "swarm":
		return swarm(x, y, t, ph, p)
	case "blackhole":
		return blackHole(x, y, t, ph, p)
	case "pulsar":
		return pulsar(x, y, t, ph, p)
	case "mask":
		return mask(x, y, t, ph, p)
	}
	return orb(x, y, t, ph, p)
}

// orb is the living blob, the default form and the core of the others.
func orb(x, y, t, ph float64, p params) (rgb, float64, float64) {
	// Error trembles instead of growing thorns.
	x += p.shake * math.Sin(t*53)
	y += p.shake * 0.6 * math.Sin(t*41)
	// Domain warp by noise: the whole surface churns, not only the outline.
	qx, qy := fbm(x*1.6+ph*0.3, y*1.6), fbm(x*1.6-5.2, y*1.6+ph*0.25)
	wx, wy := x+p.warp*(qx-0.5)*2, y+p.warp*(qy-0.5)*2
	r, th := math.Hypot(wx, wy), math.Atan2(wy, wx)

	// Edge: low harmonics give the big lobes, noise read on a circle (no seam
	// at ±π) adds the fine ripples.
	big := 0.5*math.Sin(3*th+ph) + 0.3*math.Sin(5*th-1.7*ph+1) + 0.2*math.Sin(7*th+2.3*ph+2)
	fine := fbm(math.Cos(th)*2+ph*0.4, math.Sin(th)*2-ph*0.3) - 0.5
	R := p.size * (1 + p.amp*big + p.amp*1.2*fine)
	R *= 1 + p.pulse*math.Sin(4*t)
	d := r / R

	var c rgb
	if d < 0.6 {
		c = lerpC(p.center, p.mid, d/0.6)
	} else {
		c = lerpC(p.mid, p.edge, math.Min((d-0.6)/0.4, 1))
	}
	if d >= 1 {
		return c, 0.18 * math.Exp(-(d-1)*12), d // halo fading out just past the edge
	}

	// Soft relief: the orb as a dome lit almost head-on, so the centre stays
	// the brightest point and the bulges catch the light.
	u, v, h := wx/R, wy/R, math.Sqrt(1-d*d)
	diffuse := math.Max(0, -0.3*u-0.4*v+0.866*h)
	spec := math.Pow(math.Max(0, -0.15*u-0.2*v+0.968*h), 30)
	// Filaments flowing through the inside.
	tex := fbm(wx*5+qx*2, wy*5-ph*0.5)

	i := (0.45 + 0.55*diffuse) * (0.7 + 0.6*(tex-0.5)) * (1 - 0.35*d)
	i = math.Min(1, i+0.5*spec)
	i = i * i * (3 - 2*i) // S-curve: deeper shadows, punchier highlights
	// Eyes use the warped coordinates, so they wobble with the matter.
	k, g := eyes(wx/p.size, wy/p.size, t, p.eyes)
	i = math.Max(i*(1-0.95*k), g)
	// The rim softens into the halo rather than stopping dead.
	i = lerp(0.18, i, smoothstep(1, 0.9, d))
	return lerpC(c, rgb{255, 255, 255}, math.Max(0.7*spec, 0.6*g)), i, d
}

// sparks returns a brightness grid at braille dot resolution (2 per cell
// across, 4 down) of dust leaving the orb's edge and drifting off.
// Stateless: each particle's age is a function of the phase, so there is
// nothing to update and it follows state speed for free.
func sparks(w, rows int, scale, ph float64, p params) []float64 {
	gw, gh := 2*w, 4*rows
	g := make([]float64, gw*gh)
	for k := 0; k < 90; k++ {
		life := 1.5 + 2*hash(k, 2)
		age := math.Mod(ph*0.6+hash(k, 3)*life, life) / life
		th := hash(k, 1)*2*math.Pi + 0.4*age
		r := p.size * (0.95 + 0.8*age)
		b := math.Min(1, age*8) * (1 - age) * (1 - age)
		x, y := int((r*math.Cos(th)*scale+float64(w)/2)*2), int((r*math.Sin(th)*scale/2+float64(rows)/2)*4)
		if x >= 0 && x < gw && y >= 0 && y < gh {
			g[y*gw+x] = math.Max(g[y*gw+x], b)
		}
	}
	return g
}
