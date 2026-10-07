# dotplasma — KDE Plasma config backup, diff, and restore CLI

Save, compare, and restore KDE Plasma desktop configuration as plain files.

`dotplasma` is an open-source CLI for people who want their KDE Plasma setup to be inspectable, backed up, and easy to move between machines. It snapshots an allowlisted set of Plasma desktop configuration files from your home directory into regular folders that you can review, copy, or commit to Git yourself. 


<img width="960" height="540" alt="dotplasmademo" src="https://github.com/user-attachments/assets/7e330581-bcd3-4ced-ac8e-04b0b2206565" />


## Good fit

`dotplasma` is most useful when you want to:

- keep a known-good Plasma layout before experimenting with panels, widgets, shortcuts, or desktop settings;
- review what changed in Plasma's config files before committing dotfiles;
- back up and restore Plasma configuration on the same machine;
- move a profile between similar Plasma installations while reviewing it before applying.
- backup to git
- 
`dotplasma` focuses on that workflow: snapshot, diff, review, and only then restore if you choose to.

## Features

- **Save Plasma profiles** into `profiles/<name>/` as normal files.
- **Review changes** between a saved profile and your live desktop.
- **Import profiles safely** from another machine without applying them.
- **Restore cautiously** with `--dry-run` support and automatic pre-restore backups.
- **Git-friendly output** that you control; no automatic commits or remote sync.
- **Conservative file scope** using an allowlist of known Plasma config files.
- **No root and no network access** required.

## Commands

```text
dotplasma [--config PATH] doctor [--out DIR]
dotplasma [--config PATH] inspect-live
dotplasma [--config PATH] save <profile> [--out DIR] [--dry-run]
dotplasma [--config PATH] diff [profile] [--out DIR] [--format text] [--color auto|always|never] [--verbose]
dotplasma [--config PATH] list [--out DIR]
dotplasma [--config PATH] import <source-dir> <profile> [--out DIR] [--dry-run]
dotplasma [--config PATH] apply <profile> [--out DIR] [--dry-run] [--reload none|plasmashell]
dotplasma [--config PATH] version
```

<img width="982" height="693" alt="dotplasma-diff" src="https://github.com/user-attachments/assets/49b5bb1c-dc49-4b61-a544-fb2bb1e0d14e" />


Useful notes:

- `diff` exits `0` when there are no differences, `1` when differences are found, and `2` for usage or runtime errors.
- `list` shows saved profile names, captured Plasma version when available, distro when available, and saved time.
- `apply` restores only manifest-listed files that are also present in the current allowlist.
- `apply` refuses old or incomplete Plasma Shell layout snapshots that are missing required screen-binding data.
- `--format json` for `diff` is reserved for future support; use the default text output today.
- `diff` shows a changed-files summary by default. Add `--verbose` to show per-setting values in vertical detail blocks.
- `diff --color auto` colorizes statuses only when writing to a terminal. Use `always` or `never` to override it.


## Quick start

By default, `dotplasma` stores profiles and restore backups under:

```text
${XDG_CONFIG_HOME:-~/.config}/dotplasma/
```

You can also choose a separate directory with `--out`, which is useful if you want to keep profiles in a Git repository.

```sh
# Check that dotplasma can see your Plasma files and output directory.
dotplasma doctor 

# Show which known Plasma config files exist on this machine.
dotplasma inspect-live

# Save your current desktop as a profile named "main".
dotplasma save main 

# See what changed since that snapshot.
dotplasma diff main 

# Preview a restore without writing anything.
dotplasma apply main --dry-run
```

## What gets captured?

`dotplasma` tracks an allowlisted set of KDE Plasma configuration files, including files related to panels, widgets, desktop layout, shortcuts, Plasma Shell, KWin, and display/layout state where supported.

Run this to see exactly which tracked files exist on your machine:

```sh
dotplasma inspect-live
```

## Configuration

`dotplasma` reads this config file by default:

```text
${XDG_CONFIG_HOME:-~/.config}/dotplasma/config.toml
```

Supported setting:

```toml
# Root directory containing profiles/ and backups/.
output_dir = "/home/adam/dotplasma-backups"
```

`--out DIR` overrides the config file for a single command. `--config PATH` reads a different config file.


## Common workflows

### Keep Plasma config in Git

```sh
dotplasma save main --out ~/src/plasma-profiles
cd ~/src/plasma-profiles
git status
git diff
git add profiles/main
git commit -m "chore: update plasma profile"
```

