# mori

> Remember your days, not just your tasks.

mori is a private, local-first journal that lives in your terminal. It is the
quiet companion to [tuki](https://github.com/rmpato/tuki):

```
tuki → what do I need to do?
mori → what happened?
```

Your pages are plain Markdown files on your own machine, one per day. No
account, no cloud, no network, nothing sent anywhere.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/rmpato/mori/main/install.sh | sh
```

Or with Go:

```sh
go install github.com/rmpato/mori@latest
```

The installer checks the download against the release's own sha256 checksum
before it installs anything, and writes nothing outside the install directory
and one line in your shell config. Later, `mori update` does the same thing
for the binary you already have — it is the only command that touches the
network, and your journal is never part of it.

## Using it

```bash
mori
```

opens today's page, ready to write in. There is no "new entry" step — every
day already exists, most of them are simply empty.

```
🌿  Monday, August 17, 2026                                    today
────────────────────────────────────────────────────────────────────

  Today was actually pretty good.

  I finally started working on that little Go project I've been
  thinking about. It feels nice to build something small just
  because I want to. #go

  23:04

  Went back to it after dinner and got the storage layer working.

────────────────────────────────────────────────────────────────────
←/h previous day • →/l next day • ↵ write • t today • ? keys • q quit
```

Horizontal moves through time, vertical moves through the page.

| key | |
|---|---|
| `←` `h` / `→` `l` | previous / next day |
| `↑` `k` / `↓` `j` | scroll |
| `↵` `i` | write |
| `esc` | done writing |
| `n` | new section |
| `t` | today |
| `g` | go to a date |
| `c` | calendar |
| `/` | search |
| `#` | tags |
| `m` | mood |
| `E` | open in `$EDITOR` |
| `?` `q` | keys, quit |

The calendar shows the month with the days you wrote on picked out. Search
runs as you type, newest day first. Tags are just the hashtags in your
writing — choosing one is choosing a search.

Your writing is saved when you pause, when you stop, and when you leave. You
should never have to think about it.

mori also works in pipes and scripts:

```bash
mori today                 # print today's page
mori show yesterday        # any day: fri, -3d, 2026-08-17, "17 aug"
mori show 2026-08-17 --json
mori list --since 2026-01-01
mori search photography
mori search "the zine idea"
mori search gym since:2026-01-01
mori tags                  # the tags you've used, and how often
mori path                  # where the journal lives
mori path yesterday        # the file for a day, whether or not it exists yet
```

Bare search words match the start of a word, so `photo` finds `photography`.
A quoted phrase is matched exactly. Every printing command takes `--plain`
(tab-separated) and `--json`.

The optional [tuki](https://github.com/rmpato/tuki) bridge is next — see the
design doc below.

## Where your journal lives

```
~/.local/share/mori/journal/2026/08/2026-08-17.md
```

Override with `MORI_DIR`, `MORI_HOME`, or `XDG_DATA_HOME`. `MORI_THEME` forces
`light` or `dark`; `MORI_HEMISPHERE=south` flips the little seasonal glyph
beside the date.

A page is just Markdown, with no ceremony unless you want some:

```markdown
Today was actually pretty good.

I finally started that little Go project. #go

## 23:04

Went back to it after dinner.
```

Tags are hashtags in the prose. `## HH:MM` marks a sitting, if you came back
to the day later. Anything mori doesn't recognise, it leaves exactly as you
wrote it.

## Building

```bash
make build
make check   # fmt, vet, test
```

Releases are cut by tagging: `git tag v0.1.0 && git push --tags` runs
GoReleaser from CI. The archive names in `.goreleaser.yaml` are load-bearing —
`install.sh` and `mori update` both construct them, and a test in
`internal/update` fails if the three ever stop agreeing.

## Design

[`docs/DESIGN.md`](docs/DESIGN.md) has the architecture, the data model, the
storage decisions, the planned interface, and the roadmap.

## License

MIT
