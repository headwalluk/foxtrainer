# Contributing

For people changing foxtrainer's code or catalogue. [Architecture](architecture.md) explains how
the pieces fit together, and [The catalogue](catalogue.md) covers the pref data.

## Building

[Compiling from source](compiling.md) covers installing Go and building a binary. To work on
foxtrainer you also need [golangci-lint](https://golangci-lint.run/) v2 and
[ShellCheck](https://www.shellcheck.net/).

```console
$ make            # lint, test, build
$ make build      # bin/foxtrainer
$ make test       # go test -race ./...
$ make e2e        # end-to-end tests against a real Firefox
$ make lint       # golangci-lint, plus shellcheck on install.sh
$ make fmt
$ make release    # release tarballs and SHA256SUMS in dist/
```

foxtrainer is developed and tested on Linux. It builds on macOS, but is untested there.
Windows is not supported.

## Project layout

| Path | Contents |
|---|---|
| `cmd/foxtrainer/` | Entry point |
| `internal/cli/` | Command-line parsing and dispatch |
| `internal/config/` | Runtime configuration. **The only package that reads the environment** |
| `internal/paths/` | foxtrainer's folders and Firefox profile roots, per platform |
| `internal/answers/` | Loading and saving `config.toml` |
| `internal/wizard/` | The interactive `configure` wizard |
| `internal/apply/` | Building `user.js`, backups, `prefs.js` cleanup and the per-instance manifest |
| `internal/language/` | The generated language and spelling group |
| `internal/buildinfo/` | The version stamped in at build time |
| `internal/logger/` | Levelled logging |
| `internal/fsutil/` | Atomic file writes and other filesystem helpers |
| `catalogue/` | The catalogue data (`catalogue.toml`, `groups/*.toml`), compiled into the binary |
| `internal/catalogue/` | Loading, checking and resolving the catalogue |
| `internal/sources/` | Downloading and verifying pinned upstream files; `betterfox/` reads Betterfox's format |
| `internal/prefs/` | Pref values and the `user_pref(...)` tokeniser |
| `internal/firefox/` | Firefox installs, profiles, locks, the install hash and Profile Group shared prefs |
| `internal/cityhash/` | CityHash64 v1.0, as Mozilla uses for install hashes |
| `internal/e2e/` | End-to-end tests (build tag `e2e`) |
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

See [The catalogue](catalogue.md#making-a-change). In short: edit `catalogue/`, then
`foxtrainer catalogue check` and `make test` must both pass.

## Testing

- Unit tests use temporary folders and a fake `HOME`. They never read or modify your real
  Firefox profiles.
- `make e2e` runs foxtrainer against a real headless Firefox on a throwaway profile in a fake
  `HOME`, with a hermetic environment, and checks the values Firefox saved in `prefs.js`. It
  uses `firefox-devedition` by default; pass another binary with
  `go test -tags e2e ./internal/e2e/ -args -firefox=PATH`. The tests are skipped when no
  Firefox is found.
- The interactive wizard needs a real terminal; piped tests cover only `configure --accessible`.

## Releasing

Pushing a `v*` tag runs `.github/workflows/release.yml`. It lints, runs the tests, runs
`make release`, checks the catalogue against the live pinned upstream files, and publishes a
GitHub Release.

`make release` cross-compiles for linux/amd64 and linux/arm64. Each
`dist/foxtrainer_<os>_<arch>.tar.gz` holds the binary, `LICENSE`, `THIRD-PARTY-NOTICES.md` and
`README.md`, and `dist/SHA256SUMS` lists their checksums. `install.sh` at the repository root
installs the latest release (or `FOXTRAINER_VERSION`) into `~/.local/bin` (or
`FOXTRAINER_INSTALL_DIR`).

## Reporting problems

Open an issue on [GitHub](https://github.com/headwalluk/foxtrainer/issues). For a configuration
problem, include `foxtrainer version`, your Firefox version, and the output of `foxtrainer diff`.
