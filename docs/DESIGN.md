# mori — design

> Remember your days, not just your tasks.

mori is a private, local-first journal that lives in the terminal. It is the
quiet half of a two-app ecosystem: **tuki asks what you need to do, mori asks
what happened.**

This document is the architecture, data model, storage, package layout, TUI
structure, and MVP scope. It was written before any code was, and is kept
true as the code catches up — §12 tracks how far along that is.

---

## 1. Principles

These are the constraints that decide arguments later.

1. **The file is the truth.** A mori entry is a plain Markdown file you could
   have written in any editor. mori is a nice way to write and find them, not
   the thing that owns them. If mori disappears, your journal is still there.
2. **No ceremony.** Opening mori puts a cursor in today's page. No "create
   entry" step, no title prompt, no required fields. Every day already exists;
   most of them are simply empty.
3. **Never measure the user.** No streaks, no scores, no word-count goals, no
   "you haven't written in 11 days". Numbers appear only when they are being
   interesting, never when they are being a scoreboard.
4. **Never lose writing.** Autosave, atomic writes, no destructive action
   without a confirmation, and nothing that can leave a half-written file.
5. **Minimal dependencies.** The same set tuki already uses, and nothing else
   unless it earns its place. No YAML library, no SQLite in v1.
6. **Independently useful.** mori must be complete and delightful with tuki
   uninstalled. The bridge is a feature, not a foundation.
7. **Local and offline. Always.** No network calls, no telemetry, no accounts,
   no sync in v1.

---

## 2. Architecture

```
                    ┌──────────────────────────────┐
    mori ──────────▶│            cli               │  cobra + fang
                    │  today, show, search, tags…  │  works in pipes
                    └───────────┬──────────────────┘
                                │
                    ┌───────────▼──────────────────┐
                    │            tui               │  bubbletea
                    │  read · write · calendar ·   │
                    │  search · help               │
                    └───────────┬──────────────────┘
                                │
      ┌─────────────┬───────────┼───────────┬──────────────┐
      │             │           │           │              │
 ┌────▼────┐  ┌─────▼────┐ ┌────▼────┐ ┌────▼────┐   ┌─────▼─────┐
 │  entry  │  │  store   │ │ search  │ │  facts  │   │    ui     │
 │ domain  │  │ markdown │ │  scan   │ │  tuki   │   │  theme    │
 │ no deps │  │  files   │ │         │ │read-only│   │  voice    │
 └─────────┘  └────┬─────┘ └─────────┘ └────┬────┘   └───────────┘
                   │                        │
                   ▼                        ▼
      ~/.local/share/mori/journal    ~/.local/share/tuki/tasks.json
              (mori writes)                (mori only reads)
```

Dependency direction is strictly downward. `entry` imports nothing but the
standard library. `store`, `search`, and `facts` import `entry`. `cli` and
`tui` sit on top. `ui` is a leaf that only knows about colours and words.

---

## 3. Data model

### `entry.Date` — a calendar day, not an instant

A journal is indexed by the day you lived, not by a timestamp. Using
`time.Time` for that invites timezone bugs (an entry written at 23:40 in
Buenos Aires must not become tomorrow's entry on a flight). So:

```go
// Date is a calendar day with no time and no location.
type Date struct{ Year int; Month time.Month; Day int }

func Today() Date
func ParseDate(s string) (Date, error)   // "2026-08-17", "today", "yesterday", "-3d", "aug 17"
func (d Date) String() string            // "2026-08-17"
func (d Date) Human() string             // "August 17, 2026"
func (d Date) Short() string             // "17 Aug"
func (d Date) Weekday() time.Weekday
func (d Date) Add(days int) Date
func (d Date) Before(o Date) bool
```

`ParseDate` should reuse tuki's vocabulary (`today`, `yesterday`, `fri`,
`-3d`, `2026-08-17`, `08-17`) so the two tools speak the same date language.
Note the sign flip: tuki looks forward (`+3d`), mori looks back (`-3d`).

### `entry.Entry` — one day's page

