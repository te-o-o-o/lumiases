# lumiases

A living orb (or a statue) animated in ASCII, braille or pixels in a terminal pane,
meant to follow a Claude Code session. Go, stdlib only, one `main` package.
Claude Code hooks run `lumiases hook`; `lumiases -follow` plays the state.
Decisions and traps in detail: `docs/notes.md`.

## Layout

| File | Role | Rule |
|---|---|---|
| `main.go` | Flags, keys, the 30 fps loop, terminal size | A state's target is `states` + `stateEyes` + `stateStatue`, eased with `ease`. |
| `orb.go` | `params`, the 8 `states`, noise, `sample` (the orb), `sparks` | `sample` works in square units, orb about 1 wide; phase `ph`, not time, drives motion. |
| `eyes.go` | `eyeParams`, `stateEyes`, the five eye shapes | Eyes return carve and glow; they never draw glyphs themselves. |
| `statue.go` | `statueParams`, `stateStatue`, photo loading and keying, `sampleFace` | Effects bend where the photo is read, never the photo. |
| `render.go` | `frame`: braille dithering, ASCII glyph matching, quarter-block and octant pixels, kitty-protocol images | One goroutine per row; one write per frame inside synchronized output. |
| `hooks.go` | `lumiases hook` (Claude Code event → state file) and `readState` for `-follow` | The hook never fails and never waits: bad input leaves the state alone, exit 0. |
| `styles.go` | The looks drawn one cell at a time (`cellStyles`): sextant, wedge, shade, bars, halftone, contour, engrave, matrix, geometric | A style reads its cell through `cell.at`; adding one is an entry in `cellStyles` and `styleNames`. |
| `term_unix.go`, `term_windows.go` | `ttySize` | Windows has none yet. |
| `tools/cutout.swift` | Cuts a statue out of a photo into a transparent PNG (macOS Vision) | A helper to prepare heads, outside the Go build; heads live in the ignored `heads/`. |

Every new state needs an entry in `states`, `stateEyes` and `stateStatue`
(tests check it).

## Commands

```
go vet ./... && go test ./...        # before every commit
gofmt -l .                           # must print nothing (CI checks it)
GOOS=windows go vet . && GOOS=linux go vet .
go run . -style ascii -state thinking -eyes void
COLUMNS=110 LINES=50 go run . -frames 45 < /dev/null > out.ansi   # capture without a terminal
```

To look at a capture: turn the last frame (after the last `ESC[?2026h`) into
an HTML page of coloured spans, then `qlmanage -t -s 900 page.html` renders it
to PNG on macOS.

## Commit scopes

`orb`, `eyes`, `statue`, `ascii`, `braille`, `pixel`, `hooks`, and a system name
(`macos`, `linux`, `windows`) for a change that matters on one system only.

## Traps

- Judge a look from a rendered capture, not from reasoning: the first braille
  orb was nearly invisible and the first ASCII outline four glyphs thick, and
  neither showed until a capture was looked at.
- `eyeStyle` and `face` are globals read concurrently by the row goroutines:
  set them only between frames.
- The statue key only finds warm stone on green; anything else fails with
  "no statue found".
- Docs come in pairs: `README.md` / `README.fr.md`, `docs/notes.md` / `docs/notes.fr.md`.
