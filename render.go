package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
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

// ramp textures flat cells, from empty to dense. Round glyphs (o, O) suit
// an orb; few steps means little flicker as the noise moves.
var ramp = []rune(" .·:;=+*xoO#%@")

// styleNames is the order s steps through.
var styleNames = []string{"braille", "ascii", "pixel", "octant", "sextant", "wedge", "shade", "bars",
	"halftone", "contour", "engrave", "matrix", "geometric", "image"}

// glyphShapes says where each glyph puts its ink in a cell cut into 2
// columns by 4 rows (left then right, top row first), in tenths. ASCII
// samples the same 2x4 grid as braille and takes the glyph whose shape is
// closest, so `"` lands where only the top is lit, `(` where the left is,
// `/` on a diagonal: the inner contours of a face survive.
// ponytail: shapes estimated by eye for a typical monospace font, not
// measured; rasterising the real font would make the match exact.
var glyphShapes = map[rune]string{
	' ': "00000000", '.': "00000033", ',': "00000042", '_': "00000066",
	'-': "00005500", '=': "00444400", '\'': "22000000", '`': "31000000",
	'"': "44000000", '^': "44220000", ':': "00330033", ';': "00330042",
	'!': "33330022", '|': "33333333", '(': "14404014", ')': "41040441",
	'[': "54505054", ']': "45050545", '/': "05233250", '\\': "50322305",
	'+': "11445511", '*': "22553300", 'x': "00334433", 'o': "00445544",
	'O': "55444455", '#': "66666666", '@': "77777777", 'b': "50505555",
	'd': "05055555", 'p': "00555550", 'q': "00555505", 'T': "66333333",
	'L': "50505066", 'J': "05050566", 'r': "00554040",
}

type glyphShape struct {
	r     rune
	cover [8]float64
}

var glyphs = func() []glyphShape {
	var gs []glyphShape
	for r, s := range glyphShapes {
		g := glyphShape{r: r}
		for k := range g.cover {
			g.cover[k] = float64(s[k]-'0') / 7 // 7 tenths of ink reads as full light
		}
		gs = append(gs, g)
	}
	return gs
}()

// bestGlyph returns the glyph for a cell's 8 lit values: the density ramp
// for a flat cell, else the glyph whose ink pattern is closest, after
// pushing the values apart from their mean so edges pick a shaped glyph.
func bestGlyph(s [8]float64) rune {
	mean, lo, hi := 0.0, 1.0, 0.0
	for _, v := range s {
		mean, lo, hi = mean+v/8, math.Min(lo, v), math.Max(hi, v)
	}
	// A flat cell has no shape to follow: shape matching would fill every
	// flat grey area with `|`.
	if hi-lo < 0.22 {
		return ramp[min(int(mean*float64(len(ramp))), len(ramp)-1)]
	}
	best, bestD := ' ', math.Inf(1)
	for _, g := range glyphs {
		d := 0.0
		for k, v := range s {
			e := math.Max(0, math.Min(1, mean+(v-mean)*1.7)) - g.cover[k]
			d += e * e
		}
		if d < bestD {
			best, bestD = g.r, d
		}
	}
	return best
}

// quadrants are the block glyphs by which quarters they fill: bit 0 top
// left, 1 top right, 2 bottom left, 3 bottom right.
var quadrants = []rune(" ▘▝▀▖▌▞▛▗▚▐▜▄▙▟█")

// octants are the same for a 2×4 grid (bit 2*row+column). Unicode 16 gives
// 230 octant glyphs in mask order and leaves out the 26 patterns older
// blocks already draw (halves, quadrants, quarter bars).
var octants = func() []rune {
	older := map[int]rune{
		0: ' ', 1: 0x1CEA8, 2: 0x1CEAB, 3: 0x1FB82, 5: '▘', 10: '▝', 15: '▀',
		20: 0x1FBE6, 40: 0x1FBE7, 63: 0x1FB85, 64: 0x1CEA3, 80: '▖', 85: '▌',
		90: '▞', 95: '▛', 128: 0x1CEA0, 160: '▗', 165: '▚', 170: '▐', 175: '▜',
		192: '▂', 240: '▄', 245: '▙', 250: '▟', 252: '▆', 255: '█',
	}
	t, next := make([]rune, 256), rune(0x1CD00)
	for m := range t {
		if r, ok := older[m]; ok {
			t[m] = r
		} else {
			t[m], next = next, next+1
		}
	}
	return t
}()

