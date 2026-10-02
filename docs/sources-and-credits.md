# Sources and credits

> **Draft.** The source list will be finalised with the first catalogue release.

foxtrainer builds its configuration from these sources:

| Source | Licence | How foxtrainer uses it |
|---|---|---|
| [Betterfox](https://github.com/yokoffing/Betterfox) by yokoffing | MIT | Performance, clutter and privacy prefs behind the Lean, Balanced and Full-fat levels |
| [arkenfox user.js](https://github.com/arkenfox/user.js) | MIT | Additional hardening prefs behind the Hardened privacy level |
| Mozilla Firefox source (`StaticPrefList.yaml`, `firefox.js`, …) | MPL-2.0 | Checks only: does a pref still exist in a given Firefox version |
| foxtrainer's own groups | MIT | AI controls, language and spelling, and anything upstream doesn't cover |

## Pinning

Each upstream file is pinned by **full git commit SHA** and **sha256**. foxtrainer never pulls an
upstream project's latest commit straight into your browser. Moving to a new upstream version is
a reviewed catalogue change, and `apply` tells you what it changed.

## Not affiliated

foxtrainer is an independent project. Its output is not Betterfox or arkenfox, and problems
with foxtrainer's configuration should be reported here, not upstream. Licence texts are in
[THIRD-PARTY-NOTICES.md](../THIRD-PARTY-NOTICES.md).
