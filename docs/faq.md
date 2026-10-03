# FAQ

### Why must Firefox be closed?

Firefox rewrites `prefs.js` while it runs. Changing a profile underneath a running Firefox risks
losing the changes or corrupting the file, so foxtrainer refuses and asks you to close it first.

### Can two Firefox versions share one profile?

Not safely. Firefox upgrades a profile's databases when a newer version opens it. An older
version (ESR after Developer Edition, say) will then refuse it with "You've launched an older
version of Firefox". Use one profile per install, and let foxtrainer apply the same answers to each.

### Why is my spellchecker in US English when I installed a British English language pack?

Language packs translate Firefox's menus; they don't include a dictionary. Mozilla's Linux
packages are US English builds plus a language pack, so the only built-in dictionary is en-US.
foxtrainer's language and spelling answer fixes this. On Linux it points Firefox at your
system's hunspell dictionaries (install them with e.g. `sudo apt install hunspell-en-gb`).
foxtrainer doesn't set up dictionaries on macOS; install Mozilla's dictionary add-on for your
language there. See [How it works](how-it-works.md#language-and-spelling).

### Why does `foxtrainer list` say my profile is "in a Profile Group"?

Firefox's newer profile manager groups profiles together, and for grouped profiles it keeps a few
prefs (mostly telemetry and data-reporting consent, studies and the default-browser check)
group-wide. Those group-wide values override `user.js` at startup, and foxtrainer 1.0 doesn't
write them. `diff` and `apply` list which of your prefs are affected; set them the same way in
Firefox's own settings in one profile of the group. See
[How it works](how-it-works.md#profile-groups).

### Which platforms are supported?

Linux, with pre-built binaries for amd64 and arm64. foxtrainer builds from source on macOS, but
it is untested there and doesn't yet find installs in `/Applications`. Windows is not supported.

### Is there a Hardened privacy level?

Not yet. The choices are Standard and Strict; `configure` rejects `--privacy hardened`. See
[Experience levels](experience-levels.md#privacy).

### Will foxtrainer move my bookmarks between profiles?

No. foxtrainer only manages preferences. Use Firefox Sync to share data between profiles.
