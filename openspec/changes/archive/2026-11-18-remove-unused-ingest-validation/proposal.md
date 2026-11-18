# Proposal

## Why

`internal/domain/ingest` contains a media-quality validation subsystem that no production code can reach: nothing outside the package calls `ValidateMediaUnit`, and `MediaUnit.Validation` is never written by `NewMediaUnit` and never read by anything. The subsystem stays green at 97.8% coverage purely because three test suites assert against it, which hides the real fact that the pipeline learns nothing about a suspect media unit. Cleaning this up makes the package describe only what the pipeline actually uses.

## What Changes

- Remove the ingest validation subsystem: `ValidateMediaUnit`, `validateVideo`, `validateAudio`, `validateDurations`, `rationalMeanFloat`, and the `ValidationIssue` / `ValidationIssues` types (`internal/domain/ingest/unit.go:84-221`).
- **BREAKING** (internal API) Remove the `MediaUnit.Validation` field (`internal/domain/ingest/unit.go:13`). Verified: no reference to it exists outside `internal/domain/ingest`, in production or in tests.
- Remove the `TagMap` type (`internal/domain/ingest/parse.go:53-54`), which has zero references in the module.
- Remove the orphaned `// resolveID extracts product IDs …` comment left behind by the identity-extraction refactor (`internal/domain/ingest/unit.go:81-82`), and drop `buildStreams`' second return value, which its only call site discards (`internal/domain/ingest/unit.go:36,70-79`).
- Delete the three validation test suites `TestValidateMediaUnit`, `TestValidateMediaUnit_Details`, `TestValidateMediaUnit_Comprehensive` (`internal/domain/ingest/unit_test.go:162-262,551-665`) together with the code they test.
- Fix two test fixtures that write `"tag"` where ffprobe emits `"tags"` (`internal/domain/ingest/unit_test.go:85,265`), which made the interlaced assertion at `unit_test.go:130` vacuous.
- Keep every descriptive `Stream` field (`Codec`, `Width`, `Height`, `FrameRate`, `PixelFormat`, `SampleRate`, `SampleFormat`, `Forced`, `Duration`, `BitRate`) and the parsing that fills them, even though no consumer reads them yet, and record that as a deliberate decision in an ADR rather than leaving it implicit.
- Keep the helper edge-case tests (`TestParseDuration`, `TestParseRational`, `TestParseBitRate`, `TestStreamKind`, `TestDetectInterlaced`): they cover the `N/A`, `0/0`, empty and malformed fallback branches that `TestParseStreams` never exercises.

Not changing: how a unit's identity is derived, which names are candidates for that decision, how unit files are ordered, what `NewMediaUnit` accepts and returns, or any consumer of `MediaUnit`.

## Capabilities

### New Capabilities

- none

### Modified Capabilities

- none. No standing requirement changes: no spec in `openspec/specs/` mentions media-quality validation, `MediaUnit.Validation`, or any `Stream` fact being removed. This change removes code that no spec ever claimed, so it opts out of spec deltas via `skip_specs: true` in the change's `.openspec.yaml`. The durable outcome of the discussion is recorded as an ADR (`doc/adr/`), which is where the project keeps decisions, not requirements.

## Impact

- `internal/domain/ingest`: `unit.go` (validation block and `Validation` field), `parse.go` (`TagMap`), `unit.go` comment and `buildStreams` signature, `unit_test.go` (three suites). Roughly 355 lines of the package's 1353 disappear; `stream.go`, `extract.go`, `order.go`, `port.go` and the remaining tests are untouched.
- No call sites change. `internal/app/ingest`, `internal/app/pipeline`, `internal/domain/callname` and `internal/domain/script` never referenced the removed symbols, so nothing outside the package is edited.
- `doc/adr/` gains one record: descriptive stream facts that nothing is required to read, and the fact that ingest does not validate media quality. `doc/adr/INDEX.md` gains the matching row.
- Risk: if media-quality validation is wanted later it returns as its own change that wires `MediaUnit` issues into the pipeline, not as resurrected dormant code.
- Gate: `./scripts/full_test.sh` must pass, since both live pipeline runs exercise ingest end to end.
