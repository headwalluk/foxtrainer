# foxtrainer

[![CI](https://github.com/headwalluk/foxtrainer/actions/workflows/ci.yml/badge.svg)](https://github.com/headwalluk/foxtrainer/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/headwalluk/foxtrainer?include_prereleases&sort=semver)](https://github.com/headwalluk/foxtrainer/releases)
[![Go version](https://img.shields.io/github/go-mod/go-version/headwalluk/foxtrainer)](go.mod)
[![Licence: MIT](https://img.shields.io/badge/licence-MIT-blue.svg)](LICENSE)
[![Platforms](https://img.shields.io/badge/platforms-Linux%20%7C%20macOS-lightgrey.svg)](docs/getting-started.md)

**Tame Firefox.** Answer a few plain questions once: how lean you want it, whether you want AI
features, how strict privacy should be, which language you spell in. foxtrainer then builds and
applies the matching configuration to every Firefox you use, and re-applies it whenever Firefox
or the upstream pref collections change.

```console
$ foxtrainer configure    # pick an instance, answer about 6 questions
$ foxtrainer apply        # rebuild user.js for every configured instance
```

> **Status: early development (0.x).** Commands are being built milestone by milestone, so
> expect breaking changes until 1.0.

## Who it's for

- **Firefox users** who want AI features, telemetry and sponsored content gone, or who want a
  sensible privacy and performance baseline, without hand-editing `about:config` after every update.
- **People who run more than one Firefox**, for example ESR for everyday browsing, Developer
  Edition for work and Nightly for testing, each with different preferences.
- **Tinkerers and security-minded users** who want to see exactly which prefs are set, where
  each one came from, and what changed since the last apply.

foxtrainer works only inside your Firefox profiles. It never needs root or administrator
rights, and it never touches your bookmarks, passwords or history.

It builds on the work of [Betterfox](https://github.com/yokoffing/Betterfox) and
[arkenfox](https://github.com/arkenfox/user.js), with credit. It is not affiliated with either
project. See [Sources and credits](docs/sources-and-credits.md).

## Documentation

- [Getting started](docs/getting-started.md): install, first run, everyday use
- [How it works](docs/how-it-works.md): instances, answers, the catalogue, and what `apply` does to a profile
- [Experience levels](docs/experience-levels.md): what Lean, Balanced and Full-fat (and each privacy level) change
- [Disable AI in Firefox](docs/disable-ai-in-firefox.md): Firefox's AI features and how foxtrainer handles them
- [Sources and credits](docs/sources-and-credits.md): where the prefs come from and how versions are pinned
- [Security model](docs/security-model.md): what foxtrainer reads, writes, downloads and verifies
- [FAQ](docs/faq.md)
- [Contributing](docs/contributing.md): building, testing and code conventions

## Author

Created by [Paul Faulkner](https://headwall-hosting.com/a-web-guys-blog/) at
[Headwall Hosting](https://headwall-hosting.com/).

## Licence

[MIT](LICENSE). Third-party notices are in [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md).