```go
type Entry struct {
    Date Date
    Mood string // optional, one short word; "" most of the time
    Body string // Markdown, exactly as the user typed it
}

func (e Entry) IsEmpty() bool            // no body after trimming
func (e Entry) Tags() []string           // #tags parsed out of the body
func (e Entry) Words() int
func (e Entry) Excerpt(width int) string // first meaningful line, for lists
func (e Entry) Sections() []Section      // the day's writing sessions
```

That is the entire model. No IDs, no created/updated timestamps (the file has
those), no title, no attachments. The date *is* the identity.

### Sessions within a day

A day is one page, but you don't always write it in one sitting. So a day's
body may contain **timestamped sections** — and they are a *reading* of the
body, never a second kind of object:

```go
// Section is one writing session within a day.
type Section struct {
    At   string // "23:04"; empty for the day's opening block
    Body string
}
```

`Sections()` parses the body for lines matching exactly `## HH:MM` and splits
there. `Body` stays the single source of truth; sections are derived on
demand, the way tags are. Nothing in `store` knows they exist, and nothing in
the model needs entry IDs or ordering.

```markdown
Slow morning. Coffee on the balcony, and I finally started that little
Go project I've been thinking about. #go

## 23:04

Went back to it after dinner and got the storage layer working. Didn't
touch the #photography edits at all.
```

Three rules keep this from becoming ceremony:

- **The first block gets no heading.** A day written in one sitting is a plain
  page with no timestamps in it at all — the common case stays clean. mori
  never retroactively stamps a block it didn't watch you start.
- **A heading appears only on a genuine return.** Entering writing mode on a
  day that already has content, more than ~2h after the file was last touched,
  inserts `## HH:MM` and puts the cursor beneath it. `n` does the same on
  demand, whenever you want the break. (The gap check reads the file's mtime,
  which is only ever used to decide a cosmetic heading — if a `git checkout`
  makes it lie, the cost is one line you delete.)
- **The match is strict.** Only `## HH:MM` alone on a line is a section, so the
  `## Things I did` headings from `--from-tuki` — and any other Markdown you
  write — stay ordinary prose.

Because the format is plain Markdown, a day edited in vim, Obsidian, or by
hand round-trips perfectly, and removing a heading you didn't want is just
deleting a line.

**Tags live in the prose.** `#photography` typed anywhere in the body is a
tag, parsed at read time with the same normalisation rules as tuki
(`NormalizeTag`: lowercase, strip `#`, letters/digits/`-`/`_`). Nothing is
stored twice, nothing has to be kept in sync, and you tag a day by writing the
way you already write. Unlike tuki, an untagged entry gets no `misc` fallback —
tags here are genuinely optional.

---

## 4. Storage

### Why Markdown files

| | Markdown files | SQLite | one JSON file |
|---|---|---|---|
| readable in 20 years | ✅ | ⚠️ needs a tool | ⚠️ awkward prose |
| editable outside mori | ✅ | ❌ | ❌ |
| `git`-friendly, `rg`-friendly | ✅ | ❌ | ⚠️ |
| survives corruption | ✅ one bad day | ❌ whole journal | ❌ whole journal |
| search at scale | good enough | ✅ | ⚠️ |
| dependency cost | zero | cgo or pure-Go driver | zero |

Prose belongs in files. The one thing SQLite would win — search — is a problem
mori does not have yet: ten years of daily journalling is ~3,650 files and a
few megabytes, which a linear scan reads in tens of milliseconds. If that ever
stops being true, an index goes *behind* the `Store` interface without the
file layout changing.

### Layout

```
~/.local/share/mori/journal/
  2026/
    08/
      2026-08-16.md
      2026-08-17.md
```

Year/month directories keep any single directory small and make "give me
August" a single `readdir` — which is exactly what the calendar view needs.
The filename is redundant with the path on purpose: the files stay meaningful
if you ever drag them somewhere else.

Path resolution mirrors tuki's, in order:
`$MORI_DIR` → `$MORI_HOME/journal` → `$XDG_DATA_HOME/mori/journal` →
`~/.local/share/mori/journal`. `mori path` prints the answer.

### File format

A mori entry is a Markdown file. Frontmatter is written **only when there is
something to put in it**, so the common case is a file with no ceremony at all:

