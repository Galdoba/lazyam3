# HANDOFF: Output Naming, Script/Output Path Separation, Deprecated Cleanup

## Summary

Over the session we fixed three categories of issues in the lazyam3 pipeline:

1. **Output naming inconsistency** — output files (video, audio, subtitles) were missing the product `_idNNNN` component and had an inconsistent root format compared to input files.
2. **Script vs. rendered output path confusion** — scripts were landing in `target/` (the executor's rendered output directory) instead of `buffer/` (the generator output directory), and rendered output files were going to `buffer/` instead of `target/`.
3. **Deprecated functions** — 8 unused functions in `internal/domain/script/` (replaced by the `callname` package) needed to be commented out.

---

## 1. Output Naming Consistency

### Problem

Output files produced by the rendering script did not include the product ID (`_idNNNN`) in their filename root, while input files always included it. This created an inconsistency between input and output naming conventions.

**Before:**

| Artifact | Pattern |
|---|---|
| Input file | `Arliss_s01e06_PRT261023110725_id12254.mp4` |
| Output video | `Arliss_s01_06_PRT261023110725_HD.mp4` ❌ missing `_id12254` |
| Output audio | `Arliss_s01_06_PRT261023110725_HD_AUDIO00_RUS20.m4a` ❌ missing `_id12254` |

**After:**

| Artifact | Pattern |
|---|---|
| Input file | `Arliss_s01e06_PRT261023110725_id12254.mp4` |
| Output video | `Arliss_s01_06_PRT261023110725_id12254_HD.mp4` ✅ |
| Output audio | `Arliss_s01_06_PRT261023110725_id12254_HD_AUDIO00_RUS20.m4a` ✅ |

### Changes

| File | Change |
|---|---|
| `internal/domain/callname/names_output.go` | Added `prodID int64` parameter to `PredictOutputs`; inserts `_id{prodID}` into `outBase` after PRT timestamp |
| `internal/domain/callname/names_output.go` | Added `deriveOutputNameTag()` function (returns `_s01_06` format, replacing `deriveNameTagWithUnderscore`) |
| `internal/domain/callname/names_output.go` | Added `deriveSeasonTag()` and `deriveEpisodeTag()` helper functions |
| `internal/app/pipeline/config.go` | Pass `prod.ID` to `PredictOutputs(prod, unit, targetRoot, hardsubTag, prod.ID)` |
| `internal/domain/script/script.go` | Pass `product.ID` to `PredictOutputs` in `derivePredictionsFromStreams` |
| `internal/app/script/build_test.go` | Pass `prod.ID` to `PredictOutputs` in test |
| `internal/domain/callname/names_output_test.go` | Updated all 6 test calls to pass the product's ID |

### Additional fixes in this area

- **Subtitle `_HD` suffix** — Subtitle files now include `_HD` for consistency: `{OutBase}_HD.srt` instead of `{OutBase}.srt`.
- **Fallback products** — `fallbackProduct()` in `pipeline.go` now accepts `unitID` and sets `Product.ID = unitID`, so fallback-named units (those not found in metadata.json) also get `_idNNNN` in output filenames.

---

## 2. Script vs. Rendered Output Path Separation

### Problem

The pipeline used a single `targetRoot` variable for both script destination and rendered output paths. The `targetRoot` was set from `executorPaths["target"]` (the executor's rendered output directory), which caused:

- **Scripts** to be written to `target/` instead of `buffer/`
- **Rendered output paths** (in ffmpeg commands) to point to `buffer/` instead of `target/`

### Solution

Split the root path into two distinct variables:

- **`renderRoot`** — from `executorPaths["target"]`, used by `PredictOutputs` for rendered output video/audio paths
- **`scriptRoot`** — from the generator output directory (`root`), used by `PredictScriptName` for script destination and `Env.TargetRoot`

**Before:**

```
Scripts → target/
Rendered output → buffer/
```

**After:**

```
Scripts → buffer/ (generator output)
Rendered output → target/ (executor target)
```

### Changes

| File | Change |
|---|---|
| `internal/app/pipeline/config.go` | Split `targetRoot` into `renderRoot` (from executorPaths["target"]) and `scriptRoot` (from root); pass `renderRoot` to `PredictOutputs`, `scriptRoot` to `PredictScriptName` |
| `internal/app/pipeline/config.go` | `Env.TargetRoot` set to `scriptRoot` (buffer directory) |
| `internal/app/pipeline/pipeline.go` | Removed unused `cfg config.Config` field and its import |

---

## 3. Season Directory Fix

### Problem

Season directories were missing the underscore separator between the base name and season tag:

- **Before:** `target/Arlisss01/` (double `s`, no underscore)
- **After:** `target/Arliss_s01/` (correct)

### Changes

| File | Change |
|---|---|
| `internal/domain/callname/names_output.go` | Changed `seasonDir = "s" + seasonTag` to `seasonDir = "_s" + seasonTag` |

---

## 4. Deprecated Functions Cleanup

### Problem

8 unused functions in `internal/domain/script/` (replaced by the `callname` package) were flagged by the linter as `unused`.

### Functions Commented Out

**`internal/domain/script/audio.go`:**
- `selectAudioTracks()` — audio track selection logic
- `buildOutputFiles()` — ffmpeg output block rendering

**`internal/domain/script/naming.go`:**
- `deriveOutBase()` — output filename base derivation
- `deriveBaseName()` — transliterated series title
- `deriveSeasonTag()` — season number formatting
- `deriveEpisodeTag()` — episode number formatting
- `deriveNameTag()` — season/episode suffix
- `deriveSeasonDir()` — season directory suffix

All 8 functions are now commented out with `// Deprecated: replaced by callname package.` annotations. Unused imports in `naming.go` were removed.

### Linter Result

| Before | After |
|---|---|
| 12 issues (9 unused + 3 errcheck) | 3 issues (3 errcheck in `logging_test.go` — pre-existing) |

---

## Files Modified

| File | Category |
|---|---|
| `internal/domain/callname/names_output.go` | Output naming, season dir |
| `internal/domain/callname/names_output_test.go` | Test updates |
| `internal/domain/callname/names_input.go` | (unchanged, read for context) |
| `internal/app/pipeline/config.go` | Path separation |
| `internal/app/pipeline/pipeline.go` | Removed unused `cfg` field |
| `internal/domain/script/audio.go` | Deprecated functions commented |
| `internal/domain/script/naming.go` | Deprecated functions commented, imports removed |
| `internal/domain/script/script.go` | Pass `product.ID` to `PredictOutputs` |
| `internal/app/script/build_test.go` | Pass `prod.ID` to `PredictOutputs` |

## Final State

```
Input files (buffer/):          Output files (target/):
Show_s01e06_PRT..._id12345.mp4  Show_s01_06_PRT..._id12345_HD.mp4
Show_s01e06_PRT..._id12345.srt  Show_s01_06_PRT..._id12345_HD_AUDIO00_RUS20.m4a
Show_s01e06_PRT...sh           Show_s01_06_PRT..._id12345_HD.srt
                               Show_s01_06_PRT..._id12345_HD.ready
```

- Input naming: compact (`s01e06`), includes `_idNNNN`
- Output naming: spaced (`_s01_06`), includes `_idNNNN`, includes `_HD` suffix
- Scripts → `buffer/`
- Rendered output → `target/`
- 8 deprecated functions commented out, linter clean (3 pre-existing test issues only)
