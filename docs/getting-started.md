# Getting started

> **Draft.** foxtrainer is in early development. This page will be completed when the first
> usable release ships.

## Requirements

- Linux or macOS. Windows support is not decided yet.
- Firefox (release, ESR, Beta, Developer Edition or Nightly) that has been run at least once,
  so it has a profile.

## Install

Pre-built binaries will be published on the
[releases page](https://github.com/headwalluk/foxtrainer/releases). Until then, build from source
with Go 1.26 or later:

```console
$ go install github.com/headwalluk/foxtrainer/cmd/foxtrainer@latest
```

## First run

```console
$ foxtrainer list                                   # find your instances
$ foxtrainer configure --profile dev-edition-default-1 --feel balanced --ai off
$ foxtrainer diff                                   # see exactly what would change
$ foxtrainer apply                                  # close Firefox first
```

`configure` saves answers for one instance (Firefox install + profile). Answers you don't pass
keep their saved value, or the defaults: feel `balanced`, AI `off`, privacy `strict`, HTTPS-only
on, and languages from your locale. An interactive wizard is on its way; flags will keep working
for scripts.

| Flag | Values |
|---|---|
| `--feel` | `lean`, `balanced`, `full` |
| `--ai` | `off` (blocks every AI feature, including translations), `local` (blocks cloud AI only), `all` |
| `--privacy` | `standard`, `strict` (`hardened` is coming) |
| `--https-only` | `true`, `false` |
| `--languages` | e.g. `en-GB,en` |

## Everyday use

Close Firefox, then:

```console
$ foxtrainer apply
```

`apply` rebuilds the configuration for every configured instance from your saved answers and the
current catalogue, and reports what changed. `foxtrainer diff` (or `apply --dry-run`) previews it
without writing anything. Add `--offline` to use only previously downloaded upstream files.

## Where foxtrainer keeps things

Run `foxtrainer paths` to see the exact folders on your system.

| What | Linux (default) | macOS |
|---|---|---|
| Your answers | `~/.config/foxtrainer/config.toml` | `~/Library/Application Support/foxtrainer/config.toml` |
| Downloaded sources | `~/.cache/foxtrainer/` | `~/Library/Caches/foxtrainer/` |
| Manifests and backups | `~/.local/state/foxtrainer/` | `~/Library/Application Support/foxtrainer/state/` |

On Linux, `XDG_CONFIG_HOME`, `XDG_CACHE_HOME` and `XDG_STATE_HOME` are honoured.
