package main

import "math"

// The styles below draw one cell at a time from samples taken inside it.
// braille, ascii, the quarter and octant blocks and the image style live in
// render.go; these came later, as looks to try side by side.

// cell is what a style sees of one terminal cell: at reads colour and
// brightness at u, v in 0..1 across it, dust included.
type cell struct {
	i, j int
	t    float64
	at   func(u, v float64) (rgb, float64)
}

var cellStyles = map[string]func(cell) string{
	"sextant":   sextantCell,
	"wedge":     wedgeCell,
	"shade":     shadeCell,
	"bars":      barsCell,
	"halftone":  halftoneCell,
	"contour":   contourCell,
	"engrave":   engraveCell,
	"matrix":    matrixCell,
	"geometric": geometricCell,
}

// grid samples the cell at the centres of an nx×ny grid, row by row, and
// returns the colours already dimmed by brightness, and the brightness.
func (c cell) grid(nx, ny int) ([]rgb, []float64) {
	px, as := make([]rgb, nx*ny), make([]float64, nx*ny)
	for k := range px {
		col, a := c.at((float64(k%nx)+0.5)/float64(nx), (float64(k/nx)+0.5)/float64(ny))
		a = math.Min(1, a)
		px[k], as[k] = rgb{col.r * a, col.g * a, col.b * a}, a
	}
	return px, as
}

// mean returns the cell's average colour (undimmed) and brightness.
func (c cell) mean() (rgb, float64) {
	var sum rgb
	total := 0.0
	for k := 0; k < 4; k++ {
		col, a := c.at(0.25+0.5*float64(k%2), 0.25+0.5*float64(k/2))
		sum, total = rgb{sum.r + col.r/4, sum.g + col.g/4, sum.b + col.b/4}, total+math.Min(1, a)/4
	}
	return sum, total
}

// vivid pushes a colour to full strength, keeping its hue: for glyphs whose
// size or density already carries the brightness.
func vivid(c rgb) rgb {
	peak := math.Max(c.r, math.Max(c.g, c.b)) + 1
	return rgb{c.r * 255 / peak, c.g * 255 / peak, c.b * 255 / peak}
}

// sextants are the block glyphs of a 2×3 grid (bit 2*row+column), in mask
// order; four patterns are older glyphs.
var sextants = func() []rune {
	older := map[int]rune{0: ' ', 21: '▌', 42: '▐', 63: '█'}
	t, next := make([]rune, 64), rune(0x1FB00)
	for m := range t {
		if r, ok := older[m]; ok {
			t[m] = r
		} else {
			t[m], next = next, next+1
		}
	}
	return t
}()

func sextantCell(c cell) string {
	px, _ := c.grid(2, 3)
	return blockCell(px, sextants)
}

// wedges are the diagonal block glyphs (U+1FB3C..1FB67) with the part of a
// 4×6 grid each one fills (bit 4*row+column), worked out from their Unicode
// names: "LOWER LEFT BLOCK DIAGONAL UPPER MIDDLE LEFT TO LOWER CENTRE" fills
// the lower left side of that line.
var wedges = map[rune]uint32{
	0x1FB3C: 0x110000, 0x1FB3D: 0x710000, 0x1FB3E: 0x311000, 0x1FB3F: 0x733100,
	0x1FB40: 0x311100, 0x1FB41: 0xFFFFFC, 0x1FB42: 0xFFFFE8, 0x1FB43: 0xFFFEEC,
	0x1FB44: 0xFFFE80, 0x1FB45: 0xFEEECC, 0x1FB46: 0xFFE800, 0x1FB47: 0x800000,
	0x1FB48: 0xE80000, 0x1FB49: 0xC88000, 0x1FB4A: 0xECC800, 0x1FB4B: 0xC88800,
	0x1FB4C: 0xFFFF73, 0x1FB4D: 0xFFFF71, 0x1FB4E: 0xFFF773, 0x1FB4F: 0xFFF710,
	0x1FB50: 0xF77733, 0x1FB51: 0xFF7100, 0x1FB52: 0xCEFFFF, 0x1FB53: 0x8EFFFF,
	0x1FB54: 0xCEEFFF, 0x1FB55: 0x8CCEFF, 0x1FB56: 0xCCEEEF, 0x1FB57: 0x000001,
	0x1FB58: 0x000017, 0x1FB59: 0x000113, 0x1FB5A: 0x000137, 0x1FB5B: 0x001113,
	0x1FB5C: 0x0017FF, 0x1FB5D: 0x77FFFF, 0x1FB5E: 0x17FFFF, 0x1FB5F: 0x377FFF,
	0x1FB60: 0x1137FF, 0x1FB61: 0x33777F, 0x1FB62: 0x000008, 0x1FB63: 0x00008E,
	0x1FB64: 0x00088C, 0x1FB65: 0x0008CE, 0x1FB66: 0x00888C, 0x1FB67: 0x008EFF,
}

