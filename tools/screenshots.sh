#!/bin/zsh
#
# Capture every screenshot the site uses, into docs/.
#
#   ./tools/screenshots.sh
#
# Opens a second Ghostty instance against a throwaway journal, drives it with
# keystrokes, captures each screen, and closes only the window it opened.
# Your own journal and your own terminal are never touched — see
# runbooks/screenshot.md for why that second part matters.
#
# Needs: Ghostty, and Accessibility permission for whatever runs this (System
# Events is used to find the window and to type into it).

set -eu

ROOT="${0:A:h:h}"
OUT="$ROOT/docs"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT INT TERM

# mori's own dark palette, so the capture blends into the page rather than
# sitting on it as a darker rectangle.
BG="#111410"
FG="#e7e6dc"

# ------------------------------------------------------------------ setup --

go build -o "$WORK/mori" "$ROOT"
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

printf 'A slow Sunday. Walked out to the lake and back. #photography\n'    > "$WORK/journal/2026/08/2026-08-16.md"
printf 'Rain all afternoon. Read most of the GitOps book instead. #work\n' > "$WORK/journal/2026/08/2026-08-14.md"
printf 'Long call with mum. Started sketching the zine again. #ideas\n'    > "$WORK/journal/2026/08/2026-08-11.md"
printf 'Bariloche photos finally selected. 200 down to 19. #photography\n' > "$WORK/journal/2026/08/2026-08-09.md"
printf 'Gym, then groceries. An ordinary one. #health\n'                   > "$WORK/journal/2026/08/2026-08-05.md"
printf 'First proper day of the new project. #go #work\n'                  > "$WORK/journal/2026/08/2026-08-02.md"
printf 'Packed. Nervous in the good way. #travel\n'                        > "$WORK/journal/2026/07/2026-07-28.md"
chmod 600 "$WORK"/journal/2026/*/*.md

# A tuki task list, so the context pane has something to show.
cat > "$WORK/tasks.json" <<'EOF'
{"version":1,"tasks":[
 {"text":"Finish the storage layer","tag":"go","done":true,"created_at":"2026-08-17T09:00:00Z","done_at":"2026-08-17T18:20:00Z"},
 {"text":"Read the GitOps chapter","tag":"work","done":true,"created_at":"2026-08-17T09:00:00Z","done_at":"2026-08-17T20:10:00Z"},
 {"text":"Go to the gym","tag":"health","done":true,"created_at":"2026-08-17T08:00:00Z","done_at":"2026-08-17T19:00:00Z"},
 {"text":"Edit the Bariloche photos","tag":"photography","done":false,"created_at":"2026-08-17T08:05:00Z"},
 {"text":"Select photos","tag":"photography","done":true,"created_at":"2026-08-01T09:00:00Z","done_at":"2026-08-09T18:00:00Z"},
 {"text":"Book the flights","tag":"travel","done":true,"created_at":"2026-07-20T09:00:00Z","done_at":"2026-07-28T12:00:00Z"}
]}
EOF

# ---------------------------------------------------------------- capture --

# ghostty <rows> <command...> — opens a window and prints its pid.
ghostty() {
  local rows="$1"; shift
  cat > "$WORK/run.sh" <<EOF
#!/bin/zsh
export MORI_DIR="$WORK/journal"
export MORI_CONFIG="$WORK/none.json"
export TUKI_FILE="$WORK/tasks.json"
export MORI_THEME=dark
$@
EOF
  chmod +x "$WORK/run.sh"

  cat > "$WORK/ghostty.conf" <<EOF
command = $WORK/run.sh
window-save-state = never
window-width = 84
window-height = $rows
window-position-x = 120
window-position-y = 120
font-size = 20
window-padding-x = 20
window-padding-y = 18
background = $BG
foreground = $FG
title = mori
EOF

  local before after
  before=$(ps -Ao pid,comm | grep 'MacOS/ghostty' | awk '{print $1}' | tr '\n' '|' | sed 's/|$//')
  open -na Ghostty --args --config-file="$WORK/ghostty.conf"
  sleep 5
  after=$(ps -Ao pid,comm | grep 'MacOS/ghostty' | awk '{print $1}' | grep -vE "^(${before})$" | head -1)
  [[ -n "$after" ]] || { echo "no new Ghostty window appeared" >&2; exit 1 }
  print -r -- "$after"
}

# shot <pid> <name> — capture exactly that window into docs/<name>.png.
shot() {
  local pid="$1" name="$2" bounds x y w h
  bounds=$(osascript -e "tell application \"System Events\" to tell (first process whose unix id is $pid) to get {position, size} of front window")
  x=${${(s:,:)bounds}[1]// /}; y=${${(s:,:)bounds}[2]// /}
  w=${${(s:,:)bounds}[3]// /}; h=${${(s:,:)bounds}[4]// /}

  screencapture -x -o -R "$x,$y,$w,$h" "$WORK/$name.png"
  sips -Z 1600 "$WORK/$name.png" --out "$WORK/$name-scaled.png" >/dev/null
  sips -s format bmp "$WORK/$name-scaled.png" --out "$WORK/$name.bmp" >/dev/null
  python3 "$ROOT/tools/round_corners.py" "$WORK/$name.bmp" "$OUT/$name.png" 14
}

# type <pid> <keys...> — send keystrokes to that window.
send() {
  local pid="$1"; shift
  osascript -e "tell application \"System Events\" to tell (first process whose unix id is $pid) to set frontmost to true" >/dev/null
  sleep 1
  for key in "$@"; do
    # Named keys go by key code. `keystroke "tab"` types the letters t, a, b,
    # which in mori's reader means "jump to today" and two keys that do
    # nothing — a screenshot that looks fine and shows the wrong screen.
    case "$key" in
      esc)   osascript -e 'tell application "System Events" to key code 53' ;;
      tab)   osascript -e 'tell application "System Events" to key code 48' ;;
      enter) osascript -e 'tell application "System Events" to key code 36' ;;
      *)     osascript -e "tell application \"System Events\" to keystroke \"$key\"" ;;
    esac
    sleep 0.6
  done
  sleep 1
}

# The interface, one window driven through four screens.
PID=$(ghostty 18 "exec \"$WORK/mori\"")
shot "$PID" screenshot            # today's page
send "$PID" c;            shot "$PID" screen-calendar
send "$PID" esc / p h o t o; shot "$PID" screen-search
send "$PID" esc '#';      shot "$PID" screen-tags
send "$PID" esc tab;      shot "$PID" screen-context
kill "$PID"; sleep 2

# The command line, which needs its own shorter window.
PID=$(ghostty 12 "\"$WORK/mori\" looking-back august; sleep 3600")
shot "$PID" screen-lookingback
kill "$PID"; sleep 1

ls -lh "$OUT"/*.png | awk '{print $9, $5}'
