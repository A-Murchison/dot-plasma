# dotplasma

`dotplasma` is a conservative CLI for inspecting, saving, comparing, and eventually restoring KDE Plasma configuration.

It is designed around a plain folder layout so you can put the generated output under Git yourself. 

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

Create or update a profile snapshot:

```sh
dotplasma save main --out ~/dotplasma-backups
```

Preview a save without writing files:

```sh
dotplasma save main --out ~/dotplasma-backups --dry-run
```

Compare a saved profile against the live system:

```sh
dotplasma diff main --out ~/dotplasma-backups
```

> [!WARNING]
> Saved profiles should be treated as private by default. Review generated files before sharing or committing them to a public repository; Plasma configuration can contain local paths, widget settings, wallpaper paths, monitor layout details, and other machine-specific data.

Example discovery output:

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
dotplasma diff [profile] [--out DIR] [--format text]
dotplasma list [--out DIR]
dotplasma apply <profile> [--out DIR] [--dry-run]
```

Currently implemented:

- `doctor`: validates the embedded allowlist, live roots, output directory, and Plasma version detection.
- `inspect-live`: lists allowlisted live files and whether they are present.
- `save`: snapshots allowlisted files into `profiles/<profile>/`.
- `diff`: compares saved profile files against the live system with text output. It exits `0` for no differences, `1` for differences found, and `2` for usage/runtime errors.
- `version`: prints the `dotplasma` version.

Still in development:

- `list`
- `apply`

## Safety rules

- No network access.
- No root requirement.
- No automatic Git integration.
- Every writing command must support `--dry-run`.
- Live writes are limited to `~/.config` and `~/.local/share`.
- Output is intended to be deterministic so users can commit snapshots to Git.
- Saved profiles may contain personal or machine-specific data; review before publishing publicly.
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
    privacy.md
```

## Contributing

See `CONTRIBUTING.md` for development commands.
