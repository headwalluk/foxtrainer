# foxtrainer

[![CI](https://github.com/headwalluk/foxtrainer/actions/workflows/ci.yml/badge.svg)](https://github.com/headwalluk/foxtrainer/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/headwalluk/foxtrainer?sort=semver)](https://github.com/headwalluk/foxtrainer/releases)
[![Go version](https://img.shields.io/github/go-mod/go-version/headwalluk/foxtrainer)](go.mod)
[![Licence: MIT](https://img.shields.io/badge/licence-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux-lightgrey.svg)](docs/getting-started.md)

**Tame Firefox.** Answer a few plain questions once: how lean you want it, whether you want AI
features, how strict privacy should be, which language you spell in. foxtrainer then builds and
applies the matching configuration to every Firefox you use, and re-applies it whenever Firefox
or the upstream pref collections change.

```console
$ foxtrainer configure    # pick an instance, answer about 6 questions
$ foxtrainer apply        # rebuild user.js for every configured instance
```

## Install

On Linux (x86-64 or ARM64), install the latest release into `~/.local/bin`:

```console
$ curl -fsSL https://raw.githubusercontent.com/headwalluk/foxtrainer/main/install.sh | sh
```

The script checks the download against the release's `SHA256SUMS`. For a manual download, see
[Getting started](docs/getting-started.md#install); to build it yourself (or for macOS), see
[Compiling from source](docs/compiling.md).

## Who it's for

- **Firefox users** who want AI features, telemetry and sponsored content gone, or who want a
  sensible privacy and performance baseline, without hand-editing `about:config` after every update.
- **People who run more than one Firefox**, for example ESR for everyday browsing, Developer
  Edition for work and Nightly for testing, each with different preferences.
- **Tinkerers and security-minded users** who want to see exactly which prefs are set, where
  each one came from, and what changed since the last apply.

foxtrainer works only inside your Firefox profiles. It never needs root or administrator
rights, and it never touches your bookmarks, passwords or history.

It builds on the work of [Betterfox](https://github.com/yokoffing/Betterfox), with credit, and
is not affiliated with it. See [Sources and credits](docs/sources-and-credits.md).

## Documentation

**Using foxtrainer**

- [Getting started](docs/getting-started.md): install, first run, everyday use
- [Experience levels](docs/experience-levels.md): what each answer changes in Firefox
- [How it works](docs/how-it-works.md): instances, answers, and what `apply` does to a profile
- [Disable AI in Firefox](docs/disable-ai-in-firefox.md): Firefox's AI features and the prefs behind them
- [FAQ](docs/faq.md)
- [Sources and credits](docs/sources-and-credits.md): where the prefs come from

**Building it yourself**

- [Compiling from source](docs/compiling.md): installing Go, `go install`, building a checkout, macOS

**Reviewing its security**

- [Security model](docs/security-model.md): what foxtrainer reads, writes, downloads and verifies

**Working on foxtrainer**

- [Contributing](docs/contributing.md): setup, project layout, conventions, testing, releasing
- [Architecture](docs/architecture.md): install discovery, the profile lock, Profile Groups, the apply engine
- [The catalogue](docs/catalogue.md): the pref data, the checker rules, moving an upstream pin

## Author

Created by [Paul Faulkner](https://headwall-hosting.com/a-web-guys-blog/) at
[Headwall Hosting](https://headwall-hosting.com/).

## Licence

[MIT](LICENSE). Third-party notices are in [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md).
