# P2 Strategy: Path Management Between Generator and Executor

## Problem Statement

The generator (lazyam3 CLI) and the executor (bash script) run on different machines with different filesystem namespaces. The generator writes scripts that the executor runs, but paths must be in the executor's format. Additionally:

- Scripts are named by unit ID (unreliable) instead of by OUTBASE
- PRIORITY is hardcoded to 0 instead of reading from config
- Windows paths use `\` instead of `/`
- Audio output lines are incomplete (missing `-map_metadata -1 -map_chapters -1`)
- Ready file notification is architecturally outdated

## Architecture Decision: Generator vs Executor Paths

### Config Structure

```go
type Paths struct {
    Generator GeneratorPaths `toml:"generator"`
    Executor  map[string]string `toml:"executor"`
}

type GeneratorPaths struct {
    InputDirectory  string `toml:"input_directory"`   // where MediaUnits live (generator's filesystem)
    OutputDirectory string `toml:"output_directory"`  // where scripts are written (generator's filesystem)
}
```

TOML (section format):

```toml
[paths.generator]
input_directory  = "/home/user/media"
output_directory = "/home/user/scripts"

[paths.executor]
buffer           = "/mnt/pemaltynov/IN/_BUFFER"
processing_dir   = "/mnt/pemaltynov/IN/_IN_PROGRESS"
target_root_dir  = "/mnt/pemaltynov/ROOT/EDIT"
done_dir         = "/mnt/pemaltynov/IN/_DONE"
```

### Two namespaces, clear separation

| Namespace | Paths | Who sets | Who accesses |
|---|---|---|---|
| **Generator** | `InputDirectory`, `OutputDirectory` | User (TOML) | Generator (reads/writes) |
| **Executor** | `buffer`, `processing_dir`, `target_root_dir`, `done_dir`, ... | User (TOML) | Executor (runs script) |

The generator does NOT access executor paths. It reads them from config and writes them as literals into the script. The executor accesses executor paths on its own filesystem.

### Path normalization

All paths written into scripts must use forward slashes `/`. If the generator runs on Windows, apply `filepath.ToSlash()` to all paths before embedding in the script template.

## Script Generation Strategy

### Principle: compute everything in Go, nothing in bash

The script should contain NO path concatenation. All paths are computed in Go and embedded as literals:

```bash
# GOOD: all paths are literals, computed by Go
ffmpeg ... "/mnt/.../Arliss_s01e06_HD.mp4" \
  -map "[aud0]" ... "/mnt/.../Arliss_s01e06_HD_AUDIO00_RUS20.m4a"

# BAD: paths computed in bash
TARGET_DIR="${ROOT}/${BASENAME}"
ffmpeg ... "${TARGET_DIR}/${OUTBASE}_HD.mp4"
```

### Script filename: by OUTBASE, not by ID

Current: `<root>/scripts/12254.sh` (by unit ID)
Target: `<root>/scripts/Arliss_s01e06.sh` (by OUTBASE)

The OUTBASE is extracted from predictions in `script.go:deriveNamesFromPredictions()`.

### ffmpeg command: single invocation, full paths

The ffmpeg command must be a single invocation with both video and audio outputs. Full absolute paths must be used (not `${TARGET_DIR}/...` variables).

## Implementation Details

### 1. Config struct reorganization

**File: `internal/infra/config/config.go`**

Change `Paths` from flat struct to two-section:

```go
// Before (current):
type Paths struct {
    InputDirectory   string `toml:"input_directory"`
    OutputDirectory  string `toml:"output_directory"`
    MetadataPath     string `toml:"metadata_path"`
    FfprobeBinary    string `toml:"ffprobe_binary,omitempty"`
    SourceRootIn     string `json:"source_root_in"`
    TargetRootDir    string `json:"target_root_dir"`
    Buffer           string `json:"buffer"`
    ProcessingDir    string `json:"processing_dir"`
    DoneDir          string `json:"done_dir"`
    ScriptArchiveDir string `json:"script_archive_dir"`
}

// After:
type Paths struct {
    Generator GeneratorPaths `toml:"generator"`
    Executor  map[string]string `toml:"executor"`
    Storage   Storage        `toml:"storage"`  // stays
    Logger    Logging        `toml:"logger"`   // stays
    Hardsub   Hardsub        `toml:"hardsub"`  // stays
}