// wedgeShapes are the wedges plus the sextants, all on the 4×6 grid: the
// sextants draw the straight edges and flat areas, the wedges the slopes.
var wedgeShapes = func() map[rune]uint32 {
	shapes := map[rune]uint32{}
	for r, m := range wedges {
		shapes[r] = m
	}
	for m, r := range sextants {
		var big uint32
		for k := 0; k < 24; k++ {
			if m>>((k/4/2)*2+(k%4)/2)&1 == 1 {
				big |= 1 << k
			}
		}
		shapes[r] = big
	}
	return shapes
}()

// wedgeCell picks the shape that splits the cell's 24 samples into the two
// most even-toned groups, the brighter one drawn as the glyph.
func wedgeCell(c cell) string {
	px, as := c.grid(4, 6)
	// An unlit cell stays empty: else the full block wins on the faint glow
	// round the picture and paints it a near-black box.
	lit := false
	for _, p := range px {
		lit = lit || p.r+p.g+p.b > 12
	}
	if !lit {
		return " "
	}
	best, bestE := ' ', math.Inf(1)
	for r, m := range wedgeShapes {
		var in, out, nin, nout float64
		for k, a := range as {
			if m>>k&1 == 1 {
				in, nin = in+a, nin+1
			} else {
				out, nout = out+a, nout+1
			}
		}
		if nin == 0 || (nout > 0 && in/nin < out/nout) {
			continue
		}
		e := 0.0
		for k, a := range as {
			ref := out / math.Max(nout, 1)
			if m>>k&1 == 1 {
				ref = in / nin
			}
			e += (a - ref) * (a - ref)
		}
		if e < bestE || (e == bestE && r < best) { // ties broken by rune, so frames don't flicker
			best, bestE = r, e
		}
	}
	var f, b rgb
	var nf, nb float64
	for k, p := range px {
		if wedgeShapes[best]>>k&1 == 1 {
			f, nf = rgb{f.r + p.r, f.g + p.g, f.b + p.b}, nf+1
		} else {
			b, nb = rgb{b.r + p.r, b.g + p.g, b.b + p.b}, nb+1
		}
	}
	if nf == 0 {
		return " "
	}
	f = rgb{f.r / nf, f.g / nf, f.b / nf}
	if nb > 0 {
		b = rgb{b.r / nb, b.g / nb, b.b / nb}
	}
	return twoTone(f, b, best)
}

// shadeCell is the 80s look: four densities of shading.
func shadeCell(c cell) string {
	col, a := c.mean()
	k := int(a * 4.99)
	if k == 0 {
		return " "
	}
	return fg(vivid(col), 0.45+0.55*a) + string([]rune(" ░▒▓█")[k])
}

// barsCell fills each cell from the bottom up to its brightness, like an
// equaliser.
func barsCell(c cell) string {
	col, a := c.mean()
	k := int(a * 8.99)
	if k == 0 {
		return " "
	}
	return fg(vivid(col), 0.5+0.5*a) + string([]rune(" ▁▂▃▄▅▆▇█")[k])
}

// halftoneCell is a printed photo: round dots on a screen turned 45°, each
// growing with the light until they touch, like a newspaper's. Drawn in
// octants, so each dot is a solid patch (braille dots read as dotted lines,
// not as print). An octant pixel is lit when it falls inside its screen
// dot, whose area is the brightness: radius² · π = a, on a unit screen cell.
func halftoneCell(c cell) string {
	const period = 6.0 // pixels between screen dots
	var mask int
	var sum rgb
	lum := 0.0
	for k := 0; k < 8; k++ {
		dx, dy := k%2, k/2
		col, a := c.at((float64(dx)+0.5)/2, (float64(dy)+0.5)/4)
		sum, lum = rgb{sum.r + col.r/8, sum.g + col.g/8, sum.b + col.b/8}, lum+math.Min(a, 1)/8
		x, y := float64(2*c.i+dx), float64(4*c.j+dy)
		u, v := (x+y)/(period*math.Sqrt2), (x-y)/(period*math.Sqrt2)
		fu, fv := u-math.Round(u), v-math.Round(v)
		if 0.75*math.Pow(a, 1.3) > math.Pi*(fu*fu+fv*fv) {
			mask |= 1 << k
		}
	}
	if mask == 0 {
		return " "
	}
	return fg(vivid(sum), 0.55+0.45*math.Sqrt(lum)) + string(octants[mask])
}

