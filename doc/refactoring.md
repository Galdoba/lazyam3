# Ingest Package Refactoring Plan

> **Status:** completed on branch `refactor/ingest-clean`  
> **Target package:** `internal/domain/ingest`  
> **Governing skill:** `golang-refactoring` (Plan mode gate)

---

## 0. Rationale

The ingest domain is the entry point of the pipeline: it turns raw physical files into a structured, validated `MediaUnit`. The package currently works, but several functions have grown beyond a single responsibility, making them hard to test in isolation and to reason about. Refactoring will improve testability and readability **without changing observable behavior**.

Per the project rules:
- Domains are pure — no imports from infra or app.
- Infrastructure implements domain ports (`Organizer`).
- Every fixed bug / change MUST produce a test.
- Structural and behavioral changes must never be mixed in one commit.

---

## 1. Refactoring Inventory (ordered by risk)

| # | File | Function | Current Issue | Proposed Transform | Risk |
|---|------|----------|---------------|--------------------|------|
| 1 | `extract.go` | `ExtractID` | Convoluted stem parsing + digit loop; `fmt.Sscanf` best-effort parsing | Extract stem via `strings.Cut`/`strings.TrimStart`; keep `fmt.Sscanf` but wrap in a small helper that documents best-effort semantics | Low |
| 2 | `unit.go` | `ValidateMediaUnit` | 93-line function mixing video/audio/duration/duplicate checks | Extract `validateVideo`, `validateAudio`, `validateDurations`; keep public signature | Medium |
| 3 | `parse.go` | `parseStreams` | Repetitive `if stream.Kind == ...` blocks for video/audio/subtitle fields; has `// TODO: rework` | Extract stream-type-specific field assignment into private helpers | Medium |
| 4 | `unit.go` | `NewMediaUnit` | Mixed responsibilities: JSON parse, sort, skip-non-media, build streams, ID resolution, consistency check in one loop | Split into `buildStreams`, `collectMediaFiles`, `resolveID` | Medium-High |

---

## 2. Detailed Plan per Function

### 2.1 `extract.go` — `ExtractID` (Low risk)

**What it does now:** Strips the extension by scanning backwards for a `.`, then checks if the stem starts with a digit; if so, collects leading digits and parses them with `fmt.Sscanf`.

**Why refactor:** The stem-stripping loop (lines 12-19) is convoluted and hard to read. The digit-collection loop (lines 26-32) is fine but the whole function could be clearer.

**Proposed change:**
- Replace the manual extension strip with `strings.SplitN(filename, ".", 2)[0]` (or `path.Base` if paths are involved).
- Keep the digit-extraction logic but consider `strings.TrimStart` or a small helper.
- **Behavior preserved:** returns `SyntheticID` (9999) for non-digit-prefixed stems; parses leading digits only.

**Safety net:** Existing `unit_test.go` / `order_test.go` cover `ExtractID` via `NewMediaUnit`. No new tests needed unless behavior changes.

---

### 2.2 `unit.go` — `ValidateMediaUnit` (Medium risk)

**What it does now:** A single 93-line function that:
1. Detects duplicate coordinates.
2. Validates video streams (resolution, duration, bitrate).
3. Validates audio streams (duration, bitrate).
4. Aggregates durations and checks cross-file consistency (>10% variance).

**Why refactor:** The function mixes four distinct validation categories, making it hard to test each rule in isolation and to add new rules later.

**Proposed change:** Extract into three private helpers:
- `validateVideo(s Stream, i int, issues ValidationIssues) ValidationIssues`
- `validateAudio(s Stream, i int, issues ValidationIssues) ValidationIssues`
- `validateDurations(durations []Rational) ValidationIssues`

The public `ValidateMediaUnit` becomes a thin orchestrator that calls these helpers and returns the combined `issues`.

**Safety net:** Add targeted tests for each helper (resolution threshold, bitrate thresholds, duration consistency, duplicate coordinate) before touching the code.

