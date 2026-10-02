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
| `catalogue/` | The catalogue data (`catalogue.toml`, `groups/*.toml`), compiled into the binary |
| `internal/catalogue/` | Loading, checking and resolving the catalogue |
| `internal/sources/` | Downloading and verifying pinned upstream files; `betterfox/` reads Betterfox's format |
| `internal/prefs/` | Pref values and the `user_pref(...)` tokeniser |
| `internal/firefox/` | Firefox installs, profiles, locks and the install hash |
| `internal/cityhash/` | CityHash64 v1.0, as Mozilla uses for install hashes |
| `testdata/upstream/` | Pinned upstream files used by the tests (MIT; see THIRD-PARTY-NOTICES.md) |
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

## Changing the catalogue

1. Edit `catalogue/groups/*.toml` (see [How it works](how-it-works.md#the-catalogue)).
2. Run `foxtrainer catalogue check` and `make test`. Both must pass.
3. Use `foxtrainer catalogue show` with different answers to see the effect.

To move an upstream pin, update `commit`, `tag` and the sha256 of each file in
`catalogue.toml`, copy the new files into `testdata/upstream/`, and resolve whatever the checker
reports. That's usually new prefs needing a home.

## Testing

- Unit tests use temporary folders and a fake `HOME`. They never read or modify your real
  Firefox profiles.
- End-to-end tests (coming) run Firefox headless on throwaway profiles and check the result over
  Marionette.

## Reporting problems

Open an issue on [GitHub](https://github.com/headwalluk/foxtrainer/issues). For a configuration
problem, include `foxtrainer --version`, your Firefox version, and the output of `foxtrainer diff`.
