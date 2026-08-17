# mori web — design direction

> The terminal is where you write. The browser is where you remember.

A second interface to the same journal: a local HTTP server you start with
`mori web` and read in your browser. It is **not** implemented yet. This
document is the shape it should take, written before the code so the
architecture can be checked against it.

The one-line summary of the whole thing:

> `mori` is for writing. `mori web` is for looking back. Neither owns the data.

---

## 1. What it is, and what it isn't

**It is** a reading room. Calendar, timeline, search, tags, on-this-day,
Markdown rendered properly, long entries comfortable to read on a screen that
isn't 80 columns wide.

**It isn't** a CRUD dashboard for journal entries. No tables of records, no
row of action buttons, no "entries (247)" counter at the top. If it ends up
looking like an admin panel, it has failed, however well it works.

The TUI can be utilitarian, because you are writing. The browser can be
beautiful, because you are remembering.

---

## 2. Architecture

The web interface must not become a second application. It is a presentation
layer, exactly like the TUI is:

```
                        ┌─────────────┐
                        │    store    │   Markdown files, one per day
                        └──────┬──────┘
                               │
                 ┌─────────────┼─────────────┐
                 │             │             │
            ┌────▼────┐   ┌────▼────┐   ┌────▼────┐
            │  entry  │   │ search  │   │  facts  │
            └────┬────┘   └────┬────┘   └────┬────┘
                 │             │             │
        ┌────────┴─────────────┴─────────────┴────────┐
        │                                             │
   ┌────▼────┐                                   ┌────▼────┐
   │   tui   │  bubbletea                        │   web   │  net/http
   └─────────┘                                   └────┬────┘
                                                      │
                                                  a browser
```

`internal/web` sits beside `internal/tui`, at the same level, importing the
same four packages. It gets a `*store.Store` and a `facts.Source` handed to it
by `internal/cli`, the same way the TUI does today.

**Nothing in `entry`, `store`, `search`, or `facts` may change shape to suit
HTTP.** If the web interface needs something the TUI didn't, it belongs in the
shared layer as a capability, not as a web-specific model. There is one
journal; there are two ways to look at it.

The concrete test of this: adding the web interface should not require editing
a single line in `internal/entry` or `internal/store`.

---

## 3. Privacy and security

This is the part worth getting right before writing any handlers, because a
local server holding a diary is a genuinely sensitive thing and several of the
mistakes are silent.

### Bind to the loopback address, and say so

```
$ mori web

  (-.-)  mori web

  Listening on  http://127.0.0.1:7432
  Local access only. Nothing else on your network can reach this.

  Ctrl+C to stop.
```

Bind to `127.0.0.1` literally — **not** `localhost`, which resolves through
the name service and may give `::1`, both, or something a hosts file decided.
The default must be an address, not a name.

### Reject requests that arrive under the wrong name

A localhost server with no authentication is reachable from any web page you
happen to visit, via **DNS rebinding**: a hostile page resolves its own domain
to `127.0.0.1` and then reads your journal with your browser's own
credentials. The same-origin policy does not help, because to the browser it
is the attacker's own origin.

The fix is one middleware, and it must be there from the first commit:

- Reject any request whose `Host` header is not the address mori is actually
  serving (`127.0.0.1:7432`, `localhost:7432`, `[::1]:7432`).
- Reject any request carrying an `Origin` that is not one of those.
- Send no CORS headers at all — ever.

### The rest of the headers

- `Cache-Control: no-store` on every response. Journal text should not linger
  in a shared browser cache.
- `Referrer-Policy: no-referrer`, so no address you write about leaks through
  a click.
- `Content-Security-Policy: default-src 'self'; img-src 'self' data:` — the
  page has no business fetching anything, and this makes that true rather than
  merely intended.
- `X-Content-Type-Options: nosniff`.

### Assets are embedded

CSS and JS ship inside the binary via `embed.FS`. No CDN, no webfonts, no
analytics. A page displaying a private journal must make zero network requests,
and the only way to be sure of that is to have nothing to request.

### LAN access is opt-in and loud

```
$ mori web --host 0.0.0.0

  (-.-)  mori web

  Listening on  http://0.0.0.0:7432
  ⚠ Anyone on your network can read your journal at this address.
    There is no password. Ctrl+C to stop.
```

Never the default. When a non-loopback host is given, the Host-header
allowlist has to widen, which is precisely when authentication starts to
matter — so **if LAN access is ever more than a warning, it grows a token
first**, printed once at startup and required as a query parameter or cookie.
That is a later decision; the note is here so it isn't forgotten.

### Read-only to start

The first version serves `GET` and nothing else. No editing, no deleting, no
`POST`. That removes CSRF from the threat model entirely for v1, and when
editing does arrive it needs: `POST` only, an `Origin` check, and a per-session
token. Writing goes through `store.Put` like everything else, atomically.

---

## 4. Scope

**v1 — reading**

| | |
|---|---|
| `/` | today, or the most recent day with writing |
| `/day/2026-08-17` | one day, rendered |
| `/month/2026-08` | the calendar, with the written days marked |
| `/year/2026` | twelve months at a glance |
| `/search?q=…` | the same query language as the CLI |
| `/tags` and `/tag/photography` | the way in by subject |
| `/on-this-day` | the same date, in every year you have |

Previous and next day navigation on every day page, because that is how you
actually read a journal.

**Later**

- Editing in the browser
- Attachments and images
- The tuki context panel on a day page — visually distinct from the writing,
  clearly not part of it
- A timeline that puts journal and task history side by side

**Never**

- Productivity analytics. `87% productive this month` is the exact thing this
  project exists in opposition to. The `looking-back` rules apply here in full:
  counts of what happened, past tense, no comparison, no adjective.

---

## 5. Markdown

The one place a dependency is genuinely needed. Hand-rolling a Markdown
renderer is a trap, and hand-rolling a *safe* one is a worse trap.

Use [goldmark](https://github.com/yuin/goldmark): pure Go, no cgo,
CommonMark-compliant, and the same engine Hugo uses. Render with raw HTML
**disabled**, so a stray `<script>` in a page you wrote years ago on a
different machine cannot execute. Everything else goes through
`html/template`, which escapes by default — never `text/template`.

`## HH:MM` section headings should render as quiet timestamps rather than as
headings, matching how the TUI shows them. Tags become links to `/tag/x`.

---

## 6. Design

Paper, not chrome. A serif column, wide margins, generous line height, and the
same seasonal leaf the terminal uses. Light and dark, following the system.

The project page in `docs/` is the reference for the visual language: that is
what mori looks like on a screen with proportional type available. The web
interface should feel like the same publication.

---

## 7. CLI

```
mori web                      127.0.0.1:7432
mori web --port 8080
mori web --host 0.0.0.0       explicit, warned about
mori web --open               open a browser too
mori web --no-tuki            don't show task context
```

Port 7432 by default — high, unregistered, and stable so a bookmark keeps
working. If it's taken, say so and suggest `--port` rather than silently
picking another one; a journal that moves address is a journal you can't
bookmark.

---

## 8. What this must not do to the rest of mori

- No new dependencies in `entry`, `store`, `search`, or `facts`.
- No change to the file format. A journal written before `mori web` existed
  must render identically to one written after.
- `mori` with no arguments must still start instantly. The web server is a
  subcommand and its dependencies must not be reachable from the TUI's start-up
  path in any way that costs milliseconds.
- The TUI remains complete on its own. Somebody who never runs `mori web`
  should never notice it exists.
