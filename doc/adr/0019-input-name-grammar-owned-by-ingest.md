---
id: "0019-input-name-grammar-owned-by-ingest"
title: "Input Name Grammar Owned By Ingest"
description: "Incoming file names are read for a product identity in exactly one place, inside ingest, with one accepted grammar and a unanimous-or-nothing vote. An unresolved identity is the value 0 rather than a placeholder number, and naming no longer derives an identity from a name."
status: accepted
date: "2026-11-18"
scope: cross-package
---

# Input Name Grammar Owned By Ingest

## Context

Three copies of the leading-digit scan existed: `ingest.ExtractID`, `callname.ExtractID`, and an inline scan inside `callname.BuildInputName`. Two of them cut the stem at the first dot and one at the last, so the same name could yield different claims depending on which code read it. Only `ingest` had a production caller. The identity it produced was checked for consistency across a unit's files, and an unresolved identity was reported as `SyntheticID` (`9999`) — a fabricated number in the same shape as a catalog key.

The import graph decided the owner: `ingest` is a leaf, `callname` imports `ingest`, and `script` imports both. The only consumer of the identity value is the metadata lookup in `pipeline.processUnit`; naming reads the resolved product's title and season/episode numbers and never its ID.

## Decision

**One extractor, owned by ingest.** `extractID(candidateNames ...string) int64` is package-private to `ingest`. It is the only code that reads a product identity out of an incoming name.

**One accepted grammar.** A name claims an identity only when it matches `<digits> [ '-' tag [ '_' tag ]* ] [ '.' extension ]`, meaning the leading digit run is terminated by `-`, `.`, or the end of the name. `12254rus.srt` and `63604_1080.mp4` claim nothing.

**Unanimous, or nothing.** Every candidate name the extractor is given must claim the same ID. If any candidate claims nothing, or the claims differ, the unit has no identity and no single candidate's claim is adopted. The extractor returns a value and never an error.

**`0` is reserved for "no identity", and `9999` is deleted.** `0` is not a claimable value, so a literal `0` stem claims nothing, and no component downstream can receive a fabricated ID that looks like a catalog key. `MediaUnit.Synthetic` now simply means `ID == 0`.

**Who takes part is decided before extraction.** The candidate names are selected by the ingest adapter (media files only; unprobeable and sidecar files are appended after the decision). The extractor never admits or excludes a name from the set it is given, so the candidate set can change later without changing the rule.

**Naming carries no product identity.** `ScriptInput.ID`, `callname.ExtractID`, and the inline scan in `BuildInputName` were deleted. Naming uses the identity the pipeline resolved and works unchanged for a product whose identity is `0`. Tag parsing stays in `callname` and reads the name as given, so a name with no identity claim still contributes its `rus`-style tags to the call name.

## Rationale

The stated migration of extraction into `callname` was backwards: `callname` runs after the only consumer of the identity and already depends on `ingest`. Identity is a fact about the incoming directory, established before anything is named, so the stage that inspects the directory owns the grammar. Splitting the vote into a per-name extractor plus a separate consistency walk is what the old `resolveID` did, and the two halves could disagree about the meaning of `0`.

This decision continues the series that removed identity from names: ADR 0012 dropped `_idN` from output file names and ADR 0013 dropped the `prodID` parameter from `PredictOutputs`. With the identity gone from both naming families, a second parser of it in `callname` had no consumer left.

## Consequences

- A directory holding `63603.mp4` beside `no_id_in_video.mp4` is now unnamed rather than named after the one readable file, and a directory holding `63603.mp4` beside `99999.mp4` falls back instead of aborting with an inconsistent-IDs error. Both are reported as unidentified: a pipeline warn, a fallback product named after the source directory, a processing record and a completion event carrying identity `0`.
- The metadata lookup for `0` is a plain miss, so unidentified units land in the existing fallback branch. The catalog in the test ground spans 9257..64033, so `0` cannot collide with a real product there.
- Two grammars coexist: strict for claims in `ingest`, lenient for tags in `callname`. Residue from a malformed sidecar name can still reach a call name. The follow-up is a parsed-name value owned by ingest.
- Records written before this change report `unit_id: 9999` for unidentified units; nothing reads that value, and new records report `0`. No data migration is needed, because storage keys and tracking are PRT and script-name based.

## Options

- **Keep the `9999` sentinel.** Rejected: a placeholder in the same shape as a catalog key, sitting in an unused band of the ID space, and it silently turns "unknown" into a lookup for a product number.
- **A nullable identity or a separate `IdentityKnown` flag.** Rejected: two sources of truth that can disagree about the same unit.
- **An unclaimed candidate abstaining while another carries the unit.** Rejected: a directory whose own files disagree is exactly the case where the product cannot be named, and adopting one file's number hides the disagreement.
- **Keep the hard rejection for disagreeing names.** Rejected: that failure was triggered by the fabricated sentinel, and a unit with readable files should reach fallback rather than abort.
- **A new leaf package shared by `ingest` and `callname`.** Rejected: the import cycle would require it only to serve a function `callname` no longer needs.
