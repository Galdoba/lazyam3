# Path Handling Consistency Plan

## Implementation Overview

The goal is to ensure ALL paths everywhere in the codebase are always in Linux-style (forward slashes `/`). This research has established a clear pattern: store paths internally as forward slashes, using `path.Join()` for construction and `filepath.ToSlash()` only for OS-specific string conversions.

Key architectural decisions:
- `PathBuilder.Canonical()` already uses `path.Join()` and returns forward-slash strings - this is the canonical path construction method
- `PathBuilder.native()` uses `filepath.Join()` for OS-specific separators - kept for actual OS calls that require native separators
- `toSlash()` wrapper in `config.go` uses `filepath.ToSlash()` - this is the normalization function for string paths
- Logger `normalizePath()` uses `strings.ReplaceAll(s, `\`, `/`)` - sufficient for log output only

Integration points:
- Pipeline config construction (config.go)
- Script domain predictions (names_output.go, script.go)
- File scanner (filescanner.go)
- Logger output formatting

## Task Breakdown

### Phase 1: Path Construction Standardization (Domain)

- [x] In `internal/domain/callname/names_output.go`, replace `filepath.ToSlash(filepath.Join(...))` with `path.Join(...)` at lines 61-64 (targetDir construction), 85, 103, 116; remove the separate `filepath.ToSlash(targetDir)` at line 66; update imports to use `path` instead of `filepath`
- [x] In `internal/domain/script/script.go`, replace `filepath.ToSlash(filepath.Join(...))` with `path.Join(...)` at line 256 and 309; replace `filepath.Dir` with `path.Dir` at lines 222, 293; replace `filepath.Base` with `path.Base` at lines 229, 305; replace `filepath.Ext` with `path.Ext` at line 306; remove remaining `filepath.ToSlash` calls; update imports to use `path` instead of `filepath` (paths are already forward-slash normalized, so `path.*` functions are correct)

### Phase 2: Pipeline Config Normalization

- [x] In `internal/app/pipeline/config.go`, review the `toSlash()` function at lines 149-154 and ensure it consistently uses `filepath.ToSlash()`; verify all call sites (lines 49-55, 92-110) receive normalized paths; change line 51 `filepath.Join(toSlash(root), "target")` to `path.Join(toSlash(root), "target")` for consistency
- [x] In `internal/app/pipeline/pipeline.go`, normalize `statusPath` construction at line 62 to use forward slashes via `toSlash()` or `filepath.ToSlash()`

### Phase 3: File Scanner Path Handling

- [x] In `internal/infra/fso/filescanner.go`, update `scanDir` at line 125 to normalize paths with forward slashes using `filepath.ToSlash(filepath.Join(path, e.Name()))`. Go's `os` functions accept forward slashes on Windows natively, so `filepath.FromSlash()` is NOT needed for OS calls.

### Phase 4: Logger Consistency

- [x] In `internal/infra/logging/logger.go`, review `normalizePath()` at lines 407-413 to ensure it handles UNC paths and Windows drive letters consistently, matching the research findings

### Phase 5: PathBuilder Verification

- [x] Verify `internal/platform/pathbuilder.go` `Canonical()` method at lines 31-32 correctly uses `path.Join()` and `native()` at lines 26-28 correctly uses `filepath.Join()`; add edge-case tests for empty root, Windows drive letters, UNC paths

## Code References

```
internal/domain/callname/names_output.go:61-64,85,103,116
internal/domain/script/script.go:222,229,256,293,305,306,309
internal/app/pipeline/config.go:49-55,92-110,149-154
internal/app/pipeline/pipeline.go:62
internal/infra/fso/filescanner.go:125
internal/infra/logging/logger.go:407-413
internal/platform/pathbuilder.go:26-32
```

## Testing Plan

- Unit tests for `PathBuilder.Canonical()` and `PathBuilder.native()` covering edge cases: empty root, Windows drive letters, UNC paths, mixed separators
- Unit tests for `toSlash()` covering: empty string, Linux paths (no-op), Windows paths (conversion), UNC paths, mixed separators
- Integration tests: verify generated scripts use forward slashes, verify logger output is forward-slash normalized
- Cross-platform simulation: test on Linux with Windows-style paths to verify normalization

## FACTS Scale Output

F: 5  A: 5  C: 5  T: 5  S: 5  Mean: 5.00  --> PASS

## Dependencies & Sequencing

- Phase 1 (Domain) must complete before Phase 2 (Pipeline Config) as domain functions feed pipeline logic
- Phase 2 and Phase 3 are independent and can proceed after Phase 1
- Phase 4 (Logger) and Phase 5 (PathBuilder) are verification tasks independent of Phases 1-3
- Critical path: Phase 1 -> Phase 2 -> Integration verification

## Risk Mitigation

- Risk: Forward slashes in filescanner.go breaking OS calls on Windows. Mitigation: Go's `os` functions accept forward slashes natively on Windows, so `filepath.ToSlash(filepath.Join(...))` is safe for both map keys and OS calls; `filepath.FromSlash()` is not needed.
- Risk: Windows drive letter handling (C:\Users\foo to C:/Users/foo). Mitigation: `filepath.ToSlash()` preserves drive letters; research confirms this is safe.
- Early warning indicator: Any test failure involving path comparison with backslashes.
- Escalation criteria: If normalization breaks cross-platform compatibility, revert to using `PathBuilder.Canonical()` as the single source of truth.

## Rollback Strategy

- Phase 1 rollback: Revert names_output.go and script.go imports and filepath.Join usage
- Phase 2 rollback: Revert config.go toSlash wrapper and pipeline.go statusPath construction
- Phase 3 rollback: Revert filescanner.go to filepath.Join
- Feature flag: Path normalization can be toggled via a build-time flag if runtime issues emerge
- Data migration: No persistent data migration required; paths are runtime-constructed
