# HANDOFF — Cross-Platform Path Resolution + Marker Path Fix

## Session: naming-001 cross-platform path resolution (OPT-3)

### What was implemented
1. **T1 — TOML paths validation**: `Paths.Executor` map (buffer, in_progress, done, notifications, archive, target) validated from TOML. Added `requiredExecutorPaths` schema validation in `config.Validate()`.
2. **T2 — PathBuilder service**: `internal/platform/pathbuilder.go` with `Native()` (host-OS separators via `filepath.Join`) and `Canonical()` (forward-slash strings via `path.Join`).
3. **T3 — PathBuilder wiring**: Added `PathBuilder` to `pipeline.Options` and `Pipeline` struct, wired from `cmd_process.go` via `platform.NewPathBuilder(destination)`.
4. **T4 — Script paths**: `pipeline/config.go` uses `path.Join` for script paths; script domain emits forward-slash paths.
5. **T5 — Metadata markers**: Generator writes and executor reads `.processing`/`.done` markers using `PathBuilder.Canonical("metadata", ...)`.
6. **T6 — Source paths**: Ingest tests verify canonical source path handling; `app/ingest` uses `filepath.Join(dir, name)` for host-OS paths.

### Bug fix: marker-path-fix
**Problem**: Generated script marker line embedded the generator's destination path (e.g. `/192.168.31.4/buffer/IN/metadata/...processing`), but the executor runs at its own buffer path (e.g. `/home/pemaltynov/IN`), so the `.processing` file did not exist at the embedded path and `mv` to `.done` failed with "No such file or directory".

**Root cause**: In `pipeline/config.go` `buildScriptConfig`, `metadataDir` was constructed as `path.Join(toSlash(root), "metadata")` using the generator's destination `root`. The script's marker line embedded this generator path, but the executor needs its metadata path derived from the executor's buffer path.

**Fix**: Changed `metadataDir` in `buildScriptConfig` to use the executor's buffer path: `path.Join(buffer, "metadata")`, where `buffer` is `toSlash(executorPaths["buffer"])` (falling back to generator root when unset). Marker lines in generated scripts now reference the executor's metadata location.

### Verification results
- **Live test confirmed**: `process --once` completed with 6 processed, 0 failed; metadata markers in canonical forward-slash form referencing the buffer path (`/home/galdoba/workspace/lazyam3/buffer/metadata/...`).
- **Unit tests pass**: PathBuilder (native/canonical), config executor validation, pipeline marker canonical form, ingest source paths, script generation.
- **Gates pass**: `go build ./...`, `go test -race -cover ./...`, `go vet ./...`, `golangci-lint run ./...` (0 issues).

### Files changed
- `internal/infra/config/config.go`, `config_test.go`: executor path schema validation.
- `internal/platform/pathbuilder.go`, `pathbuilder_test.go`: PathBuilder service.
- `internal/app/pipeline/dto.go`, `pipeline.go`, `config.go`: PathBuilder wiring, canonical markers, executor-buffer metadata dir, marker path fix.
- `internal/app/pipeline/pipeline_test.go`: PathBuilder initialization in tests.
- `internal/infra/commands/cmd_process.go`: PathBuilder construction.
- `doc/feature_list.json`: naming-001 and marker-path-fix entries status → done.
- `AGENTS.md`: Current State/Feature updated.

### Maintenance notes
- PathBuilder root defaults to destination in `NewPipeline` when unset (prevents relative `metadata/` dirs).
- Canonical forward-slash markers ensure generator (Windows) and executor (Linux) share identical path format.
- `metadataDir` in `buildScriptConfig` uses the executor buffer path so marker lines reference the executor's metadata location.
- No unrelated changes included; all 7 plan tasks (T1-T7) completed and bug fixed.

### Next agent
Maintenance phase. Cross-platform path resolution complete and marker path bug fixed. Address new issues or features as they arise per project rules.
