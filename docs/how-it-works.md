# How it works

What foxtrainer does to your Firefox, in plain terms. For the internals, see
[Architecture](architecture.md).

## Instances

An **instance** is one Firefox install plus one profile. For example, Developer Edition from
`/usr/lib/firefox-devedition` with its `dev-edition-default` profile. Each instance has its own
answers, so ESR for everyday browsing and Nightly for testing can be configured differently.

`foxtrainer list` shows every instance it finds, whether Firefox is running on it, and two
kinds of leftovers:

- **Orphaned profiles:** profiles last used by a Firefox install that no longer exists. That
  happens, for example, after switching from a tarball to a distribution package.
- **Other profiles:** profiles that were never started, whose folder is missing, or that were
  last used by an install foxtrainer doesn't recognise.

foxtrainer finds Firefox installed from Mozilla's apt repository or a distribution package, and
Mozilla tarballs in `/opt` or your home folder. A tarball unpacked anywhere else is found once
Firefox has been run from it. Snap and Flatpak Firefox aren't found yet.

## Answers, not prefs

foxtrainer saves your **answers** ("AI: off", "Privacy: strict"), not a list of prefs. Each
`apply` turns the answers into prefs using the current catalogue: the list of pref groups built
into foxtrainer, drawn from [Betterfox](sources-and-credits.md) and foxtrainer's own groups. So
when a new foxtrainer release adds prefs for a new Firefox feature, or follows an upstream
change, re-applying picks it up.

[Experience levels](experience-levels.md) describes what each answer changes.
`foxtrainer catalogue show` lists the exact prefs for a set of answers, with where each value
came from.

## What `apply` does to a profile

`apply` works through every configured instance. First it works out what would change, without
writing anything. That's also exactly what `foxtrainer diff` (or `apply --dry-run`) shows, and
those work while Firefox is running. Then it:

1. **Refuses if Firefox is using the profile.** Close Firefox first. foxtrainer holds Firefox's
   own profile lock while it writes, so Firefox can't start on the profile part-way through.
2. **Backs up** `user.js` and `prefs.js`, keeping the last 10 backups per instance. Nothing is
   backed up or written when there is nothing to change.
3. **Writes `user.js`:** a header naming the instance, your answers and the sources used, then
   one commented block per group, with each pref labelled with where its value came from.
   Firefox reads `user.js` every time it starts. Re-applying unchanged answers leaves the file
   alone.
4. **Resets prefs foxtrainer no longer sets.** Firefox copies `user.js` values into `prefs.js`
   and keeps them there, even after they leave `user.js`. foxtrainer remembers every pref it has
   written, and removes from `prefs.js` exactly the ones it set before but no longer sets, so
   Firefox's default comes back. Prefs foxtrainer never set are left alone.

If you already had a `user.js` that foxtrainer didn't write (a hand-installed Betterfox, say),
the first `apply` backs it up and replaces it. The prefs it had set stay in `prefs.js` unless you
add `--reset-previous`.

If extensions are installed for all users (Debian's `webext-*` packages, for example), the Strict
privacy pref `extensions.enabledScopes` is left out, because it would disable them. `diff` and
`apply` say so in a note.

A change you make in Firefox's settings to a pref foxtrainer manages lasts only until Firefox next
starts, because `user.js` sets it again. Change your answers with `foxtrainer configure` instead.

## Profile Groups

Firefox's newer profile manager can put several profiles in a **Profile Group**. For a grouped
profile, Firefox keeps a few prefs the same across the whole group: mostly telemetry and
data-reporting consent, studies and the default-browser check. Those group-wide values win over
`user.js`, and foxtrainer 1.0 doesn't change them.

So for a grouped profile:

- `foxtrainer list` marks it "in a Profile Group".
- `diff` and `apply` warn, and name which of your prefs are group-wide.

To make those settings stick, change them in Firefox's own settings in one profile of the group
(for example, turn off data collection under Privacy & Security). Firefox copies the change to
the rest of the group.

## Language and spelling

Mozilla's own Linux builds (the tarball and the apt packages) are US English builds. A language
pack translates the menus but brings no dictionary, so spellchecking stays US English even with,
say, `firefox-devedition-l10n-en-gb` installed.

foxtrainer's languages answer (defaulting to your system locale) fixes this:

- Websites are told which languages you prefer.
- On Linux, foxtrainer points Firefox at your system's hunspell dictionaries and selects the ones
  for your languages. You can have several, e.g. `en-GB,fr`.
- When a dictionary is missing, `diff` and `apply` name the package to install, e.g.
  `sudo apt install hunspell-de-de`.

It doesn't change the language of Firefox's menus. On macOS, foxtrainer only sets the website
languages; install Mozilla's dictionary add-on for your language by hand.

## Where foxtrainer keeps its files

| What | Linux (default) |
|---|---|
| Your answers | `~/.config/foxtrainer/config.toml` |
| What was applied to each instance | `~/.local/state/foxtrainer/instances/` |
| Backups | `~/.local/state/foxtrainer/backups/` |
| Downloaded upstream files | `~/.cache/foxtrainer/sources/` |

`XDG_CONFIG_HOME`, `XDG_CACHE_HOME` and `XDG_STATE_HOME` are honoured. `foxtrainer paths`
prints the folders in use.

foxtrainer never changes `profiles.ini`, `installs.ini`, your bookmarks, history, passwords or
any other browsing data. See the [Security model](security-model.md).
