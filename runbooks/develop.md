# Working on mori

Go 1.26 or newer. No other tooling required — the Charm libraries come in via
modules, and there's nothing to generate.

```sh
git clone https://github.com/rmpato/mori
cd mori
make check      # gofmt, go vet, go test ./...
make build      # ./mori
```

## Run it without touching your own journal

```sh
MORI_DIR=$(mktemp -d) go run .
```

`MORI_DIR` (the journal directory), `MORI_HOME`, and `XDG_DATA_HOME` are
checked in that order. `mori path` prints where it landed. Do this every time:
the alternative is writing test entries into your actual diary.

`MORI_CONFIG` and `TUKI_FILE` are worth pointing somewhere throwaway too, or
your own config and task list leak into whatever you're testing.

## Layout

| package | what lives there |
| --- | --- |
| `internal/entry` | the domain: a calendar day, a page, tags, sections, the file format |
| `internal/store` | Markdown files on disk, written atomically |
| `internal/search` | queries and tag counts, as a plain scan |
| `internal/facts` | the read-only view of tuki |
| `internal/config` | the few things there are to configure |
| `internal/ui` | palette, styles, the face, the seasons |
| `internal/tui` | the full-screen interface (Bubble Tea) |
| `internal/cli` | the commands (Cobra, wrapped in Fang) |
| `internal/update` | the self-updater |
| `docs/` | the website — see [website.md](website.md) |

`internal/entry` deliberately imports nothing but the standard library, which
keeps the interesting logic cheap to test. Dependencies point downwards only:
if the TUI needs something, it goes in the shared layer or it goes in the TUI,
never sideways.

## Tests

```sh
go test ./...
go test -race ./...
go test ./internal/tui -run TestTheFooterFitsThePage -v
```

Four things worth knowing before you touch the TUI suite:

- **The harness runs commands and feeds their messages back**, the way Bubble
  Tea would, so searching and tag counting are exercised through their real
  asynchronous path. Commands get a 25ms window; the ones that don't answer in
  it are the 750ms and 3s timers, and the tests that care about those send
  `autosaveMsg` by hand.
- **Batched commands run concurrently**, so one sleeping timer alongside a
  search doesn't cost the search its window.
- **Key names come from Bubble Tea v2.** Space is `"space"`, not `" "`.
- **Layout invariants are tested, not eyeballed.** `TestTheFrameAlwaysFits`
  checks every view down to 12×5, and `TestTheFooterFitsThePage` checks the
  help line against the width of the rule above it — a footer that outgrows
  mori's 76-column page loses its tail silently on narrow terminals and runs
  past the rule on wide ones, which is exactly the kind of thing nobody
  notices until it's in a screenshot.

There are also tests that guard the project's promises rather than its code:
that `looking-back` never says "streak" or "average", that the tuki template
contains nothing but headings and your own task text, and that no command
writes to tuki's file. Those are load-bearing. If one fails, the fix is
probably the code.

## Debugging the interface

You can't print to stdout while the TUI owns the screen. Log to a file:

```go
f, _ := tea.LogToFile("/tmp/mori.log", "mori")
defer f.Close()
```

```sh
tail -f /tmp/mori.log
```

For a quick look at a rendered frame without a terminal, write a throwaway
test that builds a harness and prints `h.screen()` — that's how the site's
terminal blocks were checked against the real output.

## Conventions

- `make check` before pushing. CI runs the same thing plus the race detector
  on Linux and macOS.
- Comments explain **why**, not what. If a line needs a comment to say what it
  does, rename something instead.
- Errors read like sentences a person would say: `mori doesn't know when
  "someday" is`, not `parse error`.
- mori never counts anything at the user. No streaks, no scores, no "you
  haven't written in N days". If a feature wants a number, ask whether the
  number is interesting or whether it's a scoreboard.
