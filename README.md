# mori

[![ci](https://github.com/rmpato/mori/actions/workflows/ci.yml/badge.svg)](https://github.com/rmpato/mori/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/rmpato/mori?color=3c6b49&label=release)](https://github.com/rmpato/mori/releases)
[![go report](https://goreportcard.com/badge/github.com/rmpato/mori)](https://goreportcard.com/report/github.com/rmpato/mori)
[![license](https://img.shields.io/badge/license-MIT-3c6b49)](LICENSE)

> Remember your days, not just your tasks.

A private, local-first journal that lives in your terminal. One plain Markdown
file per day, in a folder you own. No account, no cloud, no streaks, no scores.

**[rmpato.github.io/mori](https://rmpato.github.io/mori/)**

![mori running in a terminal: Monday, August 17 2026, a mood of "calm", a page
written in two sittings with a 23:04 timestamp between them, and tags picked
out in green](docs/screenshot.png)

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/rmpato/mori/main/install.sh | sh
```

Checks the download against the release's sha256 and installs to
`~/.local/bin`. Also `go install github.com/rmpato/mori@latest`, or a binary
from [releases](https://github.com/rmpato/mori/releases).

## Start

```sh
mori          # today's page, cursor already in it
```

There is no "new entry" step — every day already exists, most are just empty.
Your writing is saved when you pause, when you stop, and when you leave.

| key | |
|---|---|
| `←` `→` | previous / next day |
| `↵` | write — `esc` when you're done |
| `n` | a new sitting, stamped with the time |
| `c` `/` `#` | calendar · search · tags |
| `m` `E` | mood · open in `$EDITOR` |
| `?` `q` | every key · quit |

It works in pipes too:

```sh
mori show yesterday      # or: fri, -3d, 2026-08-17, "17 aug"
mori search photography  # newest day first
mori looking-back august # a month, read back
mori path                # where the journal lives
```

Every printing command takes `--plain` and `--json`.

## Your journal

```
~/.local/share/mori/journal/2026/08/2026-08-17.md
```

Plain Markdown, `0600`, written atomically. Tags are the hashtags in your
prose; `## HH:MM` marks a second sitting. `MORI_DIR` moves it elsewhere.

## tuki

[tuki](https://github.com/rmpato/tuki) is mori's sibling — it keeps the things
you mean to do. With both installed, `tab` shows what you got done on a day
while you write about it: read-only, optional, and it will never turn your task
list into prose and call it your writing.

> tuki helps you move forward. mori helps you look back.

## More

[design notes](docs/DESIGN.md) ·
[the web interface, planned](docs/WEB.md) ·
[runbooks](runbooks/) ·
[MIT](LICENSE)

## Contributing

Bug reports and ideas are welcome — [CONTRIBUTING.md](CONTRIBUTING.md) says how
things are built and which ideas get turned down, so a no is never a surprise.
Vulnerabilities go through [private reporting](https://github.com/rmpato/mori/security/advisories/new),
not public issues: [SECURITY.md](SECURITY.md).
