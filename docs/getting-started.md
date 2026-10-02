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
$ foxtrainer configure
```

1. Choose the **instance** (Firefox install + profile) to configure.
2. Optionally start from another instance's answers.
3. Answer the questions: overall feel, AI features, privacy, HTTPS-only, language and spelling.
4. Review the summary, then apply or just save.

## Everyday use

Close Firefox, then:

```console
$ foxtrainer apply
```

`apply` rebuilds the configuration for every configured instance from your saved answers and the
latest catalogue, and tells you what changed. Use `foxtrainer diff` to preview without writing.

## Where foxtrainer keeps things

Run `foxtrainer paths` to see the exact folders on your system.

| What | Linux (default) | macOS |
|---|---|---|
| Your answers | `~/.config/foxtrainer/config.toml` | `~/Library/Application Support/foxtrainer/config.toml` |
| Downloaded sources | `~/.cache/foxtrainer/` | `~/Library/Caches/foxtrainer/` |
| Manifests and backups | `~/.local/state/foxtrainer/` | `~/Library/Application Support/foxtrainer/state/` |

On Linux, `XDG_CONFIG_HOME`, `XDG_CACHE_HOME` and `XDG_STATE_HOME` are honoured.
