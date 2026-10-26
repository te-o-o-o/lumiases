package main

import "math"

// forms are the shapes the agent can take, drawn from code alone, no
// picture: the orb, a swarm of dots and pulsar ridge lines. The scenes read the state through p.mood, each state's weight
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

var forms = []string{"orb", "swarm", "pulsar"}

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
