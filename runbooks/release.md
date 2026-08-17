# Cut a release

Pushing a `v*` tag is the whole trigger. Everything else is automatic.

## Before you start

- You're on `main`, up to date, and CI is green.
- Nothing to bump by hand: the version is stamped from the tag at build time
  (`.goreleaser.yaml` → `-X …/internal/cli.Version={{ .Tag }}`). There is no
  version constant in the source.
- Pick a version. Users get it via `mori update`, which compares semver, so
  `v0.2.0` must sort above the current tag.

```sh
git checkout main && git pull
make check                            # fmt, vet, test
gh run list --limit 3                 # last CI runs should be green
git describe --tags --abbrev=0        # what's current
```

CI green matters more here than usual: GoReleaser runs `go test ./...` again
as a pre-release hook, so a flaky test fails the release halfway through.

## Release

```sh
git tag -a v0.2.0 -m "mori v0.2.0"
git push origin v0.2.0
```

Then watch it:

```sh
gh run watch --repo rmpato/mori       # or: gh run list --limit 3
```

`.github/workflows/release.yml` fires on the tag and runs GoReleaser, which
runs the tests, cross-compiles for macOS, Linux and Windows, and publishes a
GitHub release with a changelog built from the commits since the last tag.

Takes about two minutes.

## Verify

Don't trust the green tick — install it the way a stranger would.

```sh
# Six assets: five archives plus checksums.txt.
gh release view v0.2.0 --repo rmpato/mori --json assets --jq '[.assets[].name]'

# The real install path, into a throwaway directory.
DIR=$(mktemp -d)
curl -fsSL https://raw.githubusercontent.com/rmpato/mori/main/install.sh \
  | sh -s -- --dir "$DIR" --no-modify-path
"$DIR/mori" --version          # should print the tag you just pushed
"$DIR/mori" update --check     # should say it's the latest
rm -rf "$DIR"
```

The archive names are built independently in `.goreleaser.yaml`, `install.sh`
and `internal/update.AssetName`. `TestInstallScriptAsksForTheSameArchive`
guards that they agree, but the check above is what proves it end to end.

## If it goes wrong

A tag that built badly can be replaced, as long as nobody has installed it
yet:

```sh
gh release delete v0.2.0 --repo rmpato/mori --yes
git push --delete origin v0.2.0
git tag -d v0.2.0
```

Then fix, and tag again. Once a version is out in the world, prefer cutting
`v0.2.1` over rewriting `v0.2.0` — `mori update` caches nothing, but someone
may already have the old bytes.
