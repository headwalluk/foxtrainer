# How it works

> **Draft.** This describes the design; details firm up as each milestone lands.

## Instances

An **instance** is one Firefox install plus one profile. For example, Developer Edition from
`/usr/lib/firefox-devedition` with its `dev-edition-default` profile. Each instance has its own
answers, so ESR for everyday browsing and Nightly for testing can be configured differently.

foxtrainer finds instances by reading Firefox's own `profiles.ini` and `installs.ini`, and each
profile's `compatibility.ini`. Firefox links an install to its default profile with a hash of the
install folder's path. foxtrainer computes the same hash, so it can tell which profile belongs to
which install and spot **orphaned** profiles whose install has gone. That happens, for example,
after switching from a tarball to a distro package.

### Where Firefox keeps profiles

| Platform | Profile root |
|---|---|
| Linux | `~/.mozilla/firefox`, or `~/.config/mozilla/firefox` (Firefox 147+, only when `~/.mozilla` does not exist) |
| macOS | `~/Library/Application Support/Firefox` |

All Firefox channels share one root per user; `installs.ini` records which profile each install uses.

## Answers, not prefs

foxtrainer saves your **answers** ("AI: off", "Privacy: strict"), not a list of prefs. Each
`apply` turns the answers into prefs using the current catalogue, so when Firefox adds a new AI
feature, or an upstream project changes a recommendation, re-applying picks it up.

## The catalogue

The catalogue maps answers to **groups**, and groups to prefs. Groups come from upstream projects
(Betterfox, arkenfox) at a pinned version, plus groups foxtrainer maintains itself (such as AI
controls and language and spelling). See [Sources and credits](sources-and-credits.md).

## What `apply` does to a profile

1. Refuses to continue if Firefox is using the profile, and locks it for the duration.
2. Backs up `user.js` and `prefs.js`.
3. Writes a new `user.js` containing the prefs for your answers, with a header listing sources and versions.
4. Removes from `prefs.js` any pref that foxtrainer set previously but no longer manages. Firefox
   keeps old values there otherwise. Prefs foxtrainer never set are left alone.
5. Records what it wrote in a per-instance manifest.
