package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestStateForEvents(t *testing.T) {
	for _, c := range []struct {
		event, tool, want string
	}{
		{"UserPromptSubmit", "", "thinking"},
		{"PreToolUse", "Read", "reading"},
		{"PreToolUse", "Edit", "editing"},
		{"PreToolUse", "Bash", "running"},
		{"PreToolUse", "mcp__github__create_issue", "running"},
		{"PostToolUseFailure", "Bash", "error"},
		{"Notification", "", "waiting"},
		{"Stop", "", "done"},
	} {
		if got, _ := stateFor(hookEvent{c.event, c.tool}); got != c.want {
			t.Errorf("%s %s: got %q, want %q", c.event, c.tool, got, c.want)
		}
	}
	if _, ok := stateFor(hookEvent{Event: "SomethingNew"}); ok {
		t.Error("an unknown event should leave the state alone")
	}
}

func TestEveryHookStateExists(t *testing.T) {
	names := map[string]bool{}
	for _, s := range states {
		names[s.name] = true
	}
	for tool, s := range toolStates {
		if !names[s] {
			t.Errorf("%s maps to %q, which is no state", tool, s)
		}
	}
}

func TestHookWritesTheState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "state")
	runHook(strings.NewReader(`{"hook_event_name":"PreToolUse","tool_name":"Edit","session_id":"x"}`), path)
	if s, _, ok := readState(path); !ok || s != "editing" {
		t.Fatalf("got %q (ok %v), want editing", s, ok)
	}
	runHook(strings.NewReader(`not json`), path)
	runHook(strings.NewReader(`{"hook_event_name":"SomethingNew"}`), path)
	if s, _, _ := readState(path); s != "editing" {
		t.Errorf("bad or unknown input changed the state to %q", s)
	}
}
