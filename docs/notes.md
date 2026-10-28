# Notes

Decisions, dead ends and traps, so they aren't paid for twice.

## Where this is going

lumiases is meant to sit in a pane next to Claude Code and follow it live:
a command hook (`lumiases hook`) maps each event (`PreToolUse`, `Notification`,
`Stop`...) to one of the eight states and writes it to a file in the user cache
dir; `lumiases -follow` checks that file six times a second.

- A file, not the socket first planned: the hook never waits on the animator,
  works with none running, and an animator started late picks up the current
  state at once. Write then rename, so a half-written file is never read.
- The hook never fails: bad input or an unknown event leaves the state as it
  is and exits 0, since a failing hook shows up in Claude Code.
- One file for all sessions; keying it by `session_id` is the way to give two
  side-by-side sessions a pane each.
- Not done: tailing the transcript for the phases no hook sees (the model
  writing its answer). `UserPromptSubmit` and `PostToolUse` set thinking,
  which covers most of it.

## Rendering

- **Braille and ASCII first.** Half-block "plasma" and metaballs were tried
  as ways to draw the orb and dropped: they read as generic next to the other
  two. Pixel (quarter blocks), octant, sextant and a kitty-protocol image
  style were built, then left out of v0.1.0 to keep the list short;
  `git revert 842f73b` brings them back.
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

## Scenes

- **Each state's weight, eased.** `p.mood` holds one weight per state, eased
  like every other parameter, so a scene blends its figures while the state
  changes instead of cutting from one to the next.
- **Lines of one width.** Contours and field lines are isolines of a field,
  their distance divided by the field's gradient (`iso`): steep or flat, a
  line keeps the same thickness.
- **Field lines from summed angles.** The lines between two poles are
  isolines of the sum of the angles round each; with the exact gradient, the
  jump where `atan2` wraps never shows.
- **The oscilloscope switches figures.** Two Lissajous figures with different
  frequencies can't be blended, so the dominant state picks one.
- **Dropped:** a black hole, a ray-marched 3D mask, tendrils, one big eye
  and a hooded figure; they looked busy or out of place next to the line
  drawings.

## Intro

- **GIFs as they are, not through a style.** A GIF drawn in braille lost what
  made it worth showing; the intro sends real pictures through the kitty
  graphics protocol, with an optional contrast or digital filter.
- **Frames composited on load.** A GIF frame is often only the patch that
  changed; frames are drawn over the ones before, following each frame's
  disposal, then shrunk to 480 pixels at most so a long GIF stays small.
- **A frame is sent only when it changes.** GIFs mostly run slower than the
  30 fps loop.

## README GIFs

`tools/readme-gif` turns a braille capture into a GIF by drawing the dots
themselves: no font, and every capture of one size gives a GIF of one size.

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
| macOS (Darwin 27), Apple Silicon | Builds, tests pass, every style and statue mode watched in a terminal, ~30 fps at 200×60; the GIF intro checked from decoded captures, not yet watched in Ghostty |
| Linux | `GOOS=linux go vet` only; never run |
| Windows | `GOOS=windows go vet` only; never run. No terminal size (falls back to `COLUMNS`/`LINES`), no keys (no `stty`) |
