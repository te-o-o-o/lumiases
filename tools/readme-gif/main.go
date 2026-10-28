// readme-gif turns a braille capture of lumiases into an animated GIF for
// the README. It draws the braille dots themselves, so no font is needed;
// every GIF of one capture size comes out the same size.
//
//	COLUMNS=80 LINES=34 go run . -form pulsar -state thinking -frames 90 < /dev/null |
//		go run ./tools/readme-gif > docs/pulsar.gif
package main

import (
	"flag"
	"image"
	"image/color"
	"image/gif"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// A cell is drawn 8×16 pixels, like a terminal's, its 2×4 dots 3×3 squares.
const cw, ch, dot = 8, 16, 3

var bg = color.RGBA{13, 13, 18, 255}

// dotBits maps braille bit k to its column and row in the cell.
var dotBits = [8][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}, {0, 3}, {1, 3}}

var sgr = regexp.MustCompile(`\x1b\[([0-9;?]*)([a-zA-Z])`)

type lit struct {
	x, y int
	c    color.RGBA
}

func main() {
	step := flag.Int("step", 2, "keep one frame in step (captures run at 30 fps)")
	flag.Parse()
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	var frames [][]lit
	cols, rows := 0, 0
	for i, f := range strings.Split(string(raw), "\x1b[?2026h")[1:] {
		if i%*step != 0 {
			continue
		}
		lines := strings.Split(strings.ReplaceAll(f, "\r", ""), "\n")
		lines = lines[:len(lines)-1] // the last row is the label
		rows = max(rows, len(lines))
		var dots []lit
		for y, row := range lines {
			fg, x := color.RGBA{204, 204, 204, 255}, 0
			for len(row) > 0 {
				if m := sgr.FindStringSubmatchIndex(row); m != nil && m[0] == 0 {
					if p := strings.Split(row[m[2]:m[3]], ";"); row[m[4]:m[5]] == "m" && len(p) == 5 && p[0] == "38" {
						r, _ := strconv.Atoi(p[2])
						g, _ := strconv.Atoi(p[3])
						b, _ := strconv.Atoi(p[4])
						fg = color.RGBA{uint8(r), uint8(g), uint8(b), 255}
					}
					row = row[m[1]:]
					continue
				}
				r := []rune(row)[0]
				row = row[len(string(r)):]
				if r >= 0x2800 && r <= 0x28ff {
					for k, cr := range dotBits {
						if (r-0x2800)&(1<<k) != 0 {
							px, py := x*cw+cr[0]*cw/2+1, y*ch+cr[1]*ch/4+1
							dots = append(dots, lit{px, py, fg})
						}
					}
				}
				x++
			}
			cols = max(cols, x)
		}
		frames = append(frames, dots)
	}
	if len(frames) == 0 {
		panic("no braille frames on stdin")
	}

	// A fixed palette: the background, then a 6×7×6 colour cube.
	pal := color.Palette{bg}
	for r := 0; r < 6; r++ {
		for g := 0; g < 7; g++ {
			for b := 0; b < 6; b++ {
				pal = append(pal, color.RGBA{uint8(r * 51), uint8(g * 42), uint8(b * 51), 255})
			}
		}
	}
	rect := image.Rect(0, 0, cols*cw, rows*ch)
	out := &gif.GIF{}
	for _, dots := range frames {
		im := image.NewPaletted(rect, pal)
		for _, d := range dots {
			idx := uint8(pal.Index(d.c))
			for dy := 0; dy < dot; dy++ {
				for dx := 0; dx < dot; dx++ {
					im.SetColorIndex(d.x+dx, d.y+dy, idx)
				}
			}
		}
		out.Image, out.Delay = append(out.Image, im), append(out.Delay, 100**step/30)
	}
	if err := gif.EncodeAll(os.Stdout, out); err != nil {
		panic(err)
	}
}
