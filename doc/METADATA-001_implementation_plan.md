# METADATA-001 Implementation Plan (Approach 1: Self-contained reload in Repository)

## Step 1: `internal/infra/metadata/repository.go`
- Add imports: `time` (and `os` for stat via `os.Stat`)
- Extend `Repository` struct with `metadataPath string` and `lastMod time.Time` fields
- Update `New(logger, path)`: after `LoadRoot`, stat the file, store `metadataPath=path` and `lastMod` from `stat.ModTime()`
- Add `Reload() (bool, error)`:
  - Read-lock, stat file; if stat fails → return `(false, err)` (fallback: keep current root)
  - If `stat.ModTime() == lastMod` → return `(false, nil)` (no change)
  - Write-lock, re-stat to avoid race, call `LoadRoot(metadataPath)`
  - On success: replace `root`, update `lastMod`, log `INFO` with old/new product counts, return `(true, nil)`
  - On failure: return `(false, err)`, root unchanged

## Step 2: `internal/infra/metadata/repository_test.go`
- `TestRepository_Reload_Success`: temp JSON → New → modify JSON (add product) → `Reload()` → assert `true` + `FindByID` returns new product
- `TestRepository_Reload_FailureFallback`: corrupt JSON → `Reload()` → assert error, repo serves old data
- `TestRepository_Reload_Untouched`: `Reload()` without changes → assert `(false, nil)`
- Run with `go test -race`

## Step 3: `internal/infra/commands/cmd_process.go`
- No change required — line 85 already passes `metadataPath` to `metadata.New(modules.Logger, metadataPath)`

## Step 4: `internal/app/pipeline/dto.go` + `pipeline.go`
- **dto.go**: change `Metadata metadata.ProductRepository` → `Metadata *infra.metadata.Repository` in `Options`
- **pipeline.go**:
  - Import infra metadata (alias `mdinfra`), change `Pipeline.md` field to `*mdinfra.Repository`
  - In `Run()` loop (line 100), before `p.RunOnce(ctx)`:
    ```go
    if changed, err := p.md.Reload(); err == nil && changed {
        p.logger.Info("metadata reloaded", "products", p.md.Count())
    } else if err != nil {
        p.logger.Warn("metadata reload failed", "error", err)
    }
    ```
  - `NewPipeline` keeps `md: opts.Metadata` assignment unchanged
