# Contributing

## Building

Requires Go 1.26+ and [golangci-lint](https://golangci-lint.run/) v2.

```console
$ make            # lint, test, build
$ make build      # bin/foxtrainer
$ make test       # go test -race ./...
$ make lint
$ make fmt
```

## Project layout

| Path | Contents |
|---|---|
| `cmd/foxtrainer/` | Entry point |
| `internal/cli/` | Command-line parsing and dispatch |
| `internal/config/` | Runtime configuration. **The only package that reads the environment** |
| `internal/paths/` | foxtrainer's folders and Firefox profile roots, per platform |
| `internal/answers/` | Loading and saving `config.toml` |
| `internal/logger/` | Levelled logging |
| `internal/fsutil/` | Atomic file writes and other filesystem helpers |
| `docs/` | User and developer documentation |

## Code conventions

The linter enforces most of these (`.golangci.toml`):

- **Descriptive names.** No single-character or cryptic identifiers, loop variables and test
  parameters included.
- **The environment is read in one place:** `internal/config` normalises it into a typed
  `Config` and reports every problem at once.
- **No ignored errors.** Return them wrapped with context (`%w`), or log them and record them
  somewhere durable. The only exception is a failed write to stderr or the log itself.
- **Single exit where reasonable.** Never `return` from inside a loop: set the result, `break`,
  return at the end. Guard clauses at the top of a function are fine. (Review-enforced.)
- **Few dependencies.** Each must earn its place.
- **Comments:** a one-line doc comment per function saying what it does; inline comments only
  where the mechanism is non-obvious. Design rationale lives in these docs.
- **Formats:** TOML for data and config. No YAML, except where a tool requires it (GitHub Actions).

## Testing

- Unit tests use temporary folders and a fake `HOME`. They never read or modify your real
  Firefox profiles.
- End-to-end tests (coming) run Firefox headless on throwaway profiles and check the result over
  Marionette.

## Reporting problems

Open an issue on [GitHub](https://github.com/headwalluk/foxtrainer/issues). For a configuration
problem, include `foxtrainer --version`, your Firefox version, and the output of `foxtrainer diff`.
