package main

import (
	"fmt"
	"math"
	"strings"
	"sync"
)

// dither is interleaved gradient noise: a threshold that looks random but
// is evenly spread, so braille shading shows no grid pattern (Bayer's flaw).
func dither(x, y int) float64 {
	v := 52.9829189 * math.Mod(0.06711056*float64(x)+0.00583715*float64(y), 1)
	return v - math.Floor(v)
}

func fg(c rgb, a float64) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", int(c.r*a), int(c.g*a), int(c.b*a))
}

// ramp runs from empty to dense. Round glyphs (o, O) suit an orb; few
// steps means little flicker as the noise moves.
var ramp = []rune(" .·:;=+*xoO#%@")

// frame draws one picture of w×h cells, the last row being the label, and
// returns it as one write: synchronized output keeps it from tearing.
func frame(w, h int, t, ph float64, p params, style, label string) string {
	rows := h - 1
	// One cell is ~1 wide by 2 tall: scale so the orb fits the smaller side.
	// The orb takes about two thirds, leaving room for the halo and dust.
	scale := math.Min(float64(w), float64(rows)*2) / 3
	at := func(cx, cy float64) (rgb, float64, float64) { // cx, cy in cell units
		return sample((cx-float64(w)/2)/scale, (cy-float64(rows)/2)*2/scale, t, ph, p)
	}
	sp, gw := sparks(w, rows, scale, ph, p), 2*w
	if face != nil { // the dust leaves an orb's rim, which a statue lacks
		sp = make([]float64, len(sp))
	}
	dust := lerpC(p.mid, p.center, 0.4)

	// Rows are independent: one goroutine each keeps braille's 8 samples
	// per cell inside the frame budget.
	lines := make([]string, rows)
	var wg sync.WaitGroup
	for j := 0; j < rows; j++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			var sb strings.Builder
			for i := 0; i < w; i++ {
				fx, fy := float64(i)+0.5, float64(j)+0.5
				if style == "ascii" {
					c, a, d := at(fx, fy)
					spark := 0.0
					for k := 0; k < 8; k++ {
						spark = math.Max(spark, sp[(4*j+k/2)*gw+2*i+k%2])
					}
					if d > 1 && spark > 0.15 {
						g := "·"
						if spark > 0.6 {
							g = "*"
						}
						sb.WriteString(fg(dust, 0.3+0.7*spark) + g)
						continue
					}
					if a < 0.015 {
						sb.WriteByte(' ')
						continue
					}
					sb.WriteString(fg(c, 0.35+0.65*a) + string(ramp[int(a*float64(len(ramp)-1))]))
					continue
				}
				bits := [8]int{0x01, 0x08, 0x02, 0x10, 0x04, 0x20, 0x40, 0x80}
				var mask int
				var sum rgb
				var lum float64
				for k := 0; k < 8; k++ {
					dx, dy := k%2, k/2
					c, a, d := at(fx-0.25+0.5*float64(dx), fy-0.375+0.25*float64(dy))
					sum = rgb{sum.r + c.r/8, sum.g + c.g/8, sum.b + c.b/8}
					gx, gy := 2*i+dx, 4*j+dy
					// A thin rim keeps the silhouette crisp; dust is lit on top.
					rim := math.Abs(d-1) < 0.45/scale
					if s := sp[gy*gw+gx]; s > a {
						a = s
					}
					lum += a / 8
					if rim || math.Pow(a, 0.7) > dither(gx, gy) {
						mask |= bits[k]
					}
				}
				if mask == 0 {
					sb.WriteByte(' ')
					continue
				}
				// Dots are thin: density carries the brightness, so the colour
				// keeps its hue at near full strength or the orb reads as mud.
				peak := math.Max(sum.r, math.Max(sum.g, sum.b)) + 1
				sb.WriteString(fg(sum, 255/peak*(0.55+0.45*math.Sqrt(lum))) + string(rune(0x2800+mask)))
			}
			lines[j] = sb.String()
		}(j)
	}
	wg.Wait()
	return "\x1b[?2026h\x1b[H" + strings.Join(lines, "\x1b[0m\r\n") +
		"\x1b[0m\r\n\x1b[2K" + fg(p.mid, 1) + " ●\x1b[0m\x1b[2m" + label + "\x1b[0m\x1b[?2026l"
}
