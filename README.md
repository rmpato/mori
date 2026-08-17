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
| `?` `q` | keys, quit |

Your writing is saved when you pause, when you stop, and when you leave. You
should never have to think about it.

mori also works in pipes and scripts:

```bash
mori today                 # print today's page
mori show yesterday        # any day: fri, -3d, 2026-08-17, "17 aug"
mori show 2026-08-17 --json
mori path                  # where the journal lives
mori path yesterday        # the file for a day, whether or not it exists yet

$EDITOR "$(mori path today)"
```

Search, a calendar, tags, and the optional [tuki](https://github.com/rmpato/tuki)
bridge are on the way — see the design doc below.

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

## Design

[`docs/DESIGN.md`](docs/DESIGN.md) has the architecture, the data model, the
storage decisions, the planned interface, and the roadmap.

## License

MIT
