# Architecture

How foxtrainer finds Firefox, decides what to write and writes it safely. This is for people
working on the code; [How it works](how-it-works.md) is the user-level version, and
[The catalogue](catalogue.md) covers the pref data. The package map is in
[Contributing](contributing.md#project-layout).

## Start-up

`internal/config` is the only package that reads the environment. It resolves every folder
through `internal/paths` (foxtrainer's config, cache and state folders, the Firefox profile roots
and the install search patterns) into one typed `Config`, and reports every problem at once.
Everything else receives that struct.

## Finding instances

`internal/firefox` builds an `Inventory` from the profile roots and the installs it can find.
Problems are collected as warnings rather than stopping discovery.

### Profile roots

| Platform | Profile root |
|---|---|
| Linux | `~/.mozilla/firefox`, or `~/.config/mozilla/firefox` (Firefox 147+, only when `~/.mozilla` does not exist) |
| macOS | `~/Library/Application Support/Firefox` |

All Firefox channels share one root per user. In each root:

- **`profiles.ini`** is the source of truth for profiles. `[ProfileN]` sections are read from 0
  and stop at the first gap, as Firefox does; later ones are reported as ignored. Sections that
  Firefox ignores (such as bare install hashes copied from `installs.ini`) are kept as stray
  sections and shown by `list`.
- **`[Install<hash>]`** sections link an install to its default profile (`Default=`).
- Each profile's **`compatibility.ini`** records the version and install folder
  (`LastPlatformDir`) that last ran it.

### Installs

On Linux, foxtrainer looks in:

- Mozilla and distribution package folders: `/usr/lib/firefox*` and `/usr/lib64/firefox*`
- Common tarball locations: `/opt/firefox*`, `~/firefox*` and `~/.local/opt/firefox*`
- Wherever a `firefox*` launcher on `PATH` points, e.g. `/usr/bin/firefox-devedition` →
  `/usr/lib/firefox-devedition`
- Every `LastPlatformDir` from a profile's `compatibility.ini`, except Snap and Flatpak sandbox
  paths

A folder counts as an install when it has an `application.ini` with a version. The channel
(release, beta, Developer Edition, Nightly, ESR) comes from `defaults/pref/channel-prefs.js`,
falling back to `update-settings.ini`, then the source repository named in `application.ini`.

An **instance** is an install plus a profile that the install either has as its default or last
ran. A profile is an **orphan** when the `LastPlatformDir` that last ran it no longer exists
(Snap and Flatpak sandbox paths excepted). An `[Install<hash>]` section whose hash matches no
live install is a **dead install section**; `list` reports it under Notes.

### The install hash

Firefox names each install's section `[Install<hash>]`. The hash is CityHash64, Mozilla's
bundled v1.0, not the later v1.1 (`internal/cityhash`). It is computed over the UTF-16LE bytes of
the folder holding the Firefox executable, with symlinks resolved, and written as uppercase hex
**without zero-padding**. So `/opt/firefox` hashes to the 15-digit `6AFDA46A1A8AD48`.

### Is Firefox running?

Firefox holds a POSIX `fcntl` write lock on `.parentlock` in the profile for as long as it runs.
`list` and `diff` probe it with `F_GETLK`, which reports the holder's PID without taking the
lock. The `lock` symlink next to it (e.g. `lock -> 127.0.1.1:+3492142`) proves nothing:
Firefox 158 leaves it behind even after a clean exit, so it is logged at debug level only.

`apply` takes the lock itself (`AcquireLock`) and holds it for the whole write. POSIX locks
belong to the process, not the descriptor: closing **any** descriptor on `.parentlock`, even
one opened just to read it, drops the lock. While the lock is held, nothing may open that file
again.

## Profile Groups

Firefox's newer profile manager can put profiles in a Profile Group. Only one `[ProfileN]`
section represents the group: it carries `StoreID=<id>`, and its `Path` is rewritten to
whichever member was used last. The other members are only in the group's datastore,
`<root>/Profile Groups/<storeID>.sqlite`. A `.sqlite` file on its own doesn't mean a group is in
use; `StoreID=` in `profiles.ini` does.

Firefox keeps the prefs in its `permanentSharedPrefs` list in that store's `SharedPrefs` table,
and at startup writes the stored values over whatever `user.js` set. The Firefox 158 list is in
`internal/firefox/sharedprefs.go` (`IsGroupSharedPref`); it changes between Firefox versions.

foxtrainer 1.0 doesn't write the store. For an instance whose profile has a `StoreID`:

- `apply.Target.ProfileGroupID` carries the ID, and `Prepare` puts the planned prefs that are
  group-wide into `Prepared.GroupShared`, with a note.
- The CLI logs a warning, and `list` marks the instance.
- `Commit` records `profile_group_id` and `group_shared` in the manifest.

foxtrainer never writes `browser.profiles.*` or `toolkit.profiles.*`, which record group
membership; the catalogue checker rejects them.

## Applying

`internal/apply` works in two stages, so `diff` and `apply` share one code path.

**`Prepare`** reads but never writes:

1. Resolves the answers against the catalogue for the instance's Firefox major version and
   platform (see [The catalogue](catalogue.md#resolving-answers)).
2. Drops `extensions.enabledScopes` when user- or system-scope extension folders are non-empty,
   since it would disable those extensions.
3. Renders `user.js`: a header (foxtrainer and catalogue versions, the instance, the answers, each
   upstream source with its licence), then one block per group, each pref commented with its
   origin. The output is deterministic.
4. Compares it with the current `user.js` (added, changed, removed) and lists the prefs to reset:
   names in the manifest that are no longer planned. If the current `user.js` wasn't written by
   foxtrainer, its prefs become **leftovers**, reset only with `--reset-previous`.

**`Commit`** takes the profile lock, then:

1. Refuses if `user.js` is a symlink, since an atomic replace would turn it into a plain file.
2. Backs up `user.js` and `prefs.js` into `<state>/backups/<instance>/<timestamp>/`, pruning to
   the last 10. Skipped when nothing changes.
3. Writes `user.js` atomically (temporary file, then rename) with mode 0600.
4. Removes the reset names from `prefs.js` with the `internal/prefs` tokeniser, preserving every
   other byte and the file's permissions.
5. Saves the manifest, then releases the lock.

### The manifest

`<state>/instances/<profile folder>-<hash>.toml` records what was last applied:

| Field | Meaning |
|---|---|
| `schema_version` | Manifest layout; an unknown version is an error |
| `install_path`, `profile_path` | The instance |
| `catalogue_version`, `applied_at`, `user_js_sha256` | What was written, and when |
| `managed` | Every pref name foxtrainer currently owns in this profile. If a `prefs.js` reset fails, the names stay here so the next apply retries |
| `profile_group_id`, `group_shared` | For a grouped profile: the store ID and the managed prefs the group store may override |

`managed` is the only thing that lets foxtrainer touch `prefs.js`. A pref name that isn't in it
is never removed.

## Upstream files

`internal/sources` downloads each pinned upstream file over HTTPS from
`raw.githubusercontent.com` at the pinned commit, checks it against the catalogue's sha256 and
caches it under `<cache>/sources/<source>/<commit>/`. Cached copies are re-verified before every
use; a mismatch is an error and is never cached. `--offline` uses the cache only.