type GeneratorPaths struct {
    InputDirectory  string `toml:"input_directory"`
    OutputDirectory string `toml:"output_directory"`
    MetadataPath    string `toml:"metadata_path"`
    FfprobeBinary   string `toml:"ffprobe_binary,omitempty"`
}
```

Default values for `GeneratorPaths` stay the same as current defaults.

**Why**: `InputDirectory`, `OutputDirectory`, `MetadataPath`, `FfprobeBinary` are generator paths. Everything else (`Buffer`, `TargetRootDir`, etc.) are executor paths that live in a different namespace.

### 2. Pipeline config.go: use executor paths from config

**File: `internal/app/pipeline/config.go`**

**Current `buildScriptConfig()` (lines 77-86):**

```go
Env: script.Environment{
    Buffer:        filepath.Join(root, "buffer"),
    InProgress:    filepath.Join(root, "in_progress"),
    Done:          filepath.Join(root, "done"),
    Notifications: filepath.Join(root, "notifications"),
    TargetRoot:    filepath.Join(root, "target"),
    Priority:      defaultPriority,
},
Destination: filepath.Join(root, "scripts", fmt.Sprintf("%d.sh", unit.ID)),
```

**Problem**: Paths are derived by convention under `root` (generator's root). This assumes generator and executor share the same filesystem namespace.

**Target:**

```go
Env: script.Environment{
    Buffer:        cfg.Env["buffer"],           // from executor paths map
    InProgress:    cfg.Env["processing_dir"],   // from executor paths map
    Done:          cfg.Env["done_dir"],         // from executor paths map
    Notifications: cfg.Env["notifications"],     // optional, from executor paths map
    TargetRoot:    cfg.Env["target_root_dir"],  // from executor paths map
    Priority:      defaultPriority,
},
Destination: filepath.Join(root, "scripts", fmt.Sprintf("%s.sh", outBase)),
```

**Why**: Paths come from executor config, not derived from generator's root. Each executor path maps to an Environment field.

### 3. Script template: remove variables, use full literal paths

**File: `internal/domain/script/template.go`**

**Current template (key lines):**

```go
`TARGET_DIR="|=target_root=|/${BASENAME}${SEASON_DIR}"`,
`BUFFER="|=buffer=|"`,
`IN_PROGRESS="|=in_progress=|"`,
`ffmpeg -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \`,
`  -filter_complex "[0:v:0]${HSUB}${YADIF}setsar=(1/1)[vidHD]|=audio_filter=|" \`,
`|=output_files=|`,
```

**Target template (key lines):**

```go
`SOURCE="|=source=|"`,
`YADIF="|=yadif=|"`,
`HSUB="|=hsub=|"`,
`ffmpeg -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \`,
`  -filter_complex "[0:v:0]${HSUB}${YADIF}setsar=(1/1)[vidHD]|=audio_filter=|" \`,
`|=output_files=|`,
```

**Changes:**
- Remove `TARGET_DIR`, `BASENAME`, `NAME_TAG`, `SEASON_DIR` lines (no longer needed in script)
- Keep `SOURCE`, `YADIF`, `HSUB` (script-time values)
- `TARGET_DIR` is no longer a variable — removed from template
- `BUFFER`, `IN_PROGRESS`, `DONE`, `NOTIFICATIONS` stay as variables (script uses them for `mv`/`touch`)

**Why**: All output paths are computed in Go and embedded as literals in `|=output_files=|`. The script has no path concatenation logic.

### 4. Audio line: add missing flags

**File: `internal/domain/script/audio.go`**

**Current `buildOutputFilesPredictions()` (line 139):**

```go
line := fmt.Sprintf(`  -map "[aud%d]" -c:a alac -compression_level 0 "%s"`, p.AudioIndex, p.Path)
```

**Target:**

```go
line := fmt.Sprintf(`  -map "[aud%d]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "%s"`, p.AudioIndex, p.Path)
```

**Why**: The working script (`working_script_example.txt`, line 42) includes `-map_metadata -1 -map_chapters -1` on the audio line. Without it, ffmpeg may copy unwanted metadata from the input.

### 5. Video line: ensure `-profile high` and `-g 0`

**Current `buildOutputFilesPredictions()` (line 136):**

```go
line := fmt.Sprintf(`  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile:v high -g 0 -map_metadata -1 -map_chapters -1 "%s"`, p.Path)
```

**Verified**: Video line already has `-profile:v high` and `-g 0`. ✅ No change needed.

### 6. OUTBASE extraction for script filename

**File: `internal/app/pipeline/config.go`**

After predictions are computed, extract OUTBASE from the first prediction's filename:

```go
// predictions come from callname.PredictOutputs()
outBase := extractOutBase(predictions[0].Path)
// outBase = filename without extension, e.g. "Arliss_s01e06"

