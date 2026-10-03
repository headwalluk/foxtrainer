# Sources and credits

foxtrainer builds its configuration from these sources:

| Source | Licence | How foxtrainer uses it |
|---|---|---|
| [Betterfox](https://github.com/yokoffing/Betterfox) by yokoffing, `user.js` and `Peskyfox.js` | MIT | Telemetry, performance, clutter and privacy prefs behind the feel and privacy answers |
| foxtrainer's own groups | MIT | AI controls, language and spelling, and the few prefs upstream doesn't cover (such as the rest of the sponsored-content and new tab switches) |

The catalogue in foxtrainer 1.0 pins Betterfox 154.0. `foxtrainer catalogue show` lists every
pref with its source, and `catalogue/catalogue.toml` lists the Betterfox prefs deliberately left
out, each with a reason.

[arkenfox user.js](https://github.com/arkenfox/user.js) (MIT) is credited in
THIRD-PARTY-NOTICES.md, but no arkenfox prefs are in the 1.0 catalogue, and there is no
Hardened privacy level built on it yet.

## Staying current

foxtrainer uses a fixed, reviewed version of each upstream project, never whatever was published
most recently. Moving to a newer Betterfox is a change in a foxtrainer release, and
`foxtrainer diff` shows what it would change in your profiles before you apply it. How the
versions are pinned and verified is in the [Security model](security-model.md#upstream-files).

## Not affiliated

foxtrainer is an independent project. Its output is not Betterfox or arkenfox, and problems
with foxtrainer's configuration should be reported here, not upstream. Licence texts are in
[THIRD-PARTY-NOTICES.md](../THIRD-PARTY-NOTICES.md).