---

### 2.3 `parse.go` — `parseStreams` (Medium risk)

**What it does now:** Iterates `input.Streams`, builds a `Stream` DTO, and uses three `if stream.Kind == ...` blocks to populate video/audio/subtitle-specific fields. Has a `// TODO: rework` comment noting it should accept bytes from format and multiple streams.

**Why refactor:** The repetitive if-blocks are a code smell; each stream kind has its own field-assignment logic that should be localized.

**Proposed change:** Extract private helpers:
- `applyVideoFields(s StreamJSON, dst *Stream)`
- `applyAudioFields(s StreamJSON, dst *Stream)`
- `applySubtitleFields(s StreamJSON, dst *Stream)`

**Important:** The `// TODO: rework` is **not** addressed in this refactor. The domain currently only parses `streams`, not `format`. Addressing the TODO would be a behavioral change and requires infrastructure coordination — it is explicitly **out of scope** for this refactoring pass.

**Safety net:** Existing tests in `parseStreams` coverage (via `NewMediaUnit` tests) must remain green.

---

### 2.4 `unit.go` — `NewMediaUnit` (Medium-High risk)

**What it does now (lines 30-72):**
1. Parses probe JSON for all files (`parseStreams`).
2. Sorts files by rank (`sortUnitFiles`).
3. Skips files with no media streams.
4. Appends filenames, assigns `FileIndex`, populates `Streams` map.
5. Extracts IDs, validates consistency across files, sets `Synthetic` flag.

**Why refactor:** This function has five distinct responsibilities in one loop. A failure mid-loop returns an empty `MediaUnit{}`. It is hard to test individual phases (e.g. "does file ranking work independently of ID resolution?").

**Proposed change:** Split into:
- `buildStreams(parsed map[string][]Stream) (map[string]Stream, int)` — assigns `FileIndex`, builds coordinate map, returns file count.
- `collectMediaFiles(parsed map[string][]Stream) []string` — returns sorted, filtered filenames.
- `resolveID(filenames []string, streams map[string]Stream) (int64, bool, error)` — extracts IDs, checks consistency, sets synthetic flag.

`NewMediaUnit` becomes a clear sequence: parse → collect → build → resolve.

**Safety net:** Full safety net required (per High risk tier). Characterization tests for `NewMediaUnit` covering: multi-file units, data-only filtering, inconsistent IDs, synthetic ID assignment.

---

## 3. Execution Order (Plan mode gate)

Per the `golang-refactoring` workflow, changes land as small, independently reviewable PRs on a `refactor/ingest-clean` branch, with human approval between steps:

1. **Step 1 (Low):** `ExtractID` string parsing cleanup. Verify: `go build ./... && go vet ./... && go test ./...`. Commit: purely structural.
2. **Step 2 (Medium):** Extract `ValidateMediaUnit` helpers. Add targeted tests first. Commit: purely structural.
3. **Step 3 (Medium):** Extract `parseStreams` field helpers. Verify behavior preserved. Commit: purely structural.
4. **Step 4 (Medium-High):** Split `NewMediaUnit`. Add safety-net tests. Commit: purely structural.

**Cross-reference checks after any rename:** grep for struct tags, `json`/`ffprobe` field references, and reflection usage to ensure no desync.

**Security/safety pass:** Steps 1-3 are behavior-preserving (tool-verified or pure extraction), so no security pass required. Step 4 touches logic flow but preserves behavior — run `golang-safety` review before landing.

---

## 4. Sign-Off Gates (to be asked via question tool)

1. **Scope approval:** Proceed with all 4 steps, or a subset? Recommend order 1→4.
2. **Safety-net test for `NewMediaUnit`:** Should I add characterization tests for multi-file units, data-only filtering, and inconsistent IDs before refactoring step 4?
3. **TODO handling:** Confirm that the `// TODO: rework` in `parseStreams` (ffprobe format-level parsing) is explicitly **out of scope** and should not be addressed.