`dotplasma` does not run Git commands for you. The saved files are yours to inspect, edit, commit, encrypt, or ignore.

### Preview before writing

Writing commands support `--dry-run`:

```sh
dotplasma save main --out ~/dotplasma-backups --dry-run
dotplasma import ~/Downloads/plasma-profile laptop --out ~/dotplasma-backups --dry-run
dotplasma apply main --out ~/dotplasma-backups --dry-run
```

### Move a profile to another machine

On the source machine:

```sh
dotplasma save laptop --out ~/dotplasma-backups
```

Copy `~/dotplasma-backups/profiles/laptop` to the other machine, then import it:

```sh
dotplasma import ~/Downloads/laptop laptop --out ~/dotplasma-backups
```

Review it before applying:

```sh
dotplasma diff laptop --out ~/dotplasma-backups
dotplasma diff laptop --out ~/dotplasma-backups --verbose
dotplasma apply laptop --out ~/dotplasma-backups --dry-run
```

Apply only after you are comfortable with the planned changes:

```sh
dotplasma apply laptop --out ~/dotplasma-backups
```

Existing tracked live files that change are backed up first under `backups/<profile>/<timestamp>/`.

## Plasma Shell reloads

Some panel, widget, wallpaper, and desktop layout changes may not appear until Plasma Shell is restarted. `dotplasma` never does this automatically.

If you explicitly request it, `dotplasma` can stop `plasmashell`, apply the profile while it is stopped, and start it again afterward:

```sh
dotplasma apply main --out ~/dotplasma-backups --reload plasmashell
```

During this reload, panels, desktop widgets, wallpaper, and command bars can briefly disappear. To preview the plan without stopping Plasma Shell or writing files:

```sh
dotplasma apply main --out ~/dotplasma-backups --reload plasmashell --dry-run
```

## Install

### Go install

If you have Go installed, this is the simplest option:

```sh
go install github.com/A-Murchison/dot-plasma/cmd/dotplasma@latest
dotplasma --help
```

### Manual install

From a local checkout:

```sh
go build -o ~/.local/bin/dotplasma ./cmd/dotplasma
# or run without installing
go run ./cmd/dotplasma --help
```


### Fedora, RHEL, CentOS, and other RPM-based systems

For releases that include RPM packages, download the RPM for your architecture from the [GitHub Releases](https://github.com/A-Murchison/dot-plasma/releases) page, then install it with `dnf`:

```sh
sudo dnf install ./dotplasma-0.1.0-1.x86_64.rpm
dotplasma --help
```

You can also install an RPM directly from a release URL:

```sh
sudo dnf install \
  https://github.com/A-Murchison/dot-plasma/releases/download/v0.1.0/dotplasma-0.1.0-1.x86_64.rpm
```

Replace `v0.1.0` and the package filename with the release and architecture you want.

### Debian and Ubuntu

For releases that include Debian packages, download the `.deb` package for your architecture from the [GitHub Releases](https://github.com/A-Murchison/dot-plasma/releases) page, then install it with `apt`:

```sh
sudo apt install ./dotplasma_0.1.0_amd64.deb
dotplasma --help
```



## Output layout

A saved profile is stored as plain files:

```text
dotplasma-backups/
  profiles/
    main/
      profile.toml
      README.md
      files/
        config/
        local-share/
      manifests/
  backups/
    main/
      20250102T030405Z/
        files/
          config/
          local-share/
```

You can inspect, copy, back up, or commit this directory yourself.


## Safety model

`dotplasma` is intentionally conservative:

- It does not require root.
- It does not access the network.
- It does not make Git commits or push anywhere.
- It only tracks a known allowlist of Plasma files.
- Writing commands support `--dry-run`.
- Restore creates backups before updating tracked live files.
- Plasma Shell reload is opt-in.


> [!WARNING]
> Treat saved profiles as private. Plasma config can include local paths, widget settings, wallpaper paths, monitor layout details, and other machine-specific data. Review files before sharing or publishing them.


## Contributing

Issues, bug reports, real-world Plasma edge cases, and focused pull requests are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for development commands and project guidelines.

---

Shout out to https://github.com/EliverLara/Nordic for the nordic theme in the gifs.

Shout out to https://github.com/Prayag2/konsave for the inspiration

KDE Plasma backup, Plasma desktop config backup, KDE dotfiles, Plasma dotfiles, KDE settings migration, Plasma panel layout backup, KWin configuration backup, `plasmashellrc`, `plasma-org.kde.plasma.desktop-appletsrc`, and `konsave` alternative.