// contourCell draws only where brightness crosses one of six levels, like
// the lines of a relief map; marching squares on the cell's corners picks
// the line's way through.
func contourCell(c cell) string {
	var band [4]int
	var sum rgb
	total := 0.0
	lo := 99
	for k := range band {
		col, a := c.at(float64(k%2), float64(k/2))
		band[k] = int(math.Min(a, 0.999) * 6)
		lo = min(lo, band[k])
		sum, total = rgb{sum.r + col.r/4, sum.g + col.g/4, sum.b + col.b/4}, total+a/4
	}
	mask := 0
	for k, b := range band {
		if b > lo {
			mask |= 1 << k
		}
	}
	// Corners above the lowest level, by bit: top left, top right, bottom
	// left, bottom right. The line runs between the two groups.
	r := map[int]rune{
		1: '╯', 2: '╰', 4: '╮', 8: '╭', 14: '╯', 13: '╰', 11: '╮', 7: '╭',
		3: '─', 12: '─', 5: '│', 10: '│', 6: '╳', 9: '╳',
	}[mask]
	if r == 0 {
		return " "
	}
	return fg(vivid(sum), 0.4+0.6*math.Min(1, total+0.2)) + string(r)
}

// engraveCell hatches along the forms, the way an engraving follows a face:
// strokes run across the brightness gradient, crossed where the light is
// flat and strong.
func engraveCell(c cell) string {
	_, as := c.grid(2, 2)
	col, a := c.mean()
	if a < 0.12 {
		return " "
	}
	gx := (as[1] + as[3] - as[0] - as[2]) / 2
	gy := (as[2] + as[3] - as[0] - as[1]) / 2
	var r rune
	switch {
	case math.Hypot(gx, gy) < 0.04:
		r = map[bool]rune{true: '╳', false: '╱'}[a > 0.6]
	default:
		// The stroke runs along the contour, at right angles to the
		// gradient; a cell is twice as tall as wide, so y counts double.
		ang := math.Mod(math.Atan2(gx, -gy*2)+2*math.Pi, math.Pi)
		r = []rune{'─', '╲', '│', '╱'}[int(math.Round(ang/(math.Pi/4)))%4]
	}
	return fg(vivid(col), 0.35+0.65*a) + string(r)
}

var katakana = []rune("ｦｱｳｴｵｶｷｹｺｻｼｽｾｿﾀﾂﾃﾅﾆﾇﾈﾊﾋﾎﾏﾐﾑﾒﾓﾔﾕﾗﾘﾜ01")

// matrixCell is falling code shown only where the picture is: each column
// scrolls at its own speed, a bright head and a fading trail.
func matrixCell(c cell) string {
	col, a := c.mean()
	if a < 0.06 {
		return " "
	}
	pos := c.t*(4+8*hash(c.i, 0)) + 40*hash(c.i, 1)
	glyph := katakana[int(hash(c.i, c.j-int(pos))*float64(len(katakana)))%len(katakana)]
	trail := math.Mod(pos-float64(c.j), 24) / 24
	if trail < 0 {
		trail++
	}
	if trail < 1.0/24 { // the head of the drop
		col = lerpC(vivid(col), rgb{255, 255, 255}, 0.6)
		return fg(col, 0.6+0.4*a) + string(glyph)
	}
	return fg(vivid(col), a*(0.25+0.75*(1-trail))) + string(glyph)
}

// geometricCell is a mosaic of solid shapes: triangles where an edge cuts
// the cell, diamonds and discs where it is filled.
func geometricCell(c cell) string {
	_, as := c.grid(2, 2)
	col, a := c.mean()
	mask := 0
	for k, v := range as {
		if v > 0.3 {
			mask |= 1 << k
		}
	}
	r := map[int]rune{
		15: '◆', 7: '◤', 11: '◥', 13: '◣', 14: '◢',
		3: '▼', 12: '▲', 5: '▶', 10: '◀', 6: '◆', 9: '◆',
		1: '•', 2: '•', 4: '•', 8: '•',
	}[mask]
	if mask == 15 && a > 0.65 {
		r = '●'
	}
	if r == 0 {
		return " "
	}
	return fg(vivid(col), 0.4+0.6*a) + string(r)
}