// blockCell draws a cell's pixels (row by row, left then right) with the
// block glyph of table. A cell shows only two colours, the glyph's and its
// background's, so pixels split at the midpoint of their brightness, each
// side taking its mean colour. Unlit pixels keep the terminal's own
// background, so the picture has no dark box around it.
func blockCell(px []rgb, table []rune) string {
	lum := func(c rgb) float64 { return c.r + c.g + c.b }
	lo, hi := math.Inf(1), 0.0
	for _, p := range px {
		lo, hi = math.Min(lo, lum(p)), math.Max(hi, lum(p))
	}
	if hi <= 12 {
		return " "
	}
	var f, b rgb
	var mask int
	var nf, nb float64
	for k, p := range px {
		if lum(p) > (lo+hi)/2 || lo == hi {
			mask, nf, f = mask|1<<k, nf+1, rgb{f.r + p.r, f.g + p.g, f.b + p.b}
		} else {
			nb, b = nb+1, rgb{b.r + p.r, b.g + p.g, b.b + p.b}
		}
	}
	f = rgb{f.r / nf, f.g / nf, f.b / nf}
	if nb > 0 {
		b = rgb{b.r / nb, b.g / nb, b.b / nb}
	}
	return twoTone(f, b, table[mask])
}

// twoTone prints r in colour f on background b; a dark b keeps the
// terminal's own background.
func twoTone(f, b rgb, r rune) string {
	if b.r+b.g+b.b <= 12 {
		return fg(f, 1) + string(r)
	}
	return fg(f, 1) + fmt.Sprintf("\x1b[48;2;%d;%d;%dm%c\x1b[49m", int(b.r), int(b.g), int(b.b), r)
}

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
	if face != nil { // a statue throws dust only when it bursts
		b := 0.6 * burst(t, p.statue)
		for i := range sp {
			sp[i] *= b
		}
	}
	dust := lerpC(p.mid, p.center, 0.4)
	foot := "\x1b[2K" + fg(p.mid, 1) + " ●\x1b[0m\x1b[2m" + label + "\x1b[0m\x1b[?2026l"

	if style == "image" {
		// A real picture through the kitty graphics protocol (Ghostty, kitty,
		// WezTerm), kept stylised on purpose: at full resolution the orb read
		// as a photo. 2×4 big pixels per cell, each a 2×2 square with a
		// 1-pixel gap around it, and brightness in 6 steps like pixel art.
		// Alpha carries the brightness so the terminal's background shows.
		const sx, sy, bs = 2, 4, 3
		pw, pht := w*sx*bs, rows*sy*bs
		pix := make([]byte, pw*pht*4)
		var wg sync.WaitGroup
		for y := 0; y < rows*sy; y++ {
			wg.Add(1)
			go func(y int) {
				defer wg.Done()
				for x := 0; x < w*sx; x++ {
					c, a, _ := at((float64(x)+0.5)/sx, (float64(y)+0.5)/sy)
					if sk := sp[(y*4/sy)*gw+x*2/sx]; sk > a {
						a, c = sk, dust
					}
					al := byte(255 * math.Round(5*math.Pow(math.Min(a, 1), 0.8)) / 5)
					if al == 0 {
						continue
					}
					for dy := 0; dy < bs-1; dy++ {
						for dx := 0; dx < bs-1; dx++ {
							o := ((y*bs+dy)*pw + x*bs + dx) * 4
							pix[o], pix[o+1], pix[o+2] = byte(math.Min(c.r, 255)), byte(math.Min(c.g, 255)), byte(math.Min(c.b, 255))
							pix[o+3] = al
						}
					}
				}
			}(y)
		}
		wg.Wait()
		return "\x1b[?2026h\x1b[H" + kittyImage(pix, pw, pht, w, rows) + fmt.Sprintf("\x1b[%d;1H", h) + foot
	}

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
				if draw, ok := cellStyles[style]; ok {
					sb.WriteString(draw(cell{i: i, j: j, t: t, at: func(u, v float64) (rgb, float64) {
						c, a, _ := at(float64(i)+u, float64(j)+v)
						if sk := sp[(4*j+min(int(v*4), 3))*gw+2*i+min(int(u*2), 1)]; sk > a {
							return dust, sk
						}
						return c, a
					}}))
					continue
				}
				if style == "pixel" || style == "octant" {
					// Pixels per cell: 2×2 drawn with quadrants, or 2×4 with
					// octants, braille's grid in solid blocks.
					table, ny := quadrants, 2
					if style == "octant" {
						table, ny = octants, 4
					}
					px := make([]rgb, 2*ny)
					for k := range px {
						dx, dy := k%2, k/2
						c, a, _ := at(fx-0.25+0.5*float64(dx), fy-0.5+(float64(dy)+0.5)/float64(ny))
						gx, gy := 2*i+dx, 4*j+dy*4/ny
						if sk := sp[gy*gw+gx]; sk > a {
							a, c = sk, dust
						}
						a = math.Pow(a, 0.8)
						px[k] = rgb{c.r * a, c.g * a, c.b * a}
					}
					sb.WriteString(blockCell(px, table))
					continue
				}
				if style == "ascii" {
					var s [8]float64
					var sum rgb
					peak := 0.0
					for k := 0; k < 8; k++ {
						dx, dy := k%2, k/2
						c, a, _ := at(fx-0.25+0.5*float64(dx), fy-0.375+0.25*float64(dy))
						if sk := sp[(4*j+dy)*gw+2*i+dx]; sk > a {
							a, c = sk, dust
						}
						s[k], peak = a, math.Max(peak, a)
						sum = rgb{sum.r + c.r/8, sum.g + c.g/8, sum.b + c.b/8}
					}
					g := bestGlyph(s)
					if g == ' ' {
						sb.WriteByte(' ')
						continue
					}
					// Brightness from the cell's brightest point: a thin stroke
					// on an edge still carries the light it stands for.
					sb.WriteString(fg(sum, 0.35+0.65*peak) + string(g))
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
					if s := sp[gy*gw+gx]; s > a {
						a = s
					}
					lum += a / 8
					// A thin rim, every other dot, keeps the silhouette without
					// drawing a hard line; dust is lit on top.
					rim := math.Abs(d-1) < 0.45/scale && dither(gx, gy) < 0.5
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
		"\x1b[0m\r\n" + foot
}

// kittyImage sends RGBA pixels as image 1, stretched over cols×rows cells,
// replacing the previous frame in place. zlib keeps a mostly empty frame
// small; the payload goes in the 4096-byte chunks the protocol requires.
func kittyImage(pix []byte, pw, ph, cols, rows int) string {
	var z bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&z, zlib.BestSpeed)
	zw.Write(pix)
	zw.Close()
	data := base64.StdEncoding.EncodeToString(z.Bytes())
	var sb strings.Builder
	for i := 0; i < len(data); i += 4096 {
		chunk, more := data[i:min(i+4096, len(data))], 0
		if i+4096 < len(data) {
			more = 1
		}
		if i == 0 {
			fmt.Fprintf(&sb, "\x1b_Ga=T,f=32,o=z,s=%d,v=%d,c=%d,r=%d,i=1,C=1,q=2,m=%d;%s\x1b\\", pw, ph, cols, rows, more, chunk)
		} else {
			fmt.Fprintf(&sb, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}
	}
	return sb.String()
}

// kittyClear removes the picture when leaving the image style.
const kittyClear = "\x1b_Ga=d,d=A,q=2\x1b\\"
