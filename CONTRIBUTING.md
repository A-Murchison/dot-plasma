# Contributing to dotplasma

Thanks for considering a contribution.

## Project priorities

1. Do not corrupt user configuration.
2. Produce deterministic output suitable for Git.
3. Prefer a small, well-understood allowlist over broad, noisy backups.
4. Make restore/apply behavior easy, explicit, and reversible.

## Development commands

```sh
go test ./...
go test -race -count=1 -timeout 60s ./...
```

If your environment has the optional tools installed:

```sh
make check
```

## Dependency policy

Ask before adding third-party dependencies. The CLI may use Cobra later, but the initial skeleton intentionally uses the standard library so the project can start without dependency churn.

## Code style

- Wrap errors with context using `%w`.
- Do not log and return the same error.
- No panics outside `main` or package init.
- Sort keys before producing user-visible output.
- Keep interfaces small and define them in the consuming package.
- Prefer table-driven tests with `t.Run` where useful.

## KDE config handling

KDE config is not generic INI. Parser changes need fixture tests for:

- nested groups such as `[Containments][1][Applets][2]`
- localized keys such as `Name[en_US]=...`
- flags such as `[$i]` and `[$e]`
- comments and blank lines
- duplicate keys/groups
- malformed input with useful line-numbered errors

When uncertain whether a file belongs in the allowlist, leave it out until it is understood and documented.
