# Getting started

Install the pre-built binary, answer a few questions, apply. To build foxtrainer yourself, or
to run it on macOS, see [Compiling from source](compiling.md).

## Requirements

- Linux on x86-64 or ARM64.
- Firefox from Mozilla's apt repository, a distribution package or a Mozilla tarball: release,
  ESR, Beta, Developer Edition or Nightly. Snap and Flatpak Firefox keep their profiles inside
  the sandbox, and foxtrainer doesn't find them yet.
- Firefox must have been run at least once, so it has a profile.

## Install

foxtrainer is a single binary that runs as your own user. It doesn't need root.

**Install script.** This downloads the latest release for your CPU, checks it against the
release's `SHA256SUMS` and puts it in `~/.local/bin`:

```console
$ curl -fsSL https://raw.githubusercontent.com/headwalluk/foxtrainer/main/install.sh | sh
```

Set `FOXTRAINER_VERSION=v1.0.0` to install a particular release, or `FOXTRAINER_INSTALL_DIR` to
install somewhere else. If the folder isn't on your `PATH`, the script prints the line to add to
your shell profile. Run the same command again to upgrade.

**By hand.** Download `foxtrainer_linux_amd64.tar.gz` (or `_arm64`) and `SHA256SUMS` from the
[releases page](https://github.com/headwalluk/foxtrainer/releases), then:

```console
$ sha256sum --check --ignore-missing SHA256SUMS
$ tar -xzf foxtrainer_linux_amd64.tar.gz
$ install -m 0755 foxtrainer_linux_amd64/foxtrainer ~/.local/bin/
```

Check it works:

```console
$ foxtrainer version
```

**To uninstall,** delete the binary. To also remove your saved answers, downloaded sources and
backups, delete the folders listed by `foxtrainer paths` (see below). The `user.js` foxtrainer
wrote stays in each profile until you delete it.

## First run

```console
$ foxtrainer configure
```

The wizard asks:

1. **Which instance** (Firefox install + profile) to configure. Skipped when there's only one.
2. **Where to start:** this instance's saved answers, a copy of another instance's, or the
   recommended defaults. Skipped for a new instance when nothing else is configured.
3. **Overall feel, AI features, privacy, HTTPS-only and languages.** Each is pre-filled; the
   languages question lists the spellcheck dictionaries found on your system.
4. **Review:** how many prefs would be written, added, changed or reset, then *Save and apply
   now*, *Save only* or *Cancel*. Apply isn't offered while Firefox is running on that profile.

`configure --accessible` asks the same questions as plain numbered prompts, for screen readers
or for piping answers in. If the input runs out before the end, nothing is saved.

For scripts, skip the wizard by passing the instance and answers as flags. Answers you don't pass
keep their saved value, or the defaults (feel `balanced`, AI `off`, privacy `strict`,
HTTPS-only on, languages from your locale):

```console
$ foxtrainer configure --profile dev-edition-default-1 --feel balanced --ai off
$ foxtrainer diff      # see exactly what would change
$ foxtrainer apply     # close Firefox first
```

| Flag | Values |
|---|---|
| `--feel` | `lean`, `balanced`, `full` |
| `--ai` | `off` (blocks every AI feature, including translations), `local` (blocks cloud AI only), `all` |
| `--privacy` | `standard`, `strict` |
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

| What | Default folder |
|---|---|
| Your answers | `~/.config/foxtrainer/config.toml` |
| Downloaded sources | `~/.cache/foxtrainer/` |
| Manifests and backups | `~/.local/state/foxtrainer/` |

`XDG_CONFIG_HOME`, `XDG_CACHE_HOME` and `XDG_STATE_HOME` are honoured. On macOS the folders are
under `~/Library`; `foxtrainer paths` lists them.
