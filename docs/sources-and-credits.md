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

Files are downloaded from `raw.githubusercontent.com` at the pinned commit and cached in
foxtrainer's cache folder (`foxtrainer paths`). A cached copy is re-verified before every use.
Content that doesn't match its sha256 is rejected and never cached.

## How Betterfox files are read

Betterfox's files are written for people, so foxtrainer reads them line by line:

- `SECTION:` banners (e.g. `SECUREFOX`) and, in `user.js`, `/** NAME ***/` subsections
  (e.g. `TELEMETRY`) give each pref its place.
- `user_pref("name", value);` is an **active** pref: Betterfox sets it. `//user_pref(...)` is a
  **commented-out** pref: optional, or documenting a value Firefox already uses (marked `DEFAULT`).
- Values are bool, 32-bit int or string, in single or double quotes with backslash escapes.
  Lines that look like prefs but don't parse (Betterfox's guides have a few) are recorded as
  malformed and can never be picked.

`user.js` is treated as Betterfox's recommendation; the guide files (such as `Peskyfox.js`)
are a pool of optional prefs that groups may pick from.

## Not affiliated

foxtrainer is an independent project. Its output is not Betterfox or arkenfox, and problems
with foxtrainer's configuration should be reported here, not upstream. Licence texts are in
[THIRD-PARTY-NOTICES.md](../THIRD-PARTY-NOTICES.md).
