// lumiases animates a living orb in the terminal, meant to sit next to a
// Claude Code session and change with what Claude does.
//
// Keys: s next style, 1-8 state, h next head
// (statue mode), e eye shape, f eyes on/off, r turn the statue, c cycle
// states, space pause, q quit.
//
// `lumiases -follow` takes its state from Claude Code, whose hooks run
// `lumiases hook` (hooks.go).
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// pathList is a flag that can be given more than once.
type pathList []string

func (l *pathList) String() string     { return strings.Join(*l, ",") }
func (l *pathList) Set(s string) error { *l = append(*l, s); return nil }

func termSize() (int, int) {
	if w, h, ok := ttySize(); ok {
		return w, h
	}
	// Not a terminal (output captured): COLUMNS and LINES pick the size.
	w, _ := strconv.Atoi(os.Getenv("COLUMNS"))
	h, _ := strconv.Atoi(os.Getenv("LINES"))
	if w == 0 || h == 0 {
		return 60, 30
	}
	return w, h
}

// keys puts the terminal in raw mode and streams key presses. No terminal
// (output captured) or no stty (Windows): no keys, and restore does nothing.
func keys() (<-chan byte, func()) {
	ch := make(chan byte, 8)
	saved, err := stty("-g")
	if err != nil {
		return ch, func() {}
	}
	stty("raw", "-echo")
	go func() {
		b := make([]byte, 1)
		for {
			if n, _ := os.Stdin.Read(b); n == 1 {
				ch <- b[0]
			}
		}
	}()
	return ch, func() { stty(strings.TrimSpace(saved)) }
}

func stty(args ...string) (string, error) {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	return string(out), err
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "hook" {
		runHook(os.Stdin, statePath())
		return
	}
	style := flag.String("style", "braille", strings.Join(styleNames, " | ")+" (image: Ghostty, kitty, WezTerm)")
	state := flag.String("state", "", "fixed state (idle, thinking, reading, editing, running, waiting, error, done); empty cycles through all")
	follow := flag.Bool("follow", false, "follow Claude Code: take the state its hooks record (see `lumiases hook` in the README)")
	frames := flag.Int("frames", 0, "stop after n frames (0 = run until q)")
	flag.StringVar(&eyeStyle, "eyes", "round", strings.Join(eyeStyles, " | "))
	var facePaths pathList
	flag.Var(&facePaths, "face", "statue mode: a photo of a statue the matter flows over (best a PNG with its background removed), or a folder of frames from tools/turntable.py; repeat it, or list photos after the flags, and h switches between them and the orb")
	cropArg := flag.String("crop", "0,0,1,1", "statue mode: part of each photo to keep, as x0,y0,x1,y1 fractions")
	flag.Parse()

	var crop [4]float64
	for i, s := range strings.SplitN(*cropArg, ",", 4) {
		crop[i], _ = strconv.ParseFloat(strings.TrimSpace(s), 64)
	}
	var heads []*head
	var headNames []string
	for _, path := range append(facePaths, flag.Args()...) {
		f, err := loadHead(path, crop)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		heads, headNames = append(heads, f), append(headNames, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	}
	head := 0 // 0 is the orb, i the heads[i-1]
	if len(heads) > 0 {
		head, face = 1, heads[0]
	}

	cur, cycle, eyesOn, spinOn := 0, *state == "" && !*follow, true, false
	var seen time.Time // when the followed state last changed
	for i, s := range states {
		if s.name == *state {
			cur = i
		}
	}
	p := states[cur].p

	kb, restoreTTY := keys()
	fmt.Print("\x1b[?1049h\x1b[?25l")
	restore := func() { fmt.Print(kittyClear + "\x1b[0m\x1b[?25h\x1b[?1049l"); restoreTTY() }
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() { <-sig; restore(); os.Exit(0) }()
	defer restore()

	last, t, ph, paused, cycleAt := time.Now(), 0.0, 0.0, false, 0.0
	tick := time.NewTicker(time.Second / 30)
	for n := 0; *frames == 0 || n < *frames; n++ {
		now := <-tick.C
		dt := now.Sub(last).Seconds()
		last = now
		for len(kb) > 0 {
			switch k := <-kb; {
			case k == 'q' || k == 3: // 3 = ctrl-c, which raw mode delivers as a byte
				return
			case k == 's':
				*style = styleNames[(slices.Index(styleNames, *style)+1)%len(styleNames)]
				fmt.Print(kittyClear)
			case k == 'c':
				cycle, cycleAt = !cycle, t
			case k == 'e':
				i := 0
				for j, s := range eyeStyles {
					if s == eyeStyle {
						i = j
					}
				}
				eyeStyle = eyeStyles[(i+1)%len(eyeStyles)]
			case k == 'f':
				eyesOn = !eyesOn
			case k == 'r':
				spinOn = !spinOn
			case k == 'h':
				head, face = (head+1)%(len(heads)+1), nil
				if head > 0 {
					face = heads[head-1]
				}
			case k == ' ':
				paused = !paused
			case k >= '1' && k <= '8':
				cur, cycle = int(k-'1'), false
			}
		}
		if paused {
			continue
		}
		t += dt
		if *follow && n%5 == 0 { // the file is tiny, but 6 checks a second are plenty
			if s, at, ok := readState(statePath()); ok {
				if !at.Equal(seen) {
					seen = at
					for i, st := range states {
						if st.name == s {
							cur = i
						}
					}
				}
				// Done celebrates, then rests until Claude does something else.
				if states[cur].name == "done" && time.Since(at) > 5*time.Second {
					cur = 0
				}
			}
		}
		if cycle && t-cycleAt > 4 {
			cur, cycleAt = (cur+1)%len(states), t
		}
		k := 1 - math.Exp(-dt*3) // ~300 ms morph between states
		want := states[cur].p
		want.eyes, want.statue = stateEyes[states[cur].name], stateStatue[states[cur].name]
		p = ease(p, want, k)
		p.eyes.shown = lerp(p.eyes.shown, map[bool]float64{true: 1}[eyesOn], k)
		if spin := map[bool]float64{true: 0.8}[spinOn] + p.statue.spin; spin > 0.01 {
			p.statue.angle += dt * spin
		} else { // the spin is over: come back to the nearest front view
			front := math.Round(p.statue.angle/(2*math.Pi)) * 2 * math.Pi
			p.statue.angle = lerp(p.statue.angle, front, k)
		}
		// A blast runs once, slowly (~3 s), from entering the state; leaving
		// it resets at once, or easing back would burst a second time.
		if want.statue.blast > 0 {
			p.statue.blast = lerp(p.statue.blast, 1, 1-math.Exp(-dt*0.8))
		} else {
			p.statue.blast = 0
		}
		ph += dt * p.speed
		w, h := termSize()
		label := fmt.Sprintf(" %s · %s", states[cur].name, *style)
		hint := "   s style · 1-8 state · e eye shape · f eyes on/off · c cycle · space pause · q quit"
		if face != nil {
			label += " · " + headNames[head-1]
			hint = "   s style · 1-8 state · r turn · c cycle · space pause · q quit"
		}
		if len(heads) > 0 {
			hint = "   h head ·" + strings.TrimPrefix(hint, "  ")
		}
		os.Stdout.WriteString(frame(w, h, t, ph, p, *style, label+hint))
	}
}
