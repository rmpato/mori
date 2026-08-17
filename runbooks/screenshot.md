# Regenerate the screenshot

`docs/screenshot.png` is a real capture of mori in a Ghostty window, used by
both the README and the site. This is fiddlier than it looks, so read the
gotchas first — one of them can close the terminal you're standing in.

## Rules

- **Never use your own journal.** Point mori at a throwaway directory with
  `MORI_DIR`. Your own pages stay out of the picture, which for a diary is
  not a small thing.
- **Never `pkill` Ghostty.** tuki's runbook says to quit Ghostty first so the
  new window doesn't open as a tab. Do not copy that here if you are working
  *inside* Ghostty — you will kill your own session, and any agent running in
  it. Open a second instance and close only that one. See the gotchas.
- **Set the terminal background to mori's page background** — `#111410`, with
  `#e7e6dc` in front of it. Ghostty's default is nearly black, which sits on
  the site as a colder rectangle instead of belonging to it. tuki's screenshot
  does the same with its own `#14120f`.
- **84×18 cells.** Wide enough for mori's 76-column page with a margin, short
  enough that the empty half of the page doesn't dominate. The `looking-back`
  shot uses 12 rows, because it's a command and not a screen.

## Just run the script

```sh
./tools/screenshots.sh
```

That builds mori, seeds a throwaway journal and a throwaway tuki task list,
opens a second Ghostty window, drives it through every screen the site shows,
captures each one, and closes only the window it opened. Six files land in
`docs/`. Read the gotchas anyway — they're why the script is shaped the way it
is, and you'll want them if you change what it captures.

The demo entry is chosen to show mori's whole vocabulary in one frame: the
season glyph, a mood, prose that wraps, a tag, and a second sitting with its
timestamp. The neighbouring days exist so the calendar, search and tags have
something to show.

## Or by hand

Build mori against a throwaway journal, then:

```sh
cat > "$WORK/ghostty.conf" <<EOF
command = $WORK/run.sh
window-save-state = never
window-width = 84
window-height = 18
window-position-x = 120
window-position-y = 120
font-size = 20
window-padding-x = 20
window-padding-y = 18
background = #111410
foreground = #e7e6dc
title = mori
EOF

BEFORE=$(ps -Ao pid,comm | grep 'MacOS/ghostty' | awk '{print $1}' | tr '\n' '|' | sed 's/|$//')
open -na Ghostty --args --config-file="$WORK/ghostty.conf"
sleep 6
PID=$(ps -Ao pid,comm | grep 'MacOS/ghostty' | awk '{print $1}' | grep -vE "^(${BEFORE})$" | head -1)
```

Ask macOS where the window is, and capture exactly that:

```sh
osascript -e "tell application \"System Events\" to tell (first process whose unix id is $PID) \
  to get {position, size} of front window"
# 120, 158, 1088, 577

screencapture -x -o -R 120,158,1088,577 "$WORK/win.png"
sips -Z 1600 "$WORK/win.png" --out "$WORK/scaled.png"
sips -s format bmp "$WORK/scaled.png" --out "$WORK/scaled.bmp"
python3 tools/round_corners.py "$WORK/scaled.bmp" docs/screenshot.png 14
kill "$PID"
```

## Gotchas

**Do not quit Ghostty to avoid the tab problem.** If you are running inside
Ghostty — and if you're reading this in a terminal, you probably are —
`pkill ghostty` takes your shell, your editor, and this runbook with it.
`open -na` opens a separate instance, which is enough; close it by PID.

**Capture the window, not the screen.** `screencapture` with no `-R` takes
the whole display, which means your desktop, your other windows, and whatever
was on them ends up in a file you then have to remember to delete. Asking
System Events for the window rect avoids ever having that file. It also
avoids the guesswork of `sips --cropOffset`.

**Ghostty remembers its window size.** A new `font-size` with a remembered
pixel size just means fewer visible cells, and the bottom of the page gets
cut. `window-save-state = never` prevents it.

**Use a config file, not `-e`.** Launching `open -na Ghostty --args … -e
script.sh` raises a macOS "Allow Ghostty to execute…" dialog, which lands on
top of the window you're trying to photograph. `command =` in a config file
doesn't.

**Named keys go by key code, not `keystroke`.** `keystroke "tab"` types the
letters t, a, b — which in mori's reader means "jump to today" and two keys
that do nothing. The result is a screenshot that looks perfectly fine and
shows the wrong screen. Tab is key code 48, escape is 53, return is 36.

**`MORI_THEME=dark` is not optional.** Outside a full terminal handshake mori
guesses the background, and a light guess makes the capture unreadable.

**Round the corners.** The window's own rounded corners let a sliver of
whatever was behind show through at each corner. `tools/round_corners.py`
masks them to transparent; without it you get four bright specks, and GitHub
can't fix it with CSS the way the site can.

## Check

- 1600px wide, a couple of hundred KB at most.
- The background matches the site's `--paper`. Sample a pixel if unsure; it
  should read `#111410`, not near-black.
- Each screen shows what its tab claims. The context one is the easiest to get
  wrong — if it shows today's page, the tab key didn't land.
- The footer reads all the way to `q quit`. If it's cut off, the help line has
  outgrown the 76-column page again — fix `ShortHelp`, not the window size.
  `TestTheFooterFitsThePage` is meant to catch that first.
- The mood, a tag, and the `23:04` timestamp are all visible.
- Corners transparent, no desktop showing.
- Nothing of yours in the frame.
