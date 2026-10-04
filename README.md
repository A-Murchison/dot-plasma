# dotplasma

`dotplasma` saves your KDE Plasma desktop configuration into a plain folder you can inspect, back up, or commit to Git.

Use it to keep a snapshot of your Plasma setup, see what changed after desktop tweaks, and move reviewed profiles between machines without immediately applying them. It is built for users who want Plasma config backups without a cloud account, root access, or hidden state.

It tracks only an allowlisted set of Plasma files from `~/.config` and `~/.local/share`.

## What you get

- **Snapshots of Plasma config** stored as regular files under `profiles/<name>/`.
- **Stable output for Git** so you can review and commit changes yourself.
- **Diffs against your live desktop** to see what changed since a snapshot.
- **Safe discovery commands** before you copy anything.
- **Import validation** for a profile received from another machine, without applying it.
- **Conservative scope**: only known Plasma config files are included by default.

> [!WARNING]
> Treat saved profiles as private. Plasma config can contain local paths, widget settings, wallpaper paths, monitor layout details, and other machine-specific data. Review files before sharing or publishing them.

## Install from source

From a checkout of this repository:

```sh
go install ./cmd/dotplasma
```

Or run without installing:

```sh
go run ./cmd/dotplasma --help
```

## Typical workflow

Check that your system and output directory are usable:

```sh
dotplasma doctor --out ~/dotplasma-backups
```

See which known Plasma files exist on this machine:

```sh
dotplasma inspect-live
```

Save a profile snapshot:

```sh
dotplasma save main --out ~/dotplasma-backups
```

Preview the save first, without writing files:

```sh
dotplasma save main --out ~/dotplasma-backups --dry-run
```

Compare the saved profile with your current live Plasma config:

```sh
dotplasma diff main --out ~/dotplasma-backups
```

Import a profile directory from another machine without applying it:

```sh
dotplasma import ~/Downloads/plasma-profile laptop --out ~/dotplasma-backups --dry-run
```

## Output layout

A saved profile is written as plain files:

```text
~/dotplasma-backups/
  profiles/
    main/
      profile.toml
      README.md
      files/
        config/
        local-share/
      manifests/
```

You can inspect, copy, back up, or commit this directory yourself.

## Commands

```text
dotplasma doctor [--out DIR]
dotplasma inspect-live
dotplasma save <profile> [--out DIR] [--dry-run]
dotplasma diff [profile] [--out DIR] [--format text]
dotplasma import <source-dir> <profile> [--out DIR] [--dry-run]
dotplasma version
```

Notes:

- `diff` exits `0` when there are no differences, `1` when differences are found, and `2` for usage or runtime errors.
- `apply` and `list` are not ready for end-user use yet.
- `--format json` for `diff` is reserved for future support; use the default text output today.

## Safety model

- No network access.
- No root requirement.
- No automatic Git integration.
- Writing commands support `--dry-run`.
- Live restore is not available yet.
- Saved output is intended to be deterministic and reviewable.
