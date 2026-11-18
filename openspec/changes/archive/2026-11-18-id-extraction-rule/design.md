# Design

## Context

See proposal.md - Why.

Three copies of the ID scan exist: `ingest.ExtractID`, `callname.ExtractID`, and an inline scan inside `callname.BuildInputName`. The first is the only one with a production caller (`ingest.resolveID`). The second has none. The third runs on every unit, per file, and bypasses the consistency check entirely. Two of them cut the stem at the first dot and one at the last.

Constraints found in the code:

- Import graph: `ingest` is a leaf, `callname` imports `ingest`, `script` imports both. Moving extraction into `callname` would create a cycle or force a new leaf package.
- The ID has exactly one consumer that needs its value: the metadata lookup in `pipeline.processUnit`. Naming needs only a nonzero check; `PredictOutputs` never reads the unit ID at all.
- Since ADR 0012 the ID is absent from output file names, and storage keys are PRT/script-name based, so identity is not used to address anything.
- `callname.BuildInputName` receives the resolved `metadata.Product`, but only `deriveOutBase` and `deriveNameTag` read it, and they read `SeriesTitle`, `SeasonNumber` and `EpisodeNumber`, never the ID.
- The identity decision currently runs over the media files a unit was built from. Subtitle sidecars and unprobeable files are filtered out earlier by the ingest adapter, and the sidecar names are appended after the decision (`internal/app/ingest/ingest.go:50-65`), so they never take part in it today.

## Goals / Non-Goals

**Goals:**

- One grammar, one extractor, one owner.
- Identity is unanimous or absent: a directory is never named after part of its own files.
- "No identity" is representable honestly, without a fabricated number.
- Delete the duplicate parsers without changing call name formats.

**Non-Goals:**

- Changing which names are candidates for the identity decision. Today's filter is kept as it is; the extractor is variadic precisely so a later change can widen the vote without touching the rule. Whatever the filter passes is, by definition, a candidate.
- Parsing tags once at ingest and passing them downstream.
- Wiring `MediaUnit.Validation` into the pipeline.
- Changing the call name or output name format.

## Decisions

**Extraction stays in ingest.** The stated migration toward `callname` is backwards: `callname` runs after the only consumer of the ID and already depends on `ingest`. Alternative considered: a new leaf package shared by both, which the cycle would require — rejected, it exists only to serve a function `callname` no longer needs.

**`0` means no identity, and `9999` is deleted.** Alternatives considered: keeping the sentinel (a fabricated number in the same shape as a catalog key, sitting in a band where the catalog is only 15 slots wide) and a nullable identity or a separate `IdentityKnown` flag (two sources of truth that can disagree). `0` is already the "not yet observed" value inside `resolveID`, so it removes a double meaning instead of adding one.

**The extractor is one variadic claim vote: `extractID(candidateNames ...string) int64`.** It returns the unanimous claim or `0`, and never an error — the current `error` return is never produced and no caller can act on it. A claim is a positive number; `0` is not claimable, so a literal `0` stem carries nothing. The candidate names are filtered before the call, so the extractor answers only "do these names agree" and never decides who is asked. Alternative considered: a name-by-name extractor plus a separate consistency walk, which is what `resolveID` does today — rejected, because the agreement is the decision and splitting it lets the two halves disagree about `0`.

**Unanimous, or nothing.** If the candidates do not all claim the same ID, the unit has no identity. Alternatives considered: a name that claims nothing abstaining while another name carries the unit (rejected — a directory whose own files disagree is exactly the case where the product cannot be named, and adopting one file's number hides the disagreement), and keeping today's hard rejection for disagreeing names (rejected — that failure is triggered by a fabricated sentinel, and a unit with readable files should reach fallback rather than abort).

**Who takes part is decided before extraction, and that is out of scope.** A name that reaches `extractID` is a candidate by definition, and a name filtered out is not. This change neither widens nor narrows the filter, so the extractor answers only whether the names it was given agree.

**Naming carries no product ID at all.** `ScriptInput.ID` is deleted along with the scan that filled it: the ID has been absent from output file names since ADR 0012, and `deriveOutBase` and `deriveNameTag` never read it. `BuildInputNames` keeps its signature because naming still needs the resolved product for its base and season tag.

**Tag and hardsub parsing stay in `callname` for this change, and they read the name as given.** They answer a different question (what rides into the output name) and moving them would drag `detectHSUB`, `findSubtitle`, and the sidecar append timing into the same change. A name that carries no identity claim therefore still contributes its tags: an unidentified sidecar keeps producing `rus`-style tags for the call name.

## Risks / Trade-offs

- A real catalog ID could collide with the unidentified value → the catalog in `testdata/metadata.json` spans 9257..64033, and the lookup for `0` is a plain miss that lands in the existing fallback branch.
- A good claim is now voided by one bad candidate (`63603.mp4` beside `no_id_in_video.mp4` loses `63603`) → that is the intended strictness: the directory is unnamed rather than named after whichever file happened to be readable. No per-name warning is wired (`MediaUnit.Validation` is a Non-Goal), so the outcome is visible as the pipeline warn plus a fallback product.
- Units that today abort now fall back (`63603.mp4` plus `99999.mp4`, `63603.mp4` plus `no_id_in_video.mp4`) → the loud failure is replaced by an unidentified unit that is still reported and still processed. The `bad-name` / `bad-name-2` live fixtures take the same route they take today, and no live fixture has candidates that disagree, so the live test outcome is unchanged.
- Removing the nonzero check could let a broken unit through → the check tested a value nothing reads; identity is still decided by lookup and fallback.
- Deleting the `ScriptInput.ID` field changes a struct that carries json tags → verified: `ScriptInput` is never marshaled anywhere, `String()` does not emit `ID`, and its only readers are `validate()` and three assertions in `names_input_test.go`.
- Two grammars coexist: strict for claims in ingest, lenient for tags in `callname` → residue from a malformed sidecar name can still appear in a call name. Accepted for this change; the follow-up is a parsed-name value owned by ingest.

## Migration Plan

No data migration. Records written before this change report `unit_id: 9999` for unidentified units; nothing reads that value, and new records report `0`. Rollback is reverting the commits — no state needs rewriting, since storage keys and tracking are PRT and script-name based.

