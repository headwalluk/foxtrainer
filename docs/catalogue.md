# The catalogue

The catalogue is the data that turns answers into prefs. This page is for anyone changing it:
adding a group, adjusting what an answer does, or moving an upstream pin. For how the rest of
foxtrainer uses the result, see [Architecture](architecture.md).

The catalogue lives in [`catalogue/`](../catalogue) and is compiled into the binary.

## Files

### `catalogue.toml`

- `schema_version` (the layout this build understands) and `catalogue_version` (shown in every
  generated `user.js` header and in the manifest).
- `[sources.<id>]`: each upstream source with its repository, licence, copyright, tag, full
  40-character `commit`, a `raw_url` template and the sha256 of every file used
  (`[sources.<id>.files]`). `coverage` names the files whose active prefs must all be accounted
  for (see the checker rules below).
- `[[excluded]]`: upstream prefs deliberately left out, each with a `reason`.

### `groups/*.toml`

One group per file:

```toml
id          = "feel.new-tab"
title       = "A quiet new tab page"
description = "No stories, no weather and no shortcuts on the new tab page; just the search box."
when        = { feel = ["lean", "balanced"] }

[[pick]]
source = "betterfox"
file   = "user.js"
prefs  = ["browser.newtabpage.activity-stream.feeds.section.topstories"]

[prefs]
"browser.newtabpage.activity-stream.feeds.topsites" = false
```

- `when` says which answers switch the group on. Every condition must match; a group with no
  `when` is always on.
- `[[include]]` takes every **active** pref in an upstream section or subsection, minus an
  optional `exclude` list.
- `[[pick]]` takes named prefs from an upstream file, including commented-out (optional) ones.
- `[prefs]` gives values directly. A value can be limited by Firefox version or platform:
  `{ value = "blocked", since = 150 }`, `until = …`, `platforms = ["linux"]`.
- `overrides = ["other.group"]` lets this group replace another active group's values.
- `generator = "…"` marks a group whose prefs are computed in code, such as `language.spelling`
  (`internal/language`), which depends on the chosen languages and the dictionaries installed.

Values are bool, 32-bit int or string. Floats and out-of-range ints are rejected.

## Resolving answers

`catalogue.Resolve` turns a set of answers into a plan for one Firefox major version and
platform:

1. Every group whose `when` matches is expanded: includes, then picks, then `[prefs]`, or the
   generator for a generated group. Upstream values are read from the pinned files, so each pref
   keeps its origin (e.g. `betterfox user.js:145`, or `foxtrainer` for the catalogue's own
   values).
2. Prefs outside the target's version or platform limits are skipped.
3. Prefs that an active group `overrides` are removed from the overridden group.
4. If two groups set the same pref to the same value, the first group's copy is kept.

`foxtrainer catalogue show --feel balanced --ai off --firefox 158` prints the plan, with each
pref's origin.

## Rules the checker enforces

`foxtrainer catalogue check` (also run by the test suite and CI) rejects a catalogue unless:

- **Every active pref in a `coverage` file is used by exactly one group, or excluded with a
  reason.** When the Betterfox pin moves, any new or changed pref fails the check until someone
  decides where it belongs.
- **No two groups that can be active together set the same pref to different values,** unless
  one explicitly declares `overrides`.
- **Nothing writes `browser.profiles.*` or `toolkit.profiles.*`.** Those prefs record Profile
  Group membership, and a wrong value can move a profile into another group.
- A pick names a pref that appears exactly once in its file, so Betterfox's "alternative" values
  can't be picked by accident.
- Every `when` answer, source, file and section exists.
- Every source is pinned by a full commit SHA and a sha256 per file, with its licence and
  copyright recorded.

## How Betterfox files are read

Betterfox's files are written for people, so `internal/sources/betterfox` reads them line by
line:

- `SECTION:` banners (e.g. `SECUREFOX`) and, in `user.js`, `/** NAME ***/` subsections
  (e.g. `TELEMETRY`) give each pref its place, which `[[include]]` refers to.
- `user_pref("name", value);` is an **active** pref: Betterfox sets it. `//user_pref(...)` is a
  **commented-out** pref: optional, or documenting a value Firefox already uses (marked
  `DEFAULT`).
- Values are bool, 32-bit int or string, in single or double quotes with backslash escapes.
  Lines that look like prefs but don't parse (Betterfox's guides have a few) are recorded as
  malformed and can never be picked.

`user.js` is treated as Betterfox's recommendation. The guide files (such as `Peskyfox.js`) are
a pool of optional prefs that groups may pick from.

## Making a change

1. Edit `catalogue/groups/*.toml` or `catalogue.toml`.
2. Run `foxtrainer catalogue check` and `make test`. Both must pass.
3. Use `foxtrainer catalogue show` with different answers to see the effect, and `make e2e` to
   confirm a real Firefox honours it.
4. Bump `catalogue_version`.

Check new pref names against the Firefox version you target: a misspelt pref is silently
ignored by Firefox.

### Moving an upstream pin

1. Update `tag`, `commit` and the sha256 of each file in `catalogue.toml`.
2. Copy the new files into `testdata/upstream/<source>/<commit>/` for the tests.
3. Run `foxtrainer catalogue check` and resolve what it reports. That's usually new prefs needing
   a home, or an exclusion.
4. Run `foxtrainer diff` against a test profile to review what changes for users.
