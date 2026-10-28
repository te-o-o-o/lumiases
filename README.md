<h1 align="center">lumiases</h1>

<p align="center">A living orb that watches Claude Code work, so you don't have to.</p>

<p align="center"><sub>v0.1.0 · Go, zero dependencies · macOS (Linux and Windows untested) · MIT</sub></p>

<p align="center">
  <a href="#features">Features</a> ·
  <a href="#install">Install</a> ·
  <a href="#usage">Usage</a> ·
  <a href="#follow-claude-code">Follow Claude Code</a> ·
  <a href="#statue-mode">Statue mode</a> ·
  <a href="#intro">Intro</a> ·
  <a href="#development">Development</a> ·
  <a href="README.fr.md">Français</a>
</p>

lumiases draws an orb of living matter in a terminal pane, meant to sit next
to a Claude Code session: its shape, speed and colours change with what Claude
is doing, through Claude Code's hooks. On its own it cycles through its
states or follows the keys.

## Features

- **Ten renderings.** Braille (8 dots per cell, dithered into fine dust) and
  ASCII, where each glyph is picked by its shape (`"` for a lit top, `(` for a
  lit left side, `/` on a diagonal) so contours inside the picture survive.
  Plus looks to try: wedge (diagonal blocks), shade, bars, halftone, contour (relief-map
  lines), engrave (hatching along the forms), matrix and geometric.
- **Eight states.** idle, thinking, reading, editing, running, waiting, error
  and done each have their own outline, speed, palette and gradient; the orb
  morphs from one to the next in about 300 ms.
- **Eyes, in five shapes.** round, slits, void, glasses and shades, carved into
  the matter and moving with it, with an expression per state: they squint
  while thinking, sweep while reading, open wide while waiting, frown on an
  error and smile when done.
- **Ten scenes drawn from code alone.** Besides the orb: a swarm of dots, and
  line drawings in the manner of scientific plots (pulsar ridge lines, an
  oscilloscope, field lines, a contour map, interfering waves, a spectrum,
  orbits, a flow field), each with a figure per state.
- **Statue mode.** The matter flows over a photo of a statue: the head breathes
  and sways, and each state adds its effect (rising smoke, a scanning band of
  light, glitches, ripples, a shine).
- **A GIF intro.** The GIFs of a folder play at start as real pictures in
  terminals that show them (Ghostty, kitty, WezTerm), plain or through a
  contrast or digital filter.
- **One small binary.** Go standard library only, about 30 frames a second on a
  200×60 pane.

## Install

