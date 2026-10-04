Go 1.26.8. Layout: cmd/ (binaries), internal/ (everything real), pkg/ (only if
genuinely importable by other repos).

## Commands
- Everything: `make check`  (gofmt, go vet, golangci-lint, go test -race)
- Test:       `go test -race -count=1 -timeout 60s ./...`
- One test:   `go test -run TestName ./internal/pkg -v`
- Bench:      `go test -bench=. -benchmem ./internal/pkg`

`-race` is always on and `-count=1` disables caching. Do not remove either to
make the suite faster.

## Errors
- Wrap with context and %w: `fmt.Errorf("load user %s: %w", id, err)`.
  Never %v — it silently breaks errors.Is for every caller.
- Compare with errors.Is / errors.As, never ==.
- Handle once: add context and return. Do not log and return the same error.
- No naked returns. No panic outside main() and package init.

## Concurrency
- Every goroutine needs a guaranteed exit path. If it can block on a send,
  buffer the channel or give it a context.
- Prefer errgroup.WithContext over WaitGroup + channels by hand.
- context.Context is the first parameter of anything doing I/O, and is
  actually plumbed through — not accepted and dropped.
- TestMain calls goleak.VerifyTestMain.
- Never range a map to produce output. Sort the keys.

## Style the linter cannot enforce
- Interfaces are declared by the CONSUMER, in the consumer's package, and are
  small. One or two methods. Return concrete types.
- Table-driven tests with t.Run subtests.
- Use the current stdlib: os.ReadFile not ioutil, any not interface{},
  slices/maps packages, log/slog not logrus, math/rand/v2.
- No dependency injection framework. Wire it explicitly in main().

## Landmines
- internal/scheduler is leader-elected. Changing tick timing needs an ops review.
- internal/proto is generated. Edit the .proto and run `make proto`.
- cmd/migrate: write migrations, never run them. A human runs migrations.

## Project
- Ensure the CONTRIBUTING.md is up-to-date.
- Use conventional commits for commit messages.
