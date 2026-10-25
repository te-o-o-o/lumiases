// lumiases animates a living orb in the terminal, meant to sit next to a
// Claude Code session and change with what Claude does.
//
// Keys: s style, 1-8 state, c cycle states, space pause, q quit.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

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
	style := flag.String("style", "braille", "braille | ascii")
	state := flag.String("state", "", "fixed state (idle, thinking, reading, editing, running, waiting, error, done); empty cycles through all")
	frames := flag.Int("frames", 0, "stop after n frames (0 = run until q)")
	flag.Parse()

	cur, cycle := 0, *state == ""
	for i, s := range states {
		if s.name == *state {
			cur = i
		}
	}
	p := states[cur].p

	kb, restoreTTY := keys()
	fmt.Print("\x1b[?1049h\x1b[?25l")
	restore := func() { fmt.Print("\x1b[0m\x1b[?25h\x1b[?1049l"); restoreTTY() }
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
				*style = map[string]string{"braille": "ascii", "ascii": "braille"}[*style]
			case k == 'c':
				cycle, cycleAt = !cycle, t
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
		if cycle && t-cycleAt > 4 {
			cur, cycleAt = (cur+1)%len(states), t
		}
		k := 1 - math.Exp(-dt*3) // ~300 ms morph between states
		p = ease(p, states[cur].p, k)
		ph += dt * p.speed
		w, h := termSize()
		hint := "   s style · 1-8 state · c cycle · space pause · q quit"
		os.Stdout.WriteString(frame(w, h, t, ph, p, *style, fmt.Sprintf(" %s · %s%s", states[cur].name, *style, hint)))
	}
}
