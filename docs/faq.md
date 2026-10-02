# FAQ

> **Draft.** Questions will be added as they come up.

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
foxtrainer's language and spelling question fixes this. On Linux it points Firefox at your
system's hunspell dictionaries; elsewhere it installs the matching dictionary add-on.

### Will foxtrainer move my bookmarks between profiles?

No. foxtrainer only manages preferences. Use Firefox Sync to share data between profiles.
