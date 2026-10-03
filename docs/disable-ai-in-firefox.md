# Disable AI in Firefox

This page lists Firefox's AI features, the prefs behind them, and what foxtrainer's AI answer
sets. The exact list for your answers and Firefox version is in
`foxtrainer catalogue show --ai off --firefox <version>`.

## AI Controls (Firefox 148 and later)

Firefox 148 added an **AI Controls** pane to Settings, with a "Block AI enhancements" switch
and a dropdown per feature. Behind it are these prefs:

| Pref | Feature | Since |
|---|---|---|
| `browser.ai.control.default` | The global switch: `"available"` or `"blocked"` | 148 |
| `browser.ai.control.translations` | Page translations | 148 |
| `browser.ai.control.pdfjsAltText` | Alt text for images in PDFs | 148 |
| `browser.ai.control.smartTabGroups` | Suggested tab groups and names | 148 |
| `browser.ai.control.linkPreviewKeyPoints` | Key points in link previews | 148 |
| `browser.ai.control.sidebarChatbot` | The AI chatbot sidebar | 148 |
| `browser.ai.control.smartWindow` | Smart Window | 150 |
| `browser.ai.control.speechRecognition` | Speech recognition for websites | 157 |

The per-feature prefs default to `"default"`, meaning "follow the global switch". Firefox ESR
140 has no AI Controls; only the feature prefs below apply there.

## Why the global switch alone isn't enough

Flipping the switch in Settings does more than set `browser.ai.control.default`: Firefox also
turns off each feature's own prefs and `extensions.ml.enabled`, the AI API for extensions.
Setting only `browser.ai.control.default = "blocked"` in `user.js` doesn't do those extra
steps. foxtrainer therefore sets the control prefs **and** the feature prefs, as Mozilla's own
`AIControls` enterprise policy does.

## Features and where they run

| Feature | Runs | Feature prefs foxtrainer sets |
|---|---|---|
| Machine-learning engine shared by on-device features | On your device (models are downloaded) | `browser.ml.enable` |
| AI API for extensions | On your device | `extensions.ml.enabled` |
| Translations | On your device | `browser.translations.enable` |
| PDF alt text | On your device | `pdfjs.enableAltText`, `pdfjs.enableGuessAltText`, `pdfjs.enableAltTextModelDownload` |
| Smart tab groups | On your device | `browser.tabs.groups.smart.enabled`, `browser.tabs.groups.smart.userEnabled` |
| Link preview key points | On your device | `browser.ml.linkPreview.enabled` |
| Speech recognition | On your device | `media.webspeech.recognition.enable` |
| Chatbot sidebar | In the cloud, with the provider you choose | `browser.ml.chat.enabled`, `.menu`, `.page`, `.sidebar`, `.shortcuts` |
| Smart Window | In the cloud, through Mozilla | `browser.smartwindow.enabled` (150+) |

## What each answer sets

- **Off** (`ai.block-all`): the global switch and every per-feature control set to
  `"blocked"`, and every feature pref in the table above set to `false`. Translations go too.
- **Local only** (`ai.block-cloud`): the chatbot and Smart Window controls set to `"blocked"`,
  and the `browser.ml.chat.*` prefs and `browser.smartwindow.enabled` set to `false`. On-device
  features stay as Firefox ships them.
- **Everything**: nothing is set. If you switch to it from Off or Local only, the next `apply`
  removes foxtrainer's earlier values from `prefs.js`, so Firefox's defaults return.

Prefs that a Firefox version doesn't have are left out for it (for example,
`browser.ai.control.speechRecognition` before 157). foxtrainer leaves the AI Controls pane
visible, so you can see the result in Settings. A change made there lasts only until Firefox
next starts, because `user.js` sets the value again; change your foxtrainer answer instead.

## Doing it by hand

In `about:config`, set `browser.ai.control.default` and each `browser.ai.control.*` pref above
to `blocked`, then set the feature prefs to `false`. Or put the same values in your profile's
`user.js` as `user_pref("name", value);` lines. Note that foxtrainer replaces `user.js`
entirely; see [How it works](how-it-works.md#what-apply-does-to-a-profile).

## New AI features

foxtrainer doesn't detect new AI prefs by itself. When a Firefox release adds an AI feature, it
needs a catalogue update in a new foxtrainer release; re-running `foxtrainer apply` after
upgrading picks it up. In the meantime, Mozilla says new AI features follow the global switch,
which the Off answer sets to `"blocked"`.
