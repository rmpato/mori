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

## Where it is

mori is early. The domain, the storage layer, and the scripting commands are
done and tested; the full-screen interface is next.

```bash
mori today                 # print today's page
mori show yesterday        # any day: fri, -3d, 2026-08-17, "17 aug"
mori show 2026-08-17 --json
mori path                  # where the journal lives
mori path yesterday        # the file for a day, whether or not it exists yet
```

Until the interface lands, this is enough to write with:

```bash
$EDITOR "$(mori path today)"
```

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
