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
- **84×18 cells.** Wide enough for mori's 76-column page with a margin, short
  enough that the empty half of the page doesn't dominate. Recount if you
  change the demo entry: header 3 + footer 3 + one line per wrapped line of
  prose, plus the blank lines between paragraphs.

## Set up the demo journal

```sh
WORK=$(mktemp -d)
go build -o "$WORK/mori" .
mkdir -p "$WORK/journal/2026/08" "$WORK/journal/2026/07"

cat > "$WORK/journal/2026/08/2026-08-17.md" <<'EOF'
---
mood: calm
---

Slow start. Coffee on the balcony while the street woke up, and for once
I didn't reach for my phone.

Finally started that little Go project I've been circling for weeks. It
feels good to build something small just because I want to. #go

## 23:04

Came back to it after dinner and got the storage layer working. Didn't
touch the #photography edits at all — that's tomorrow.
EOF

printf 'A slow Sunday. Walked out to the lake and back. #photography\n' > "$WORK/journal/2026/08/2026-08-16.md"
printf 'Rain all afternoon. Read most of the GitOps book instead. #work\n' > "$WORK/journal/2026/08/2026-08-14.md"
printf 'Long call with mum. Started sketching the zine again. #ideas\n'   > "$WORK/journal/2026/08/2026-08-11.md"
printf 'Bariloche photos finally selected. 200 down to 19. #photography\n' > "$WORK/journal/2026/08/2026-08-09.md"
printf 'Gym, then groceries. An ordinary one. #health\n'                  > "$WORK/journal/2026/08/2026-08-05.md"
printf 'First proper day of the new project. #go #work\n'                 > "$WORK/journal/2026/08/2026-08-02.md"
printf 'Packed. Nervous in the good way. #travel\n'                       > "$WORK/journal/2026/07/2026-07-28.md"
chmod 600 "$WORK"/journal/2026/*/*.md
```

That entry is chosen to show mori's whole vocabulary in one frame: the season
glyph, a mood, prose that wraps, a tag, and a second sitting with its
timestamp. The neighbouring days exist so the calendar and search have
something to show if you shoot those too.

## Take it

Ghostty needs a launcher script and a config file — see the gotchas for why
neither `-e` nor CLI flags alone will do.

```sh
cat > "$WORK/run.sh" <<EOF
#!/bin/zsh
export MORI_DIR="$WORK/journal"
export MORI_CONFIG="$WORK/none.json"
export TUKI_FILE="$WORK/none.json"
export MORI_THEME=dark
exec "$WORK/mori"
EOF
chmod +x "$WORK/run.sh"

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
```

Note the PIDs before and after, so you know which window is yours to close:

```sh
BEFORE=$(ps -Ao pid,comm | grep 'MacOS/ghostty' | awk '{print $1}')
open -na Ghostty --args --config-file="$WORK/ghostty.conf"
sleep 6
NEW=$(ps -Ao pid,comm | grep 'MacOS/ghostty' | awk '{print $1}' | grep -vx "$BEFORE" | head -1)
echo "opened $NEW"
```

Ask macOS where that window actually is, rather than measuring it by hand:

```sh
osascript -e "tell application \"System Events\" to tell (first process whose unix id is $NEW) \
  to get {position, size} of front window"
# 120, 158, 1088, 577
```

Capture exactly that rectangle. Nothing else on your screen is in the file,
which is the point:

```sh
screencapture -x -o -R 120,158,1088,577 "$WORK/window.png"
```

Then scale to 1600px wide and round the corners:

```sh
sips -Z 1600 "$WORK/window.png" --out "$WORK/scaled.png"
sips -s format bmp "$WORK/scaled.png" --out "$WORK/scaled.bmp"
python3 tools/round_corners.py "$WORK/scaled.bmp" docs/screenshot.png 14
```

Finally, close **only your window** and clear the workspace:

```sh
kill "$NEW"
rm -rf "$WORK"
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

**`MORI_THEME=dark` is not optional.** Outside a full terminal handshake mori
guesses the background, and a light guess makes the capture unreadable.

**Round the corners.** The window's own rounded corners let a sliver of
whatever was behind show through at each corner. `tools/round_corners.py`
masks them to transparent; without it you get four bright specks, and GitHub
can't fix it with CSS the way the site can.

## Check

- 1600px wide, a couple of hundred KB at most.
- The footer reads all the way to `q quit`. If it's cut off, the help line has
  outgrown the 76-column page again — fix `ShortHelp`, not the window size.
  `TestTheFooterFitsThePage` is meant to catch that first.
- The mood, a tag, and the `23:04` timestamp are all visible.
- Corners transparent, no desktop showing.
- Nothing of yours in the frame.
