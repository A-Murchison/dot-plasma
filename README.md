# dotplasma

`dotplasma` is a Go CLI for saving, comparing, and eventually restoring KDE Plasma configuration using a plain folder layout that users can commit to Git themselves.

The project is intentionally conservative: backup and diff come first; restore comes later.

## Current status

Milestone 0 is complete:

- Go module exists.
- Cobra CLI skeleton exists.
- Initial command surface exists.
- Initial allowlist data file exists.
- Tests can be run with `go test ./...`.

Milestone 1 is complete enough to proceed:

- `internal/kconfig` contains the first KConfig parser implementation.
- Parser tests cover nested groups, localized keys, flags, comments, blank lines, duplicate keys, malformed groups, and key-level diffs.
- Sanitized committed fixtures and ignored local fixtures both support parser round-trip testing.

Milestone 2 has started:

- Embedded allowlist loading and validation exists.
- Safe live-root path joining exists.
- `doctor` and `inspect-live` perform read-only discovery.

Save, diff, list, and apply currently print "not implemented yet". That is deliberate.

## Install from source

```sh
go install ./cmd/dotplasma
```

Or run locally:

```sh
go run ./cmd/dotplasma --help
```

## Commands planned

```text
dotplasma save <profile> [--out DIR] [--dry-run]
dotplasma diff [profile] [--out DIR] [--format text|json]
dotplasma list [--out DIR]
dotplasma doctor
dotplasma inspect-live
dotplasma apply <profile> [--out DIR] [--dry-run]
```

## Safety rules

- No network access.
- No root requirement.
- No automatic Git integration.
- Every writing command must support `--dry-run`.
- Live writes are limited to `~/.config` and `~/.local/share`.
- Output must be deterministic because users are expected to commit snapshots.

## Development

```sh
go test ./...
```

If available locally, run:

```sh
make check
```

See `Plan.md` for the implementation roadmap.
