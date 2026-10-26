<h1 align="center">lumiases</h1>

<p align="center">A living orb that watches Claude Code work, so you don't have to.</p>

<p align="center"><sub>prototype · Go 1.22+, zero dependencies · macOS · MIT</sub></p>

<p align="center">
  <a href="#features">Features</a> ·
  <a href="#install">Install</a> ·
  <a href="#usage">Usage</a> ·
  <a href="#follow-claude-code">Follow Claude Code</a> ·
  <a href="#statue-mode">Statue mode</a> ·
  <a href="#development">Development</a> ·
  <a href="README.fr.md">Français</a>
</p>

<table align="center">
  <tr>
    <td align="center"><img src="docs/braille-editing.png" width="330" alt="An orange and red orb drawn in braille dots, lumpy and churning, with two dark eyes looking down"><br><sub>Braille, editing</sub></td>
    <td align="center"><img src="docs/ascii-waiting.png" width="330" alt="An amber orb drawn in ASCII characters with two wide-open eyes carved into it"><br><sub>ASCII, waiting</sub></td>
  </tr>
  <tr>
    <td align="center"><img src="docs/braille-idle.png" width="330" alt="A blue braille orb with two hollow sockets, a pinpoint of light in each"><br><sub>Braille, idle, void eyes</sub></td>
    <td align="center"><img src="docs/ascii-thinking.png" width="330" alt="A violet ASCII orb wearing round glasses, the frames drawn in bright characters"><br><sub>ASCII, thinking, glasses</sub></td>
  </tr>
</table>

lumiases draws an orb of living matter in a terminal pane, meant to sit next
to a Claude Code session: its shape, speed and colours change with what Claude
is doing. Today it is the renderer alone, cycling through its states or driven
by keys; wiring it to Claude Code's hooks comes next.

## Features

- **Fourteen renderings.** Braille (8 dots per cell, dithered into fine dust) and
  ASCII, where each glyph is picked by its shape (`"` for a lit top, `(` for a
  lit left side, `/` on a diagonal) so contours inside the picture survive;
  pixel, four pixels per cell in quarter blocks, for smooth gradients;
  octant, braille's eight points per cell in solid blocks (a recent font or a
  terminal that draws blocks itself, like Ghostty); and
  image, a real picture for terminals that show them (Ghostty, kitty, WezTerm
  with `enable_kitty_graphics = true`). Plus looks to try: sextant and wedge
  (blocks and diagonal blocks), shade, bars, halftone, contour (relief-map
  lines), engrave (hatching along the forms), matrix and geometric.
- **Eight states.** idle, thinking, reading, editing, running, waiting, error
  and done each have their own outline, speed, palette and gradient; the orb
  morphs from one to the next in about 300 ms.
- **Eyes, in five shapes.** round, slits, void, glasses and shades, carved into
  the matter and moving with it, with an expression per state: they squint
  while thinking, sweep while reading, open wide while waiting, frown on an
  error and smile when done.
- **Statue mode.** The matter flows over a photo of a statue: the head breathes
  and sways, and each state adds its effect (rising smoke, a scanning band of
  light, glitches, ripples, a shine).
- **One small binary.** Go standard library only, about 30 frames a second on a
  200×60 pane.

## Install

No release yet. With Go 1.22 or later:

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
lumiases -form tendrils       # orb, tendrils, eye, hood
```

| Key | Does |
|---|---|
| `s` | next style |
| `1`–`8` | pick a state |
| `c` | cycle through the states again |
| `+` / `-` | zoom in and out: more cells on a face is more detail; a smaller font in the pane does the same |
| `h` | next shape: orb, tendrils, one big eye, a hooded figure, then the statue heads |
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

## Development

```
go vet ./... && go test ./...
go run . -style ascii
COLUMNS=110 LINES=50 go run . -state error -frames 45 < /dev/null > frame.ansi   # capture frames without a terminal
```

Decisions, dead ends and what was checked on which system: [docs/notes.md](docs/notes.md).

## License

[MIT](LICENSE)