func extractOutBase(path string) string {
    base := filepath.Base(path)        // "Arliss_s01e06_HD.mp4"
    if i := strings.Index(base, "_HD"); i > 0 {
        base = base[:i]
    }
    return base
}
```

**Why**: `callname.PredictOutputs()` already computes full output paths. The OUTBASE is the filename stem before `_HD`. This is the same logic used in `script.go:deriveNamesFromPredictions()` (lines 133-142).

### 7. Data flow: predictions drive everything

**Current flow:**

```
cmd_process.go
  → pipeline.NewPipeline(destination=OutputDirectory)
    → buildScriptConfig(root=destination)
      → callname.PredictOutputs(targetRoot=filepath.Join(root, "target"))
        → predictions with paths like "/tmp/lazyam3/output/target/Show_s01/Show_s01e01_HD.mp4"
      → BuildScript(predictions)
        → template fills placeholders → "${TARGET_DIR}/${OUTBASE}_HD.mp4"
```

**Problem**: Predictions use generator's root (`/tmp/lazyam3/output/...`). When executor runs on a different machine, these paths don't exist.

**Target flow:**

```
cmd_process.go
  → pipeline.NewPipeline(
      destination=OutputDirectory,
      executorPaths=Paths.Executor,  // NEW: pass executor map
    )
    → buildScriptConfig(root=destination, executorPaths=Paths.Executor)
      → targetRoot := executorPaths["target_root_dir"]  // e.g. "/mnt/ROOT/EDIT"
      → callname.PredictOutputs(targetRoot=targetRoot)
        → predictions with paths like "/mnt/ROOT/EDIT/Show_s01/Show_s01e01_HD.mp4"
      → BuildScript(predictions)
        → template fills output_files with FULL literal paths
