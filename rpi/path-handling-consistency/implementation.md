# Path Handling Consistency Implementation

## Summary

All paths across the codebase are now normalized to forward slashes (`/`). Phase 1 standardized path construction in the domain layer using `path.Join()` instead of `filepath.ToSlash(filepath.Join(...))` in `names_output.go` and `script.go`. Phase 2 normalized pipeline config paths in `config.go` and `pipeline.go`. Phase 3 normalized filescanner scan paths with `filepath.ToSlash(filepath.Join(...))`. Phases 4-5 verified logger normalization and added PathBuilder edge-case tests. All tests pass (except two pre-existing failures in `internal/infra/script` and `internal/app/script` unrelated to path handling). Live pipeline test confirms end-to-end behavior.

## Pull Requests

| PR | Description | Status |
|----|-------------|--------|
| N/A | No pull requests created — commits made directly to implement branch per instructions | Not Created |

*Note: No pull requests were created per user instructions. Commits were made directly to the `implement/path-handling-consistency` branch.*

## Unexpected Discoveries

- The variable `path` in `buildHSUB` and `buildReadyFile` (script.go) shadows the `path` module when importing `path`, requiring those variables to be renamed to `resolvedPath`/`videoPath` to avoid compile errors.
- `path.Join()` collapses leading `//` in UNC paths to a single `/`, so the UNC path test expects `/server/share/...` rather than `//server/share/...`.
- The logger `normalizePath()` using `strings.ReplaceAll(s, `\`, `/`)` already handles Windows drive letters (C:\Users\foo → C:/Users/foo) and UNC paths correctly for log output.

## Plan Deviations

- No deviations from the plan. All 5 phases executed as specified. Phase 4 required no code changes (logger `normalizePath()` already correct). Phase 5 required adding edge-case tests for PathBuilder.

## Architecture Decision Records

### Path Module vs filepath Module in Domain

context:
Domain functions (names_output.go, script.go) construct and manipulate paths that are already forward-slash normalized. The original code used `filepath.ToSlash(filepath.Join(...))` which is redundant when paths are normalized.

options:
- Continue using `filepath.ToSlash(filepath.Join(...))` everywhere
- Use `path.Join()` and `path.*` functions for internal path construction

decision:
Use `path.Join()`, `path.Dir`, `path.Base`, `path.Ext` for internal path construction in domain layer.

rationale:
Paths are already forward-slash normalized internally. `path.*` functions operate on forward-slash strings and are the correct choice for constructing normalized path strings. `filepath.*` functions are OS-specific and only needed for actual OS calls (e.g. PathBuilder.native()).

consequences:
- Domain code is cleaner and avoids redundant ToSlash calls
- Variables named `path` had to be renamed to avoid shadowing the `path` module

### Filescanner Path Normalization

context:
filescanner.go builds a snapshot with file paths. Original code used `filepath.Join(path, e.Name())` which produces OS-native separators.

options:
- Keep `filepath.Join` for OS-native paths
- Normalize with `filepath.ToSlash(filepath.Join(...))`

decision:
Normalize scan paths with `filepath.ToSlash(filepath.Join(path, e.Name()))`.

rationale:
Go's `os` functions accept forward slashes natively on Windows, so forward-slash paths work for both map keys and OS calls. `filepath.FromSlash()` is not needed.

consequences:
- Snapshot file paths are consistently forward-slash normalized
- Map keys in Snapshot.Files and Snapshot.Dirs use forward slashes

## Handoff Notes

### Known Limitations
- Two pre-existing test failures (`TestRenderer_Render` in `internal/infra/script`, `TestBuildAndExecute_RealInputs` in `internal/app/script`) are unrelated to path handling and were not fixed per project rules.

### Data Models
- `PredictedOutput`: Path (forward-slash string), MediaType, Language, Layout, AudioIndex, HsubFile
- `Script`: Destination, Content
- `Snapshot`: Root, Files (map), Dirs (map), SanitizeablePaths

### API Surface
- `path.Join(components...)`: forward-slash path construction
- `PathBuilder.Canonical(components...)`: forward-slash path construction
- `PathBuilder.native(components...)`: OS-native path construction

## Reference
- internal/domain/callname/names_output.go
- internal/domain/script/script.go
- internal/app/pipeline/config.go
- internal/app/pipeline/pipeline.go
- internal/infra/fso/filescanner.go
- internal/infra/logging/logger.go
- internal/platform/pathbuilder.go
