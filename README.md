# mori

> Remember your days, not just your tasks.

A private, local-first journal that lives in your terminal. One plain Markdown
file per day. No account, no cloud, no streaks, no scores. You type `mori`,
and today's page is open.

**[rmpato.github.io/mori](https://rmpato.github.io/mori/)**

![mori running in a terminal: Monday, August 17 2026, a mood of "calm", a
page written in two sittings with a 23:04 timestamp between them, and tags
picked out in green](docs/screenshot.png)

It has a sibling. [tuki](https://github.com/rmpato/tuki) keeps the things you
mean to do; mori keeps what happened — same face, eyes closed.

```
tuki → what do I need to do?
mori → what happened?
```

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
| `tab` | today's context, from tuki |
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
mori looking-back august   # a month, read back
mori path                  # where the journal lives
mori path yesterday        # the file for a day, whether or not it exists yet
```

## Looking back

```
🌿  August 2026

    You wrote on 9 days.

    #photography   7 days
    #go            2 days
    #work          1 day

    In tuki, you finished 4 things, 2 of them #photography.
```

Counts of what happened, in the past tense. No comparison to any other month,
no average, no percentage, no target, and no adjective about whether it was
enough. A month with two days in it prints two days and says nothing else
about it.

Bare search words match the start of a word, so `photo` finds `photography`.
A quoted phrase is matched exactly. Every printing command takes `--plain`
(tab-separated) and `--json`.

## tuki

If you also use [tuki](https://github.com/rmpato/tuki), mori can show you what
you got done on a day while you decide what to say about it. Press `tab`, or:

```bash
mori today --from-tuki
```

```markdown
# August 17, 2026

## Today

## Things I did

- Finish Go project
- Read GitOps chapter
- Go to the gym

## Things I didn't get to

- Edit photography photos

## Notes
```

That is a scaffold, and it stops there. **tuki holds what you did; mori holds
what it was like**, and mori will never write the second from the first. The
words are yours or they aren't worth keeping.

The integration is optional, appears only if tuki is installed, and is
**read-only**: mori reads tuki's file and never writes to it. `mori config`
shows where things stand; `{"tuki": {"enabled": false}}` turns it off.

mori works exactly the same with no tuki, and never mentions it.

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

[`runbooks/`](runbooks/) covers releasing, the website, regenerating the
screenshot, and finding your way around the code.

Releases are cut by tagging: `git tag v0.1.0 && git push --tags` runs
GoReleaser from CI. The archive names in `.goreleaser.yaml` are load-bearing —
`install.sh` and `mori update` both construct them, and a test in
`internal/update` fails if the three ever stop agreeing.

## Design

[`docs/DESIGN.md`](docs/DESIGN.md) has the architecture, the data model, the
storage decisions, the interface, and the roadmap.

[`docs/WEB.md`](docs/WEB.md) is the design direction for `mori web`: an
optional local HTTP server so the browser can be a second, read-first
interface onto the same journal. Not built yet — the document exists so the
architecture can be checked against it before it is.

## License

MIT