```

**Why**: Predictions are computed against executor's target root (from config), not generator's root. The ffmpeg output paths in the script will be paths the executor can actually access.

## Implementation Checklist

### 1. Config struct reorganization

- [ ] Split `Paths` into `GeneratorPaths` (fixed struct) and `ExecutorPaths` (`map[string]string`)
- [ ] Move `InputDirectory`, `OutputDirectory`, `MetadataPath`, `FfprobeBinary` into `GeneratorPaths`
- [ ] `Executor` is `map[string]string` — user fills it in TOML
- [ ] Update default values in `defaultCFG()`
- [ ] Update `config_test.go`

### 2. Pipeline: use executor paths

- [ ] Pass `Paths.Executor` from `cmd_process.go` → `pipeline.Options` → `buildScriptConfig()`
- [ ] In `buildScriptConfig()`, build `Environment` from executor map keys:
  - `cfg.Env["buffer"]` → `Env.Buffer`
  - `cfg.Env["processing_dir"]` → `Env.InProgress`
  - `cfg.Env["done_dir"]` → `Env.Done`
  - `cfg.Env["notifications"]` → `Env.Notifications`
  - `cfg.Env["target_root_dir"]` → `Env.TargetRoot`
- [ ] Compute predictions using `targetRoot = cfg.Env["target_root_dir"]` (not `filepath.Join(root, "target")`)
- [ ] Extract OUTBASE from predictions for script filename
- [ ] Change `Destination` from `%d.sh` to `%s.sh` using OUTBASE
- [ ] Remove `EnsureEnvDirs()` call from `cmd_process.go` (executor paths are external)

### 3. Script template changes

- [ ] Remove `TARGET_DIR`, `BASENAME`, `NAME_TAG`, `SEASON_DIR` from `AmediaGeneric` template
- [ ] Keep `SOURCE`, `YADIF`, `HSUB` (script-time values)
- [ ] Keep `BUFFER`, `IN_PROGRESS`, `DONE`, `NOTIFICATIONS` (used by `mv`/`touch`)
- [ ] Remove `|=target_root=|` placeholder — no longer needed
- [ ] Remove `|=priority=|` → replace with direct value in template

### 4. Audio line fix

- [ ] In `buildOutputFilesPredictions()` (audio.go:139): add `-map_metadata -1 -map_chapters -1` to audio line

### 5. Path normalization utility

- [ ] Create utility: `ToSlash(path string) string` — wraps `filepath.ToSlash()`
- [ ] Apply to all executor paths before embedding in script

### 6. Deferred items

- [ ] Remove notification `cp` line from template (Telegram bot unreliable)
- [ ] Remove YADIF parameter (no-op for progressive video)
- [ ] Remove `|=srt_section=|`, `|=srt_cleanup=|`, `|=hsub_move=|`, `|=hsub_cleanup=|` if unused

## Key Files to Modify

| File | Change |
|---|---|
| `internal/infra/config/config.go` | Restructure `Paths` → `GeneratorPaths` + `Executor map[string]string` |
| `internal/infra/commands/cmd_process.go` | Pass executor paths to pipeline, remove `EnsureEnvDirs` |
| `internal/app/pipeline/config.go` | Use executor paths for Environment, compute predictions with executor target root, extract OUTBASE for filename |
| `internal/domain/script/template.go` | Remove TARGET_DIR/BASENAME/NAME_TAG/SEASON_DIR, keep only SOURCE/YADIF/HSUB + env vars |
| `internal/domain/script/audio.go` | Add `-map_metadata -1 -map_chapters -1` to audio line |
| `internal/domain/script/script.go` | Remove template vars for BASENAME/NAME_TAG/SEASON_DIR/TARGET_ROOT/PRIORITY |
| `internal/infra/config/config_test.go` | Update tests for new config structure |

## Working Script Reference

See `testdata/examples/working_script_example.txt` for a reference of a successfully executed script. Key differences from current template:

1. Full paths in ffmpeg output (not `${TARGET_DIR}/...`)
2. `-map_metadata -1 -map_chapters -1` on audio line (line 42)
3. `-profile high -g 0` on video line (line 41)
4. Single ffmpeg invocation (video + audio on same command)
5. Audio filter in same `-filter_complex` as video (separated by `;`)
6. All paths are absolute literals, no concatenation

## Why This Approach

1. **Predictions as source of truth**: `callname.PredictOutputs()` already computes full output paths. We just need to feed it the executor's target root instead of the generator's derived root.

2. **No bash string manipulation**: The script becomes a linear sequence of commands. No `${BASENAME}${NAME_TAG}` concatenation, no `${TARGET_DIR}/${OUTBASE}` — everything is pre-computed.

3. **Config-driven executor paths**: User specifies executor paths in TOML. Generator reads them and embeds them literally. No path conversion needed if user provides correct format.

4. **Windows support**: `filepath.ToSlash()` on all paths handles `\` → `/` conversion.

5. **Extensible executor paths**: `map[string]string` allows adding/removing executor paths without code changes.

## Definition of Done

### Config Changes
- [ ] `internal/infra/config/config.go`: `Paths` split into `GeneratorPaths` struct + `Executor map[string]string`
- [ ] `InputDirectory`, `OutputDirectory`, `MetadataPath`, `FfprobeBinary` moved to `GeneratorPaths`
- [ ] Default values updated in `defaultCFG()` to preserve backward compatibility
- [ ] `config_test.go` updated for new structure

### Pipeline Changes
- [ ] `cmd_process.go`: Pass executor paths via new `pipeline.Options.ExecutorPaths` field
- [ ] `pipeline/config.go`: `buildScriptConfig()` reads from executor map keys (`buffer`, `processing_dir`, `done_dir`, `notifications`, `target_root_dir`)
- [ ] Predictions computed using executor's `target_root_dir` (not generator-relative `filepath.Join(root, "target")`)
- [ ] OUTBASE extracted from first prediction for script filename (`%s.sh` instead of `%d.sh`)
- [ ] `EnsureEnvDirs()` removed from `cmd_process.go` (executor paths are external)

### Script Template Changes
- [ ] `template.go`: Remove `TARGET_DIR`, `BASENAME`, `NAME_TAG`, `SEASON_DIR` from `AmediaGeneric`
- [ ] Keep `SOURCE`, `YADIF`, `HSUB`, `BUFFER`, `IN_PROGRESS`, `DONE`, `NOTIFICATIONS` as env vars
- [ ] `|=target_root=|` placeholder removed
- [ ] `|=priority=|` replaced with direct value

### Audio/Video Fixes
- [ ] `audio.go:139`: Audio line includes `-map_metadata -1 -map_chapters -1`
- [ ] Video line verified: `-profile:v high -g 0` (no change needed)

### Path Normalization
- [ ] `ToSlash()` utility created, applied to all executor paths before embedding in scripts

### Tests
- [ ] `config_test.go` passes with new structure
- [ ] Pipeline tests verify predictions use executor paths, not generator paths
- [ ] Template tests verify generated script matches `working_script_example.txt` format
- [ ] OUTBASE filename test: `Arliss_s01e06_HD.mp4` → `Arliss_s01e06.sh`

### Verification
- [ ] Generated script has full literal paths (no `${TARGET_DIR}` concatenation)
- [ ] Script filename follows `OUTBASE.sh` convention
- [ ] `-map_metadata -1 -map_chapters -1` present on audio output line
- [ ] `go build ./...`, `go test ./...`, `go vet ./...` all pass
