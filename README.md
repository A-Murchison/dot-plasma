# dotplasma

`dotplasma` is a conservative CLI for inspecting, saving, comparing, and eventually restoring KDE Plasma configuration.

It is designed around a plain folder layout so you can put the generated output under Git yourself. `dotplasma` does not create commits, contact remotes, require root, or sync data to a cloud service.


## Install from source

```sh
go install ./cmd/dotplasma
```

Or run from a checkout:

```sh
go run ./cmd/dotplasma --help
```

## Basic usage

Check whether the local machine looks usable for `dotplasma`:

```sh
dotplasma doctor
```

Check a specific output directory:

```sh
dotplasma doctor --out ~/dotfiles
```

Show which known Plasma configuration files exist on the live system:

```sh
dotplasma inspect-live
```

Example output:

```text
ROOT    PATH                                     STATUS   REQUIRED  PARSER
config  kdeglobals                               present  false     kconfig
config  kwinrc                                   present  false     kconfig
config  plasma-org.kde.plasma.desktop-appletsrc  present  false     kconfig
```

## Command overview

```text
dotplasma doctor [--out DIR]
dotplasma inspect-live
dotplasma save <profile> [--out DIR] [--dry-run]
dotplasma diff [profile] [--out DIR] [--format text|json]
dotplasma list [--out DIR]
dotplasma apply <profile> [--out DIR] [--dry-run]
```

Currently implemented:

- `doctor`: validates the embedded allowlist, live roots, output directory, and Plasma version detection.
- `inspect-live`: lists allowlisted live files and whether they are present.
- `version`: prints the `dotplasma` version.

Still in development:

- `save`
- `diff`
- `list`
- `apply`

## Safety rules

- No network access.
- No root requirement.
- No automatic Git integration.
- Every writing command must support `--dry-run`.
- Live writes are limited to `~/.config` and `~/.local/share`.
- Output is intended to be deterministic so users can commit snapshots to Git.
- Restore/apply behavior will remain conservative and will require backups first.

## Intended output layout

Profiles are planned to be stored under an output directory like this:

```text
profiles/<profile>/
  profile.toml
  README.md
  files/
    config/
      kdeglobals
      kwinrc
      plasmarc
      plasma-org.kde.plasma.desktop-appletsrc
  manifests/
    files.toml
    ignored.toml
    environment.toml
```

## Contributing

See `CONTRIBUTING.md` for development commands and `Plan.md` for the implementation roadmap.
