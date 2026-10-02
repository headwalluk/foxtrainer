# foxtrainer

A cross-platform CLI/TUI that "tames" Firefox. It asks a few high-level questions per instance,
saves the answers, and builds each profile's `user.js` from a catalogue of pref groups drawn from
upstream projects (Betterfox, arkenfox) and foxtrainer's own groups. `foxtrainer apply`
re-applies everything after Firefox or upstream changes.

## Terminology

- **Instance**: a Firefox install (binary) plus a profile. Each instance has its own saved answers.
- **Catalogue**: the TOML data that maps answers → groups → prefs, with each upstream source pinned
  by full commit SHA and sha256.
- **Source**: an upstream pref collection (Betterfox, arkenfox) or Mozilla's own pref data, used
  for validation only.

## Scope rules

- Profile-only. No root, no `policies.json`, no privilege escalation on any OS.
- foxtrainer manages prefs. It never moves bookmarks, logins or other data between profiles.
- Never brand generated output as Betterfox or arkenfox. Credit both (MIT) as in
  `THIRD-PARTY-NOTICES.md`. Don't copy content from GPL/AGPL projects (Phoenix, Narsil, ffprofile).
- Linux and macOS first. Windows only where it comes cheaply.

## Repo layout

- `cmd/foxtrainer/`: entry point.
- `internal/`: all packages. `internal/config` is the only place that reads the environment;
  `internal/paths` resolves config, cache and state folders and the Firefox profile roots.
- `docs/`: public documentation for users, developers and security reviewers. The README links here.
- `dev-notes/`: **gitignored**, local only. It holds research (`dev-notes/research/`) and the
  live roadmap `dev-notes/00-project-tracker.md`. Tick items off there as work completes and
  add a line to its done log. It may not exist in other clones.

## Commands

`make` (lint, race tests, build) · `make build` → `bin/foxtrainer` · `make test` · `make lint` · `make fmt`.
Lint config is `.golangci.toml`; `make` must be clean before anything is ticked off in the tracker.

## Go conventions

These are the global preferences, as they apply in Go:

- Names say what things are: no single-character or cryptic identifiers, loop variables
  included. Shell scripts use long ALL-CAPS variable names, always braced (`${PROFILE_DIR}`).
- The environment is read only in `internal/config`, which normalises it into a typed struct at
  start-up and reports **every** problem at once. `forbidigo` enforces this.
- No silently ignored errors. Every error is returned (wrapped with `%w` and context) or logged
  at warn/error **and** recorded somewhere durable: the apply report or the instance manifest.
  No `_ = fn()` for calls that can fail meaningfully.
- Single exit where reasonable. **Never `return` from inside a loop**: set the result, `break`,
  return at the end. Guard clauses at the top of a function are fine. Lint can't check this,
  so review for it.
- Lean dependencies. Current allowance: `BurntSushi/toml`, the Charm TUI stack
  (`huh`, `bubbletea`, `lipgloss`) and a pure-Go SQLite driver. Anything else must earn its place.
- Logging goes through our small stdlib-based logger, gated on the configured level.
- Paths: always `filepath.Join`; never hand-concatenate separators.
- Comments: one docblock line per exported function saying what it does. Inline comments only
  where the mechanism is non-obvious. Rationale and history belong in `docs/` (or `dev-notes/`
  research), referenced from the comment, not restated.
- File format: TOML for the catalogue and the saved answers. No YAML.

## Firefox safety rules

- Never modify a profile while Firefox holds it. The in-use check is a non-blocking `fcntl`
  lock on `.parentlock`; ignore the `lock` symlink (Firefox 158 leaves it even after a clean exit). Hold the
  lock for the whole apply.
- Editing `profiles.ini` or `installs.ini` requires **every** Firefox using that root to be closed.
- Back up before writing; write atomically with mode 0600.
- Remove from `prefs.js` only pref names that foxtrainer itself wrote earlier (per-instance
  manifest). Never touch anything else.
- Profile Groups: some prefs (27 in Firefox 158; the list varies by version) are group-wide and stored in `Profile Groups/<storeID>.sqlite`
  `SharedPrefs`, which overrides `user.js` at startup. Write managed shared prefs there too, with
  the whole group closed.
- Never set `toolkit.profiles.storeID`, `browser.profiles.*`, or prefs that a policy locks.

## Testing

- Unit tests run against a fake `HOME` and fixture `profiles.ini` / `installs.ini` layouts.
- End-to-end tests run Firefox headless on throwaway profiles in a temp dir and query it over
  Marionette (`--headless --marionette -remote-allow-system-access -no-remote --profile DIR`,
  with a non-default `marionette.port` in `user.js`).
- **Never** run tests against the developer's real profiles in `~/.mozilla` or `~/.config/mozilla`.

## Versioning and commits

- Start at 0.1.0; bump the minor version at the end of each milestone in the tracker.
- Commit at milestone boundaries once the user agrees the milestone is done; otherwise only when asked.
