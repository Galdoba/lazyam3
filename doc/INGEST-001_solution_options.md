---
problem_id: INGEST-001
title: Graceful handling of files with non-standard names
generated_at: 2025-11-01T00:00:00Z
verdict: APPROVED
---

## approaches:

### Approach 1: SyntheticID constant + Synthetic flag on MediaUnit + graceful ExtractID

**Description:** Add `SyntheticID` constant (9999) and `Synthetic bool` field to `MediaUnit`. Both `ingest.ExtractID()` and `callname.ExtractID()` return `SyntheticID, nil` instead of error when filename doesn't start with a digit. `BuildInputName()` falls back to `SyntheticID` when digit extraction fails. `NewMediaUnit()` sets `Synthetic = true` when ID equals `SyntheticID`. Pipeline logs a warning when processing synthetic units.

**Files changed:**
- `internal/domain/ingest/unit.go` - add SyntheticID const (9999) and Synthetic bool to MediaUnit; NewMediaUnit sets Synthetic=true when ID==SyntheticID
- `internal/domain/ingest/extract.go` - ExtractID returns SyntheticID, nil instead of 0, error for non-digit filenames
- `internal/domain/callname/names_input.go` - ExtractID returns ingest.SyntheticID; BuildInputName falls back to ingest.SyntheticID instead of error
- `internal/app/pipeline/pipeline.go` - warn log when unit.Synthetic is true after ingest
- `internal/domain/ingest/unit_test.go` - update TestExtractID and TestNewMediaUnit for synthetic IDs
- `internal/domain/callname/names_input_test.go` - update test cases for synthetic IDs

**Pros:**
- Both ExtractID functions handle non-digit filenames gracefully (consistent behavior across ingest and callname)
- Synthetic flag on MediaUnit enables pipeline to distinguish synthetic IDs for logging
- Minimal API changes: no signature changes to BuildInputNames or BuildInputNames
- Domain purity preserved: ingest and callname remain pure, no external dependencies added
- SyntheticID constant is discoverable and documented

**Cons:**
- SyntheticID is a hardcoded constant (9999) - no automatic derivation
- Both ExtractID functions need to be updated for consistency (ingest and callname)
- Synthetic flag on MediaUnit adds a domain-level field (acceptable for logging/metrics)

**Effort:** 2-3 hours

### Approach 2: Keep ExtractID pure, handle synthetic in NewMediaUnit only

**Description:** Only modify `ingest.ExtractID()` to return SyntheticID. `callname.ExtractID()` remains unchanged (still returns error). `BuildInputName` handles the error by falling back to SyntheticID. `NewMediaUnit` sets Synthetic flag when ID equals SyntheticID.

**Files changed:**
- `internal/domain/ingest/unit.go` - add SyntheticID const and Synthetic bool
- `internal/domain/ingest/extract.go` - ExtractID returns SyntheticID, nil
- `internal/domain/callname/names_input.go` - BuildInputName falls back to SyntheticID on ExtractID error
- `internal/app/pipeline/pipeline.go` - warn log when Synthetic

**Pros:**
- Minimal changes: only ingest domain handles synthetic IDs
- callname.ExtractID remains pure

**Cons:**
- Inconsistent behavior: ingest handles gracefully, callname returns error (until BuildInputName catches it)
- More error-prone: two different code paths for synthetic ID handling

**Effort:** 1-2 hours

### Approach 3: Infra-layer synthetic ID assignment

**Description:** Keep domain layers unchanged. The infra layer (`Ingest.Ingest`) catches errors from ExtractID, constructs a MediaUnit with ID 9999, and logs a warning. `BuildInputNames` gains a `unitID int64` parameter.

**Files changed:**
- `internal/app/ingest/ingest.go` - wrap NewMediaUnit in error handling
- `internal/domain/callname/names_input.go` - BuildInputNames signature change

**Pros:**
- Domain layers remain completely pure

**Cons:**
- Infra layer needs to reconstruct MediaUnit fields manually
- More code duplication and fragility
- Synthetic ID invisible to pipeline layer

**Effort:** 3-4 hours

### Approach 2: Infra-layer synthetic ID assignment + BuildInputNames unitID parameter

**Description:** Keep `NewMediaUnit` unchanged (still returns error on non-digit IDs). The infra layer (`Ingest.Ingest`) catches the error, creates a unit with ID 9999 manually, and logs a warning. `BuildInputNames` gains a `unitID int64` parameter that overrides filename-based extraction.

**Files changed:**
- `internal/domain/callname/names_input.go` - same signature change as Approach 1
- `internal/app/ingest/ingest.go` - wrap NewMediaUnit in error handling; on ExtractID error, construct unit manually with ID 9999
- `internal/app/pipeline/config.go` - same unit.ID pass-through

**Pros:**
- Domain layers (ingest, callname) remain completely pure - no synthetic ID concept in domain
- `ExtractID` and `NewMediaUnit` retain their current semantics

**Cons:**
- Infra layer needs to reconstruct MediaUnit fields (Filenames, Streams) manually from the error path - code duplication
- Synthetic ID concept is invisible to the pipeline layer (no flag on MediaUnit)
- More fragile: infra must manually handle all MediaUnit fields

**Effort:** 3-4 hours

### Approach 3: Modify BuildInputName to fall back internally

**Description:** Instead of passing a unit-level ID, modify `BuildInputName` to fall back to synthetic ID 9999 when it cannot extract digits from the filename. No changes to `MediaUnit` or `NewMediaUnit`.

**Files changed:**
- `internal/domain/callname/names_input.go` - `BuildInputName` falls back to ID 9999 when digit extraction fails
- `internal/domain/ingest/unit.go` - `NewMediaUnit` catches ExtractID error, sets ID to 9999, does NOT return error

**Pros:**
- Simplest domain change - just a fallback in `BuildInputName`
- No new fields on MediaUnit
- Self-contained: each filename gets synthetic ID independently

**Cons:**
- Violates the "unit-level synthetic ID" requirement: each filename would independently decide whether to use synthetic ID, potentially causing inconsistent IDs within a single unit
- `NewMediaUnit` still needs to handle ExtractID errors (same change as Approach 1)
- No way to log a unit-level warning about synthetic IDs
- Less visible to the pipeline layer

**Effort:** 1-2 hours

## rationale:
**Rationale:** Approach 1 (implemented) treats synthetic ID as a unit-level concept with minimal API changes. Both ExtractID functions handle non-digit filenames gracefully, the Synthetic flag enables pipeline logging, and BuildInputName falls back to SyntheticID without requiring a signature change. Approach 2 keeps domains pure but requires manual MediaUnit reconstruction in infra, which is fragile. Approach 3 is simplest but violates unit-level consistency.

## verdict: IMPLEMENTED

**Implementation summary:** Approach 1 was implemented with both ExtractID functions (ingest and callname) modified to return SyntheticID gracefully. The Synthetic flag on MediaUnit enables pipeline logging. BuildInputName falls back to SyntheticID without requiring a signature change.

**Status:** All unit tests updated. Pipeline integration test added. Build, vet, and lint pass.

## open_questions:
- When some files in a unit have valid IDs and others don't, should the unit use the valid ID or a synthetic one? (Recommendation: use the valid ID. This is handled because NewMediaUnit sets mu.ID on first ExtractID success and skips subsequent assignments when mu.ID is already set.)
