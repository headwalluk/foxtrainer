# Experience levels

What each answer in `foxtrainer configure` changes in Firefox. Each answer switches groups of
prefs on or off; the group names below (such as `feel.quiet-ui`) are the headings you'll see in
the `user.js` foxtrainer writes. To see the exact prefs for a set of answers, and where each value
comes from, run for example:

```console
$ foxtrainer catalogue show --feel lean --ai local --privacy standard --firefox 158
```

The defaults for a new instance are Balanced, AI off, Strict privacy, HTTPS-Only on, and the
languages from your locale.

## Always on

Whatever you answer, these groups apply:

| Group | What it does |
|---|---|
| `baseline.telemetry` | Turns off telemetry |
| `baseline.experiments` | Turns off studies and remote experiments |
| `baseline.crash-reports` | Turns off automatic crash reports |
| `baseline.sponsored` | No sponsored stories, shortcuts or suggestions |
| `baseline.mozilla-nags` | No extension and feature recommendations, "More from Mozilla" page or guided tours |
| `feel.performance` | Betterfox's caching, rendering and networking tweaks |
| `privacy.essentials` | Global Privacy Control, safer TLS and certificate handling, punycode for look-alike domains, no scripting in PDFs |
| `language.spelling` | Your languages and spellcheck dictionaries (see below) |

## Overall feel

| Level | Intent | Adds |
|---|---|---|
| **Lean** | Strip Firefox back: no What's New page, fewer URL bar extras | `feel.quiet-ui`, `feel.new-tab`, `feel.lean-extras` |
| **Balanced** | Quiet and tidy, everything useful kept | `feel.quiet-ui`, `feel.new-tab` |
| **Full-fat** | Keep all the features; just no telemetry, ads or nags | Nothing beyond the always-on groups |

- `feel.quiet-ui`: no welcome or default-browser prompts, no notification requests from sites,
  no full-screen animation, and a tidier URL bar.
- `feel.new-tab`: no stories, weather or shortcuts on the new tab page; just the search box.
- `feel.lean-extras`: no What's New page after updates, fewer URL bar suggestions (trending,
  add-ons, MDN, Wikipedia, recent searches), no new tab wallpapers or highlights, and the full
  address in the URL bar (`https://` and `www.` are not hidden).

## AI features

| Choice | Intent | Adds |
|---|---|---|
| **Off** | Block every AI feature, on-device and cloud, including translations | `ai.block-all` |
| **Local only** | Keep on-device features such as translations; block the chatbot sidebar and Smart Window | `ai.block-cloud` |
| **Everything** | Leave Firefox's AI features as Firefox ships them | Nothing |

See [Disable AI in Firefox](disable-ai-in-firefox.md) for the prefs involved.

## Privacy

| Level | Intent | Adds |
|---|---|---|
| **Standard** | Firefox's own protections, plus the always-on essentials | Nothing |
| **Strict** | Strict tracking protection, nothing cached to disk, no speculative connections, no search suggestions or form history, and a private geolocation provider. Occasionally a site needs an exception | `privacy.strict` |

There is no Hardened level yet.

## Other questions

- **HTTPS-only** (`https.only`): refuse insecure `http://` connections unless you allow a site.
  Firefox Sync syncs this setting between devices.
- **Languages, preferred first:** sent to websites as `intl.accept_languages`, and on Linux used
  to pick system spellcheck dictionaries. It does not change the language of Firefox's menus.
  See [How it works](how-it-works.md#language-and-spelling).
