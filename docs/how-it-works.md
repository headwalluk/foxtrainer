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

The catalogue lives in [`catalogue/`](../catalogue) and is compiled into the binary:

- `catalogue.toml` pins each upstream source (full commit SHA plus a sha256 per file), records
  the licence and copyright for attribution, and lists upstream prefs deliberately left out,
  each with a reason.
- `groups/*.toml` has one group per file. A group has a title and description, a `when` table
  saying which answers switch it on (e.g. `when = { feel = ["lean", "balanced"] }`), and its prefs:
  - `[[include]]`: every active pref in an upstream section or subsection, minus an optional
    `exclude` list;
  - `[[pick]]`: named prefs from an upstream file, including optional (commented-out) ones;
  - `[prefs]`: values given directly, optionally limited by Firefox version
    (`{ value = "blocked", since = 150 }`) or platform.
- A group with `generator = "…"` has prefs computed in code, such as language and spelling,
  which depends on your chosen languages and the dictionaries on your system.

Values from upstream are read from the pinned upstream files at apply time, so the configuration
really is built from Betterfox, with each pref traced to its file and line.

### Rules the catalogue checker enforces

`foxtrainer catalogue check` (also run by the test suite) rejects a catalogue unless:

- **Every active pref in Betterfox's `user.js` is used by exactly one group or excluded with a
  reason.** When the Betterfox pin moves, any new or changed pref fails the check until someone
  decides where it belongs.
- **No two groups that can be active together set the same pref to different values,** unless
  one explicitly declares `overrides = ["other.group"]`.
- **Nothing writes `browser.profiles.*` or `toolkit.profiles.*`.** Those prefs record
  profile-group membership, and a wrong value can move a profile into another group.
- A pick names a pref that appears exactly once, so Betterfox's "alternative" values can't be
  picked by accident, and every `when` answer, source, file and section exists.

`foxtrainer catalogue show --feel balanced --ai off --firefox 158` prints exactly which prefs a
set of answers produces, and where each one came from.

## Language and spelling

Mozilla's own Linux builds (the tarball and the apt packages) are US-English builds. A language
pack translates the menus but brings no dictionary, so spellchecking stays en-US even with,
say, `firefox-devedition-l10n-en-gb` installed. (Mozilla's en-GB tarball differs: it bundles an
en-GB dictionary inside its `omni.ja`.)

foxtrainer's `language.spelling` group is generated from your chosen languages (`--languages
en-GB,en`, defaulting to your locale):

- `intl.accept_languages` tells websites which languages you prefer.
- On Linux, foxtrainer looks for each language's hunspell dictionary (`en_GB.dic` and
  `en_GB.aff`) in `/usr/share/hunspell` and `/usr/share/myspell/dicts`. It points
  `spellchecker.dictionary_path` at that folder and selects the dictionaries it found with
  `spellchecker.dictionary` (e.g. `en-GB`, or `en-GB,fr` for several).
- A bare language next to a regional one (`en` beside `en-GB`) needs no dictionary of its own.
- When a dictionary is missing, `diff` and `apply` name the package, e.g.
  `sudo apt install hunspell-de-de`.

macOS and Windows have no system hunspell dictionaries. There, foxtrainer will install Mozilla's
dictionary add-on for the language instead (coming with macOS and Windows support).

## What `apply` does to a profile

For every configured instance, `apply` first works everything out without writing (this is
also exactly what `foxtrainer diff` and `apply --dry-run` show), then:

1. **Refuses if Firefox is using the profile,** and otherwise takes Firefox's own profile lock
   for the whole apply, so Firefox can't start on the profile part-way through.
2. **Backs up** `user.js` and `prefs.js` to the state folder, keeping the last 10 backups per
   instance. Nothing is backed up or written when there is nothing to change.
3. **Writes `user.js`:** a header naming the instance, your answers, the catalogue version and
   each upstream source with its licence, then one commented block per group. Each pref is
   annotated with where its value came from (e.g. `// betterfox user.js:123`). The output is
   deterministic, so re-applying unchanged answers leaves the file alone. A symlinked `user.js`
   is never replaced.
4. **Resets prefs foxtrainer no longer sets.** Firefox copies `user.js` values into `prefs.js`
   and keeps them there, even after they leave `user.js`. foxtrainer records every pref it
   writes in a per-instance manifest, and removes from `prefs.js` exactly the ones it set before
   but no longer sets, so Firefox's default comes back. Prefs foxtrainer never set are left alone.
5. **Replacing a `user.js` foxtrainer didn't write** (e.g. a hand-installed Betterfox) backs it
   up first. The prefs it had set stay in `prefs.js` unless you add `--reset-previous`.

If user- or system-wide extensions are installed (e.g. Debian's `webext-*` packages), the strict
privacy pref `extensions.enabledScopes` is left out, because it would disable them.

### Where foxtrainer keeps its state

| What | Linux (default) |
|---|---|
| Your answers | `~/.config/foxtrainer/config.toml` |
| Per-instance manifests | `~/.local/state/foxtrainer/instances/<profile>-<id>.toml` |
| Backups | `~/.local/state/foxtrainer/backups/<profile>-<id>/<timestamp>/` |
| Downloaded upstream files | `~/.cache/foxtrainer/sources/<source>/<commit>/` |
