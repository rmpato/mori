# Contributing

Thanks for looking. Mori is small on purpose, and the fastest way
to get a change merged is to keep it that way.

## Getting set up

```sh
git clone https://github.com/rmpato/mori
cd mori
make check      # gofmt, go vet, go test ./...
make build      # ./mori
```

Go 1.26 or newer. Nothing else to install.

**Run it against a throwaway data directory**, never your own:

```sh
MORI_DIR=$(mktemp -d) go run .
```

[`runbooks/`](runbooks/) covers releasing, the website, regenerating the
screenshots, and finding your way around the code.

## Before you open a pull request

- `make check` passes. CI runs the same thing plus the race detector on Linux,
  macOS and Windows.
- New behaviour has a test. Layout and interface changes included — the TUI
  suite drives the real update/view loop headlessly, so a regression can fail
  the build instead of showing up in someone's terminal.
- Comments explain **why**, not what. If a line needs a comment to say what it
  does, rename something instead.
- Errors read like something a person would say.

## Things that will get a change turned down

- **Anything that counts the user.** No streaks, no scores, no productivity
  metrics, no "you haven't used this in N days". This is the one rule the
  whole project is organised around.
- **Anything that phones home.** No telemetry, no analytics, no accounts, no
  cloud. `mori update` is the only command that touches the network.
- **A dependency that isn't earning its place.** The dependency list is short
  and the bar for adding to it is high.

None of these are personal — they're just the shape of the project, and it's
better to hear it before you write the code than after.

## Opening an issue

Bug reports and ideas are both welcome. Please don't paste anything private
into an issue: a redacted example of a page you wrote is always enough.

There's a sibling project, [tuki](https://github.com/rmpato/tuki),
with the same conventions. A fix that applies to both is worth mentioning.