```markdown
Today was actually pretty good.

I finally started working on that little Go project I've been thinking
about. It feels nice to build something small just because I want to. #go

I didn't get around to editing the #photography photos.
```

With a mood set:

```markdown
---
mood: calm
---

Today was actually pretty good.
```

The date is not in the frontmatter — it is the filename, and duplicating it
invites disagreement. The frontmatter parser handles a deliberately tiny
subset (`key: value`, one per line) so mori needs no YAML dependency; unknown
keys are preserved verbatim on write so a future mori — or your own edits —
never lose data.

### The `Store` interface

```go
type Store interface {
    Get(d entry.Date) (entry.Entry, error)  // missing day → empty entry, not an error
    Put(e entry.Entry) error                // empty body → removes the file
    Has(d entry.Date) (bool, error)
    Dates(from, to entry.Date) ([]entry.Date, error) // days that actually have entries
    Walk(from, to entry.Date, fn func(entry.Entry) error) error // newest first
}
```

Two decisions worth naming:

- **A missing day is not an error.** Opening a day you never wrote gives you a
  blank page, which is the correct answer to "what did I write on May 3rd".
- **Emptying an entry deletes the file.** Otherwise the calendar slowly fills
  with dots for days you opened and didn't write. Deleting is what the user
  meant.

Writes are atomic (temp file + `fsync` + rename), copied straight from tuki's
`store.Save`. `0600` on files and `0700` on the journal directory — this is a
diary, and the default umask is not good enough for it.

---

## 5. Package structure

```
mori/
  main.go                  # ~15 lines, like tuki's
  internal/
    entry/                 # Date, Entry, tag parsing, frontmatter — stdlib only
    store/                 # Store interface + Markdown-file implementation
    search/                # query parsing and matching over a Store
    facts/                 # read-only view of "what else happened today"
    ui/                    # theme, mascot, the handful of things mori says
    cli/                   # cobra commands, the pipe-friendly half
    tui/                   # bubbletea: model.go, update.go, view.go, keys.go
  docs/                    # this file, and later the little site
  Makefile  .goreleaser.yaml  install.sh  .github/workflows/
```

Six internal packages, the same shape as tuki (`task`→`entry`, plus `search`
and `facts`). Two notes:

- **`facts` is named after your own line** — *tuki provides the facts, mori
  provides the reflection*. It exposes a `Source` interface and a tuki
  implementation, so a future git-log or calendar source slots in without the
  TUI learning anything new.
- **No `internal/config` in v1.** Env vars and flags cover everything until the
  tuki bridge lands, at which point a small `config.json` appears (JSON, not
  TOML, to avoid a dependency).

---

## 6. CLI surface

`mori` with no arguments opens the TUI on today. Piped or redirected, it falls
back to printing today's entry — same rule as `tuki`.

**v1**

```
mori                        open the TUI on today
mori today                  print today's entry
mori show <date>            print a day  (2026-08-17 | yesterday | fri | -3d)
mori write [<date>]         open the TUI directly in writing mode
mori edit [<date>]          open the day in $EDITOR
mori list [--since <date>]  recent days, one line each
mori search <query>         search across the journal
mori tags                   tags you've used, with counts
mori path                   where the journal lives
```

**later**

```
mori today --from-tuki      start today's page from a tuki-derived template
mori --tag photography      filter by tag
mori looking-back <month>   the gentle monthly recap
```

Every printing command supports `--plain` (tab-separated, greppable) and
`--json`, matching tuki's scripting contract.

---

## 7. TUI

### Shape

One column, centred, capped at ~76 columns even on a huge terminal — same as
tuki. The whitespace is the point, and a journal is a reading medium.

```
  🌿  mori                                    Monday, August 17, 2026

  ─────────────────────────────────────────────────────────────────

  Today was actually pretty good.

  I finally started working on that little Go project I've been
  thinking about. It feels nice to build something small just
  because I want to.

  #go #photography

  ─────────────────────────────────────────────────────────────────
  ← →  day    ↵ write    c calendar    / search    ? keys    q quit
```