Download the binary for your system from the
[releases](https://github.com/te-o-o-o/lumiases/releases), name it
`lumiases`, make it executable and put it on your `PATH`:

```
chmod +x lumiases-darwin-arm64 && mv lumiases-darwin-arm64 /usr/local/bin/lumiases
```

The binaries are not signed. On macOS, clear the quarantine flag once with
`xattr -d com.apple.quarantine /usr/local/bin/lumiases`; on Windows, pick
"More info" then "Run anyway" if SmartScreen stops it.

From source, with Go 1.22 or later:

```
go install github.com/te-o-o-o/lumiases@latest
```

or from a clone: `go run .`

## Usage

```
lumiases                      # braille, cycling through the states every 4 s
lumiases -style ascii
lumiases -state thinking      # hold one state
lumiases -follow              # follow Claude Code (see below)
lumiases -eyes glasses        # round, slits, void, glasses, shades
lumiases -form pulsar         # orb swarm pulsar scope field topo waves spectrum orbits flow
```

| Key | Does |
|---|---|
| `s` | next style |
| `1`–`8` | pick a state |
| `c` | cycle through the states again |
| `+` / `-` | zoom in and out: more cells on a face is more detail; a smaller font in the pane does the same |
| `h` | next shape: orb, a swarm, pulsar ridge lines, an oscilloscope, field lines, a contour map, interfering waves, a spectrum, orbits, a flow field, then the statue heads |
| `r` | turn the head round (statue mode) |
| `e` | next eye shape |
| `f` | show or hide the eyes |
| space | pause |
| `q` | quit |

The picture fills the pane and follows its size. A smaller font in that pane
means more cells, so a finer orb.

## Follow Claude Code

Claude Code runs `lumiases hook` on each event; the hook writes the state it
reads into a small file, and `lumiases -follow`, in a pane next to Claude,
plays it. Install the binary (`go install .` from a clone puts it in
`~/go/bin`, which must be on your `PATH`), then add the hooks to
`~/.claude/settings.json`, or to a project's `.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart":       [{ "hooks": [{ "type": "command", "command": "lumiases hook" }] }],
    "UserPromptSubmit":   [{ "hooks": [{ "type": "command", "command": "lumiases hook" }] }],
    "PreToolUse":         [{ "matcher": "*", "hooks": [{ "type": "command", "command": "lumiases hook" }] }],
    "PostToolUse":        [{ "matcher": "*", "hooks": [{ "type": "command", "command": "lumiases hook" }] }],
    "PostToolUseFailure": [{ "matcher": "*", "hooks": [{ "type": "command", "command": "lumiases hook" }] }],
    "Notification":       [{ "hooks": [{ "type": "command", "command": "lumiases hook" }] }],
    "Stop":               [{ "hooks": [{ "type": "command", "command": "lumiases hook" }] }],
    "SessionEnd":         [{ "hooks": [{ "type": "command", "command": "lumiases hook" }] }]
  }
}
```

| Claude | lumiases |
|---|---|
| reads, searches, browses | reading |
| edits or writes a file | editing |
| runs a command or another tool | running |
| gets your prompt, between tools, runs a subagent | thinking |
| waits for you (permission, question) | waiting |
| a tool fails | error |
| finishes its answer | done, then idle after 5 s |

With several sessions open, the pane follows the latest event, whichever
session it came from.

## Statue mode

```
lumiases heads/*.png                         # every head, h switches between them and the orb
lumiases -face bust.png -style ascii
lumiases -face statue.jpg -crop 0.2,0.1,0.82,0.66
```

Give photos of statues with `-face` (repeat it) or after the flags; `h` goes
from one head to the next and back to the orb, `r` turns the head round.
Best is a PNG with the background removed (macOS: Preview or Photos,
"Remove Background", or `swift tools/cutout.swift photo.jpg heads/bust.png 0 0 1 0.7`
to cut out and keep the top 70 %); without transparency, a plain background
(white or any even colour, read from the picture's border) is keyed out, a dark
drawing on a light page has its ink turned into light, and otherwise warm stone
is told apart from greenery. `-crop` keeps a part of
each picture, as `x0,y0,x1,y1` fractions. No photo ships with lumiases: the
Metropolitan Museum's open-access pictures (CC0) are a good source.

## Intro

At start, lumiases plays the GIFs in its intro folder, in name order, then
goes on to the orb; any key skips them. The folder is `lumiases/intro` in the
user config folder (macOS `~/Library/Application Support`, Linux
`~/.config`, Windows `%AppData%`), or the one given with `-intro`:

```
lumiases -intro ~/my-gifs -intro-fx digital
```

Each GIF plays once, looped up to at least 2.5 s, as a real picture through
the kitty graphics protocol (Ghostty, kitty, WezTerm; other terminals show
nothing). `-intro-fx` picks its look: `contrast` (the default) pushes tones
and colours apart, `digital` posterizes and dithers it with scanlines and a
red/blue split, `none` leaves it as it is.

## Development

```
go vet ./... && go test ./...
go run . -style ascii
COLUMNS=110 LINES=50 go run . -state error -frames 45 < /dev/null > frame.ansi   # capture frames without a terminal
COLUMNS=80 LINES=34 go run . -form pulsar -state thinking -frames 90 < /dev/null | go run ./tools/readme-gif > docs/pulsar.gif
```

Decisions, dead ends and what was checked on which system: [docs/notes.md](docs/notes.md).

## License

[MIT](LICENSE)
