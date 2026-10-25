package main

import "math"

// eyeParams set the eyes' expression: open is the height (1 normal, <1
// squint, >1 wide), gx/gy the gaze, scan a left-right sweep, tilt a frown,
// smile turns them into ^ ^. shown fades them in and out; the f key sets
// it, not the state.
type eyeParams struct {
	open, gx, gy, scan, tilt, smile, shown float64
}

var stateEyes = map[string]eyeParams{
	"idle":     {open: 1},
	"thinking": {open: 0.45, gx: 0.08, gy: -0.07},
	"reading":  {open: 0.8, gy: 0.02, scan: 0.08},
	"editing":  {open: 0.7, gy: 0.06},
	"running":  {open: 0.9},
	"waiting":  {open: 1.3},
	"error":    {open: 0.5, tilt: 0.45},
	"done":     {open: 1, smile: 1},
}

func (a eyeParams) ease(b eyeParams, k float64) eyeParams {
	return eyeParams{
		open: lerp(a.open, b.open, k), gx: lerp(a.gx, b.gx, k), gy: lerp(a.gy, b.gy, k),
		scan: lerp(a.scan, b.scan, k), tilt: lerp(a.tilt, b.tilt, k), smile: lerp(a.smile, b.smile, k),
		shown: a.shown,
	}
}

var eyeStyles = []string{"round", "slits", "void", "glasses", "shades"}

// eyeStyle is set from main between frames, read by the row goroutines.
var eyeStyle = "round"

// roundEye is an ellipse rx wide, ry*open tall, that turns into a ^ arc
// as smile goes to 1.
func roundEye(ex, ey, rx, ry, open, smile float64) float64 {
	eye := smoothstep(1.15, 0.85, math.Hypot(ex/rx, ey/(ry*open+0.02)))
	arc := smoothstep(0.065, 0.035, math.Abs(math.Hypot(ex, ey-0.08)-rx*0.93))
	if ey > 0.08 {
		arc = 0
	}
	return lerp(eye, arc, smile)
}

// eyes returns, at x, y (orb units: radius about 1), how much is carved out
// of the matter (the dark eyes) and how much glows (pupils, frames, glints).
// Both go through the same rendering as the orb, so they stay in its style.
func eyes(x, y, t float64, e eyeParams) (carve, glow float64) {
	open := e.open
	if b := math.Mod(t, 4.3); b < 0.18 { // a blink every 4.3 s
		open *= math.Abs(b-0.09) / 0.09
	}
	tilt := e.tilt
	if eyeStyle == "slits" {
		tilt += 0.35 // always scowling
	}
	for _, s := range []float64{-1, 1} {
		// f: fixed to the face (frames, lenses); e: the eye itself, which
		// follows the gaze. Tilt turns each so its inner corner drops.
		c, sn := math.Cos(s*tilt), math.Sin(s*tilt)
		fx, fy := x-s*0.34, y+0.1
		fx, fy = c*fx+sn*fy, -sn*fx+c*fy
		ex, ey := x-(s*0.34+e.gx+e.scan*math.Sin(t*2.5)), y-(-0.1+e.gy)
		ex, ey = c*ex+sn*ey, -sn*ex+c*ey
		var k, g float64
		switch eyeStyle {
		case "slits":
			// Almond, sharp at both ends, with a glowing vertical pupil.
			h := (0.11*open + 0.025) * (1 - (ex/0.2)*(ex/0.2))
			k = smoothstep(0.015, -0.015, math.Abs(ey)-h)
			g = k * smoothstep(0.035, 0.015, math.Abs(ex))
		case "void":
			// Big hollow sockets, a pinpoint of light deep inside.
			k = smoothstep(1.1, 0.9, math.Hypot(ex/0.17, ey/(0.24*open+0.02)))
			g = k * smoothstep(0.04, 0.018, math.Hypot(ex, ey))
		case "glasses":
			r := math.Hypot(fx, fy)
			lens := smoothstep(0.19, 0.17, r)
			g = smoothstep(0.035, 0.018, math.Abs(r-0.2)) // the frame
			g = math.Max(g, 0.7*lens*smoothstep(0.02, 0.008, math.Abs((fx+fy)*0.707+0.07)))
			k = math.Max(0.3*lens, roundEye(ex, ey, 0.09, 0.12, open, e.smile)*lens)
		case "shades":
			lens := math.Pow(math.Abs(fx/0.22), 4) + math.Pow(math.Abs((fy-0.02)/0.15), 4)
			k = smoothstep(1.1, 0.9, lens)
			g = smoothstep(0.03, 0.012, math.Abs(fy+0.13)) * smoothstep(0.25, 0.23, math.Abs(fx)) // top bar
			// A glint sweeps across the lenses every few seconds.
			sweep := math.Mod(t*0.4, 2.5) - 0.6
			g = math.Max(g, 0.8*k*smoothstep(0.025, 0.01, math.Abs(fx*0.8+fy*0.6-sweep)))
		default:
			k = roundEye(ex, ey, 0.15, 0.21, open, e.smile)
		}
		carve, glow = math.Max(carve, k), math.Max(glow, g)
	}
	if eyeStyle == "glasses" || eyeStyle == "shades" {
		bridge := smoothstep(0.03, 0.012, math.Abs(y+0.12)) * smoothstep(0.16, 0.13, math.Abs(x))
		glow = math.Max(glow, bridge)
	}
	return carve * e.shown, glow * e.shown
}
