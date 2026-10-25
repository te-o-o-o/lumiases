# Notes

Decisions, dead ends and traps, so they aren't paid for twice.

## Where this is going

lumiases is meant to sit in a pane next to Claude Code and follow it live:
a command hook (`lumiases hook`) relays each event (`PreToolUse`,
`PostToolUse`, `Notification`, `Stop`...) as one JSON line to the animator over
a local socket and exits at once, so a closed animator never slows Claude down.
The animator maps events to the eight states and tails the session transcript
for the phases no hook sees (thinking, writing). None of that is built yet: the
renderer came first.

## Rendering

- **Braille and ASCII only.** Half-block "plasma", topographic rings and
  metaballs were tried and dropped: they read as generic next to the other two.
- **Braille: density carries brightness, colour keeps its hue.** Dots are thin;
  averaging the cell's colours toward a dark rim made the first orb almost
  invisible. The colour is now scaled to full strength and the dot count
  does the shading.
- **Braille dithering: interleaved gradient noise, not Bayer.** An 8×8 Bayer
  matrix showed a visible grid; IGN spreads the dots evenly without one.
- **ASCII: glyph by shape.** Each cell is sampled on the same 2×4 grid as
  braille and gets the glyph whose ink pattern is closest; flat cells use a
  density ramp, or every flat grey area fills with `|`. The glyph shapes are
  estimated by eye (`glyphShapes`), not measured from a font.
- **No drawn outline.** Two ASCII outlines were tried (`| / - \` by angle,
  then `( ) _ ' .` placed where the edge crosses the cell) and dropped: the
  orb reads better with its edge fading into the halo.
- **No thorns on error.** Spikes on the outline looked aggressive; error now
  trembles and the eyes frown.
- **One goroutine per row.** Braille and ASCII take 8 samples per cell, each
  with several noise lookups; split by rows, a 200×60 pane holds about 30
  frames a second.
- **Particles are stateless.** A particle's age is a function of the phase,
  so nothing is stored between frames and they follow the state's speed.

## Statue mode

- **The photo never moves; where it is read does.** Sway, breathing, ripples,
  glitches and smoke all bend the coordinates before the lookup, so every
  effect composes with the others for free.
- **Transparency first, colour key as a fallback.** A cut-out PNG's alpha is
  the mask; without one, "red above green" keys terracotta on foliage and
  nothing else. Museum photos are cut out with macOS Vision (the engine
  behind "Remove Background"), cropped to head and neck, kept in the
  ignored `heads/` folder.
- **Turning: one cylinder per row, the back mirrored.** Mirroring makes the
  seams at ±90° meet; the profile is still empty (no photo saw it), hence
  the soft burst there. Real profiles need 3D models.
- **Faces need contrast more than pixels.** At 30 columns a head, the
  features only show with a strong unsharp mask and an S-curve; a smaller
  font in the pane helps more than any setting.
- **Unsharp mask on load.** A face gets few cells in a terminal; without
  pushing local contrast, the eyes, nose and lips melt into the cheeks.
- **No photo in the repository.** The one used while building it came from a
  stock site; check a picture's licence before shipping one.

## Traps

- Captures: `COLUMNS=110 LINES=50 go run . -frames 45 < /dev/null > out.ansi`.
  Without a terminal, the size comes from `COLUMNS` and `LINES`, and keys are
  off. Frames are separated by `ESC[?2026h`.
- `eyeStyle` and `face` are globals read by the row goroutines: change them
  only between frames, as `main` does.
- Smaller characters can't be asked for from the program; it's the pane's
  font size. Some terminals change it per pane, others per window.

## Checked on

| System | What |
|---|---|
| macOS (Darwin 27), Apple Silicon | Builds, tests pass, both styles and statue mode watched in a terminal, ~30 fps at 200×60 |
| Linux | `GOOS=linux go vet` only; never run |
| Windows | `GOOS=windows go vet` only; never run. No terminal size (falls back to `COLUMNS`/`LINES`), no keys (no `stty`) |