Modes rather than panes: `reading` (default), `writing`, `calendar`,
`search`, `goto`, `help`. Only one is on screen at a time, which is what keeps
it calm.

### Keys

The organising idea: **horizontal moves through time, vertical moves through
the page.** That is why `h/l` are days rather than the vim default of
characters — in a one-column reading app, "left and right" has nothing else to
mean.

**reading**

| key | does |
|---|---|
| `←` `h` / `→` `l` | previous / next day |
| `↑` `k` / `↓` `j` | scroll the entry |
| `↵` `i` | start writing (continues the page) |
| `n` | start a new timestamped section |
| `t` | jump to today |
| `g` | go to a date |
| `c` | calendar |
| `/` | search |
| `#` | tags |
| `m` | set mood |
| `E` | open in `$EDITOR` |
| `tab` | today's context (tuki) — later |
| `?` `q` | keys, quit |

**writing** — a plain textarea, and the default way to write in mori. `esc`
returns to reading and saves, `ctrl+s` saves without leaving. No modal editing
inside mori; that is what `E` is for. Pressing `↵` on a day you already wrote
in puts the cursor at the end of the page, adding a `## HH:MM` section heading
first if you've been away more than a couple of hours — and dropping that
heading again if you leave without writing under it, because a sitting that
didn't happen shouldn't leave a mark.

While you are writing, the keyboard belongs to the writing: `h`, `q`, `n` and
the rest are letters, not commands. `esc` and `ctrl+c` are the only two keys
mori keeps for itself, and both of them save.

**calendar** — `h/l` days, `j/k` weeks, `[`/`]` months, `↵` opens the day,
`esc` back. Days with entries carry a `●`; today gets a ring; the current
month's empty future is dimmed rather than hidden.

**search** — type to filter, results are date + matching excerpt, `↵` opens
that day at the match, `esc` back.

### Writing safety

Autosave is not optional in a journal. The textarea's content is flushed on a
750 ms debounce, on every mode change, and on quit. Because writes are atomic
and a day is one small file, the worst case is losing the last sentence, and
only to a hard kill.

### Testability

Model, update, and view stay separable exactly like tuki's `tui_test.go`: the
model is constructed against a `Store` over a temp directory, keys are fed as
`tea.KeyMsg`, and the rendered string is asserted on. No golden screenshots.

---

## 8. Search

v1 is a streaming scan, newest day first, over `Store.Walk`:

- bare words → case-insensitive, all terms must appear
- `"quoted phrase"` → literal substring
- `#tag` → tag filter
- `since:2026-01-01` / `until:…` → date bounds, applied before reading files

Results stream, so the TUI shows the first matches immediately and a 10-year
journal never feels slow. If it ever does, the fix is an index in `store` —
a `search-index.json` or a SQLite sidecar rebuilt from the files, which stay
the source of truth. That is the whole reason `Store` is an interface.

---

## 9. Mood

One optional short word, set with `m` from a small suggested list (plus
free-form). It renders as a quiet coloured dot next to the date and nothing
else. It is filterable in search. There is no chart, no average, no trend, and
mori never comments on it. If that ever feels tempting, the answer is no.

---

## 10. The tuki bridge

**mori reads tuki's JSON file directly. mori does not import the tuki module.**

Importing it would lock the two projects' versions together and make mori's
build depend on tuki's refactors — exactly the coupling the vision warns
against. Instead `facts` declares the handful of fields it needs, ignores the
rest, and version-gates on tuki's `"version"` field. tuki's format is a
documented, stable, atomically-written JSON file; that is a fine contract.

```go
package facts

// Snapshot is what else is known about a day, from outside mori.
type Snapshot struct {
    Source string // "tuki"
    Done   []Item
    Todo   []Item
}

type Item struct{ Text, Tag string }

// Source answers "what else happened on this day?"
type Source interface {
    Day(d entry.Date) (Snapshot, error)
}

func Tuki(path string) Source // read-only; a missing tuki file is not an error
```

"Done on day D" means `done_at` falls on D. "Todo on day D" means open and
either due that day or created on or before it — deliberately generous, since
this is context for a human, not a report.

Rules, in order of importance:

1. **Read-only.** mori never writes to tuki's file. Not in v1, not by default
   ever. A future `mori` that writes tasks is a different conversation.
2. **Optional and quiet.** No tuki file → the context pane simply doesn't
   exist. No warning, no setup prompt, no nagging.
3. **Never generates prose.** `--from-tuki` produces headings and bullets of
   *facts*, and empty space underneath. mori will not write a sentence and
   attribute it to you.

```markdown
# August 17, 2026

## Today

## Things I did
- Finish Go project
- Read GitOps chapter

## Things I didn't get to
- Edit photography photos

## Notes
```

The `tab` context pane shows the same information beside the blank page while
you write. You still do the writing.

---

## 11. Voice and visual identity

tuki is amber and mischievous; mori is green and patient. Same bones — the
`ui.Theme` struct, `LightDark`, the light/dark detection, the narrow column —
different temperature.

```go
Brand   #3E7A52 / #86C79A   // moss
Text    #1F2328 / #DCDEE0   // same as tuki
Muted   #6B7075 / #9198A0
Faint   #B5BAC0 / #5A6067
Accent  #8A6A00 / #DCC96B   // autumn, used sparingly
```

Seasonal glyphs are the one whimsical touch: 🌱 spring, 🌿 summer, 🍂 autumn,
❄ winter — chosen from the entry's own date, so scrolling back through the
year quietly changes weather. Season-appropriate, never explained.

A mascot: something small, abstract, and sitting still — a sprout, a small
round creature with its eyes closed. Whatever it is, it does not react to what
you write, and it never appears at the top of an empty page looking expectant.

**Things mori says**, in full:

- opening an empty day → a blank page. No prompt.
- returning after a long gap → `welcome back.` Once. Never with a number.
- on quit → the date you were on, and nothing else.
- there is no line for "you haven't written lately", because that line is the
  reason people abandon journals.

---

## 12. MVP scope

**In**

1. `mori` opens the TUI on today, ready to write
2. Markdown files on disk, atomic writes, `0600`
3. Reading and writing any day; `←`/`→` day navigation; timestamped sections
   for days written in more than one sitting
4. Calendar view showing which days have entries
5. Search across the journal
6. Tags parsed from prose, `mori tags`, tag filter in search
7. Optional mood
8. `$EDITOR` escape hatch
9. Pipe-friendly CLI: `today`, `show`, `list`, `search`, `tags`, `path`
10. The green theme, the seasonal glyph, `?` help generated from the keymap

**Out (deliberately)**

- The tuki bridge — interfaces defined in v1, wired up in the milestone after
- Attachments, images, links-as-first-class
- Encryption, sync, export, config file
- `looking-back` monthly reflection
- Any search index

**Milestones**

| | | |
|---|---|---|
| **M0** | `entry` + `store` + tests; `mori path`, `mori today`, `mori show` | done |
| **M1** | TUI: read + write + day navigation + autosave + theme | done |
| **M2** | calendar, search, tags, mood, `$EDITOR` | done |
| **M3** | release plumbing — Makefile, goreleaser, install.sh, CI, README | |
| **M4** | `facts` + tuki bridge + `--from-tuki` + config | |
| **M5** | polish: mascot, seasons, `looking-back` | |

M0–M3 is a real, finished, releasable journal that has never heard of tuki.
That is the point.

---

## 13. Decisions taken

1. **Writing happens inside mori.** A built-in textarea is the default surface,
   so `mori` is a place you sit down in rather than a launcher. `E` opens
   `$EDITOR` for anything heavier.
2. **One page per day, with timestamped sections inside it.** The date stays
   the identity — no entry IDs, no ordering, no per-entry navigation — while a
   day written across a morning and a night still shows that shape. Sections
   are parsed out of the Markdown, not stored alongside it. See §3.
3. **Full tuki-compatible date vocabulary in v1.** `today`, `yesterday`, `fri`,
   `-3d`, `2026-08-17`, `08-17`, ported from tuki's `ParseDue` with the sign
   flipped to look backwards. The two tools should speak one date language.

Still open, but not blocking: what the mascot actually looks like, and whether
`looking-back` is a command or a view.
