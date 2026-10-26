package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Claude Code runs `lumiases hook` on each event with the event as JSON on
// stdin. The hook turns it into a state and writes it to a file the animator
// (`lumiases -follow`) watches. A file, not a socket: the hook never waits on
// the animator, works with none running, and the animator can start late.
// ponytail: one file for every session, the latest event wins; key it by
// session_id if two sessions side by side should each have their own pane.

// hookEvent holds the fields of a hook's input that pick a state.
type hookEvent struct {
	Event string `json:"hook_event_name"`
	Tool  string `json:"tool_name"`
}

// toolStates says what Claude is doing while a tool runs; any other tool
// (MCP tools included) counts as running.
var toolStates = map[string]string{
	"Read": "reading", "Grep": "reading", "Glob": "reading", "LS": "reading",
	"WebFetch": "reading", "WebSearch": "reading",
	"Edit": "editing", "MultiEdit": "editing", "Write": "editing", "NotebookEdit": "editing",
	"Task": "thinking", "Agent": "thinking", "TodoWrite": "thinking",
}

// stateFor maps a hook event to a state; ok is false for events that should
// leave the animation as it is.
func stateFor(e hookEvent) (state string, ok bool) {
	switch e.Event {
	case "SessionStart", "SessionEnd":
		return "idle", true
	case "UserPromptSubmit", "PostToolUse", "SubagentStop", "PreCompact":
		return "thinking", true
	case "PreToolUse":
		if s, ok := toolStates[e.Tool]; ok {
			return s, true
		}
		return "running", true
	case "PostToolUseFailure", "StopFailure":
		return "error", true
	case "Notification", "PermissionRequest":
		return "waiting", true
	case "Stop":
		return "done", true
	}
	return "", false
}

func statePath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "lumiases", "state")
}

// runHook reads one event and records its state. It never fails loudly: a
// hook that errors would show up in Claude Code for nothing the user did.
func runHook(in io.Reader, path string) {
	var e hookEvent
	if json.NewDecoder(in).Decode(&e) != nil {
		return
	}
	state, ok := stateFor(e)
	if !ok {
		return
	}
	os.MkdirAll(filepath.Dir(path), 0o755)
	// Write then rename, so the animator never reads half a file.
	tmp := path + ".tmp"
	if os.WriteFile(tmp, []byte(state+"\n"), 0o644) == nil {
		os.Rename(tmp, path)
	}
}

// readState returns the state last recorded and when, or ok false if there
// is none yet.
func readState(path string) (state string, at time.Time, ok bool) {
	st, err := os.Stat(path)
	if err != nil {
		return "", time.Time{}, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", time.Time{}, false
	}
	return strings.TrimSpace(string(b)), st.ModTime(), true
}
