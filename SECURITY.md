# Security

mori is a private, local-first journal that lives in your terminal. Two things about it are worth a security policy:

- **It holds something private.** Your journal — plain markdown files under ~/.local/share/mori/journal is written at
  `0600` in a directory only you can read. Anything that widens those
  permissions, writes outside that directory, or puts its contents somewhere
  you didn't ask for is a security bug.
- **It replaces its own binary.** `mori update` downloads a release archive,
  checks it against the sha256 published in that release's `checksums.txt`,
  and only then swaps the running binary — keeping the old one aside until the
  new one is in place. Anything that lets unverified bytes through that path
  is a security bug.

## Reporting

Please **don't open a public issue** for a vulnerability.

Use [private vulnerability reporting](https://github.com/rmpato/mori/security/advisories/new),
which goes to the maintainer and nobody else. Include what you did, what
happened, and what you expected. A proof of concept helps; so does the version
(`mori --version`) and your OS.

You'll get an acknowledgement within a few days. This is a small project
maintained by one person in their spare time, so please be patient with
timelines — but a real vulnerability will always be taken seriously.

## In scope

- Anything that exposes your journal to another user or process
- Anything that lets an attacker influence what `mori update` installs
- Anything in `install.sh` that runs code you didn't intend
- Path traversal, permission widening, or writes outside the data directory

## Out of scope

- Anyone who already has read access to your account and your files. mori
  stores your data in your home directory; it cannot defend against a
  compromised machine, and it doesn't pretend to.
- The absence of encryption at rest. mori deliberately stores plain files
  you can read, back up and edit yourself. Encrypting them is a feature
  request, not a vulnerability.
- Denial of service caused by hand-editing your own data file into something
  malformed.

## What mori does not do

No network calls except `mori update`. No telemetry. No accounts. No cloud.
If you see mori talking to anything else, that is a bug worth reporting.
