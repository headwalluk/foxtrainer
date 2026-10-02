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

### How installs are found

On Linux, foxtrainer looks in:

- Mozilla and distro package folders: `/usr/lib/firefox*` and `/usr/lib64/firefox*`
- Common tarball locations: `/opt/firefox*`, `~/firefox*` and `~/.local/opt/firefox*`
- Wherever a `firefox*` launcher on your `PATH` points, e.g. `/usr/bin/firefox-devedition` → `/usr/lib/firefox-devedition`
- Every install folder a profile was last run from, recorded in its `compatibility.ini`

A folder counts as an install when it has an `application.ini` with a version. The channel
(release, beta, Developer Edition, Nightly, ESR) comes from `defaults/pref/channel-prefs.js`,
falling back to `update-settings.ini`, then the source repository named in `application.ini`.

### The install hash

Firefox names each install's section in `profiles.ini` `[Install<hash>]`. The hash is
CityHash64, Mozilla's bundled v1.0, not the later v1.1. It is computed over the UTF-16LE bytes
of the folder holding the Firefox executable, with symlinks resolved, and written as uppercase
hex **without zero-padding**. So `/opt/firefox` hashes to the 15-digit `6AFDA46A1A8AD48`.

### Is Firefox running?

Firefox holds a POSIX `fcntl` write lock on `.parentlock` in the profile for as long as it runs.
foxtrainer asks the kernel whether anyone holds that lock, without taking it. The `lock` symlink
next to it (e.g. `lock -> 127.0.1.1:+3492142`) proves nothing on its own: Firefox 158 leaves it
behind even after a clean exit. foxtrainer ignores it.

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
