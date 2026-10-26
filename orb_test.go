package main

import "testing"

func TestSampleInsideAndOutside(t *testing.T) {
	for _, s := range states {
		if _, i, d := sample(0, 0, 1, 1, s.p); d >= 1 || i <= 0 {
			t.Errorf("%s: centre should be lit inside the orb, got brightness %.2f at distance %.2f", s.name, i, d)
		}
		if _, i, _ := sample(5, 5, 1, 1, s.p); i > 0.001 {
			t.Errorf("%s: far outside should be dark, got %.3f", s.name, i)
		}
	}
}

func TestEaseReachesTarget(t *testing.T) {
	a, b := states[0].p, states[6].p
	if got := ease(a, b, 1); got != b {
		t.Errorf("ease(k=1) = %+v, want %+v", got, b)
	}
	if got := ease(a, b, 0); got != a {
		t.Errorf("ease(k=0) = %+v, want %+v", got, a)
	}
}

func TestFormsCoverStatesAndStayInside(t *testing.T) {
	defer func() { form = "orb" }()
	for _, s := range states {
		if _, ok := stateAgent[s.name]; !ok {
			t.Errorf("state %s has no stateAgent entry", s.name)
		}
	}
	for _, form = range forms {
		p := states[0].p
		p.agent = stateAgent["idle"]
		if _, i, _ := sample(0.05, 0.4, 1, 1, p); i <= 0 {
			t.Errorf("%s: should be lit near the centre", form)
		}
		if _, i, _ := sample(5, 5, 1, 1, p); i > 0.001 {
			t.Errorf("%s: far outside should be dark, got %.3f", form, i)
		}
	}
}
