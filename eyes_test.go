package main

import "testing"

func TestEveryStateHasEyes(t *testing.T) {
	for _, s := range states {
		if _, ok := stateEyes[s.name]; !ok {
			t.Errorf("state %q has no entry in stateEyes", s.name)
		}
	}
}

func TestEyesCarveTheOrb(t *testing.T) {
	e := stateEyes["idle"]
	e.shown = 1
	for _, style := range eyeStyles {
		eyeStyle = style
		carve, glow := 0.0, 0.0
		for x := -0.6; x <= 0.6; x += 0.01 {
			for y := -0.4; y <= 0.2; y += 0.01 {
				c, g := eyes(x, y, 1, e)
				carve, glow = max(carve, c), max(glow, g)
			}
		}
		if carve < 0.9 {
			t.Errorf("%s: eyes never carve the orb (max %.2f)", style, carve)
		}
		if style != "round" && glow < 0.5 {
			t.Errorf("%s: nothing glows (max %.2f)", style, glow)
		}
	}
	eyeStyle = "round"
}
