# HANDOFF: PRT Timestamp on Output Files

## Problem

When the pipeline generates a rendering script, input files and the script filename itself receive a `PRT{timestamp}` tag, but the **output files** produced by the script (video, audio, subtitles, ready-file) do **not** contain this tag.

### Current behavior

| Artifact | Naming pattern | Contains PRT? |
|---|---|---|
| Input file in `in_progress/` | `Arliss_s01e06_PRT261021140537_id12254.mp4` | ✅ |
| Script file | `Arliss_PRT261021140537.sh` | ✅ |
| Output video | `Belaya_koroleva_s01e06_HD.mp4` | ❌ |
| Output audio | `Belaya_koroleva_s01e06_HD_AUDIO00_RUS20.m4a` | ❌ |
| Output subtitle | `Belaya_koroleva_s01e06.srt` | ❌ |
| Ready-file | `Belaya_koroleva_s01e06.ready` | ❌ |

### Expected behavior

All output files should include the PRT stamp:

| Artifact | Expected pattern |
|---|---|
| Output video | `{OutBase}_PRT{timestamp}_HD.mp4` |
| Output audio | `{OutBase}_PRT{timestamp}_HD_AUDIO00_RUS20.m4a` |
| Output subtitle | `{OutBase}_PRT{timestamp}.srt` |
| Ready-file | `{OutBase}_PRT{timestamp}.ready` |

Where `{OutBase} = BaseName + NameTag` (e.g., `Belaya_koroleva_s01e06`).

## Root Cause

The single source of truth for output filenames is `callname.PredictOutputs()` in `internal/domain/callname/names_output.go:35`. It computes output paths from `Product` + `MediaUnit` + `targetRoot` + `hardsubTag` but **does not receive or use a timestamp**. The PRT tag is only injected at two other places:

1. **`callname.BuildInputNames()`** (`names_input.go:90`) — receives `timestamp` as parameter and embeds `PRT{timestamp}` in call names for input files.
2. **`pipeline.buildScriptConfig()`** (`pipeline/config.go:85`) — generates the script filename with `PRT{timestamp}`.

`PredictOutputs()` has no timestamp parameter, so it can only produce names like `{OutBase}_HD.mp4`.

## Call Sites of `PredictOutputs`

| File | Line | Context |
|---|---|---|
| `internal/app/pipeline/config.go:57` | Primary pipeline wiring | Calls `PredictOutputs(prod, unit, targetRoot, hardsubTag)` — **has timestamp available** |
| `internal/domain/script/script.go:330` | `derivePredictionsFromStreams()` fallback | Calls `callname.PredictOutputs(product, unit, targetRoot, "")` — **no timestamp available** |
| `internal/app/script/build_test.go:91` | Test `realConfig()` | Calls `PredictOutputs(*prod, unit, path, "")` — **has timestamp available** |
| `internal/domain/callname/names_output_test.go` | 8+ tests | Calls `PredictOutputs(product, unit, "/mnt/ROOT/EDIT", "")` — **no timestamp** |

## Solution Approaches

### Approach 1: Add `prtTimestamp` parameter to `PredictOutputs` (RECOMMENDED)

Add an optional `string` parameter for the PRT timestamp to the `PredictOutputs` signature:

```go
func PredictOutputs(product metadata.Product, unit ingest.MediaUnit, targetRoot string, hardsubTag string, prtTimestamp string) []PredictedOutput
```

**Changes required:**

1. **`callname/names_output.go`** — Add `prtTimestamp` parameter; insert `PRT{timestamp}` into `outBase` before constructing paths. The timestamp is only inserted if non-empty (preserves backward compatibility).

2. **`pipeline/config.go:57`** — Pass the timestamp already computed at line 60 (`timestamp := time.Now().Format("060102150405")`) to `PredictOutputs`.

3. **`script/script.go:330`** — `derivePredictionsFromStreams()` has no timestamp. Options:
   - Generate a timestamp locally inside `derivePredictionsFromStreams`, or
   - Accept a `prtTimestamp string` parameter from the caller, or
   - Accept the timestamp as a new optional field in `ScriptConfig` (cleaner but more invasive).

4. **`build_test.go:91`** — Pass the timestamp already computed at line 84.

5. **`names_output_test.go`** — Update all 8+ test calls to pass `""` for the new parameter (empty string preserves current behavior).

**Pros:**
- Single source of truth for output naming logic.
- `PredictOutputs` is the only function that knows how to compute output paths; it gets the timestamp it needs.
- Script domain's `derivePredictionsFromStreams` fallback also gets consistent naming.

**Cons:**
- Signature change on a domain-level function (but it's the canonical naming function).
- Need to thread timestamp through `derivePredictionsFromStreams` or generate it locally.

### Approach 2: Derive PRT from Product ID + call names (no signature change)

Instead of passing a timestamp, derive the PRT tag from the `CallNames` map that's already available in `ScriptConfig.Predictions`. Since predictions are built before call names, this creates a chicken-and-egg problem — not viable without reordering the pipeline logic.

**Verdict: Not feasible** as a standalone approach.

### Approach 3: Post-process predictions in `pipeline.buildScriptConfig`

After calling `PredictOutputs`, iterate over predictions and inject the PRT tag into each path string in the pipeline layer.

**Changes required:**

1. **`pipeline/config.go`** — After `PredictOutputs`, call a helper that inserts `_PRT{timestamp}` into each prediction's `Path` before it's stored.

2. The `PredictOutputs` signature stays unchanged.

**Pros:**
- Minimal signature changes to domain code.
- Keeps timestamp handling in the application layer.

**Cons:**
- `derivePredictionsFromStreams` in the script domain still has no timestamp.
- Duplicates path-mutation logic between pipeline and script domain.
- The script domain's fallback path would produce different names than the pipeline's primary path.

**Verdict: Weaker than Approach 1** because it doesn't fix the script domain fallback.

### Approach 4: Add `PRTTimestamp` to `ScriptConfig`

Add a `PRTTimestamp string` field to `ScriptConfig`, set by the pipeline, consumed by both `PredictOutputs` (via pipeline) and `derivePredictionsFromStreams` (via config).

**Changes required:**

1. **`script/config.go`** — Add `PRTTimestamp string` to `ScriptConfig`.
2. **`pipeline/config.go`** — Set the field from the already-computed timestamp.
3. **`PredictOutputs`** — Keep signature; pipeline sets the timestamp before passing predictions.
4. **`derivePredictionsFromStreams`** — Read from config's `PRTTimestamp`.

**Pros:**
- No signature change to `PredictOutputs`.
- Timestamp flows naturally through the config.

**Cons:**
- `PredictOutputs` is called by the pipeline, not the script domain — it wouldn't have access to `ScriptConfig`.
- The pipeline would need to call `PredictOutputs` first (to get predictions), then pass them to `ScriptConfig`, then pass the config to `BuildScript`. This is already the current flow, so it works.
- However, `PredictOutputs` still wouldn't receive the timestamp directly.

**Verdict: Viable but more indirect than Approach 1.** Approach 1 is simpler because the timestamp is a direct parameter to the function that needs it.

## Recommended Implementation (Approach 1)

### Step-by-step changes

1. **`callname/names_output.go`** — Modify `PredictOutputs`:
   ```go
   func PredictOutputs(product metadata.Product, unit ingest.MediaUnit, targetRoot string, hardsubTag string, prtTimestamp string) []PredictedOutput {
       // ... existing logic ...
       outBase := baseName + nameTag
       if prtTimestamp != "" {
           outBase = outBase + "_PRT" + prtTimestamp
       }
       // ... rest unchanged ...
   }
   ```

2. **`pipeline/config.go:57-60`** — Reuse the existing timestamp:
   ```go
   predictions := callname.PredictOutputs(prod, unit, targetRoot, hardsubTag, timestamp)
   ```

3. **`pipeline/config.go:61`** — The timestamp used for `BuildInputNames` can be the same one (or a fresh one for the script filename).

4. **`script/script.go:313`** — Modify `derivePredictionsFromStreams`:
   ```go
   func derivePredictionsFromStreams(product metadata.Product, streams []ingest.Stream, targetRoot string, prtTimestamp string) []callname.PredictedOutput {
       return callname.PredictOutputs(product, unit, targetRoot, "", prtTimestamp)
   }
   ```
   And update its caller (line 40) — but this is called from `BuildScript` which receives `ScriptConfig`. Add `PRTTimestamp` to `ScriptConfig`.

5. **`script/config.go`** — Add `PRTTimestamp string` field to `ScriptConfig`.

6. **`pipeline/config.go:110-128`** — Set `PRTTimestamp` in the returned `ScriptConfig`.

7. **`build_test.go:91`** — Pass the timestamp from line 84.

8. **`names_output_test.go`** — Update all `PredictOutputs(...)` calls to pass `""` as the 5th argument.

### Files to modify

| File | Change |
|---|---|
| `internal/domain/callname/names_output.go` | Add `prtTimestamp` param; insert into `outBase` |
| `internal/domain/callname/names_output_test.go` | Update all calls with `""` |
| `internal/app/pipeline/config.go` | Pass timestamp to `PredictOutputs`; add `PRTTimestamp` to `ScriptConfig` |
| `internal/domain/script/config.go` | Add `PRTTimestamp` field |
| `internal/domain/script/script.go` | Update `derivePredictionsFromStreams` signature; use config's `PRTTimestamp` |
| `internal/app/script/build_test.go` | Pass timestamp to `PredictOutputs` |

### Impact on generated script

The ffmpeg command and all shell operations in the generated script will target the new filenames:

```bash
# Before:
ffmpeg -i "${IN_PROGRESS}/${SOURCE}" \
  -map "[vidHD]" -c:v libx264 ... "${TARGET_DIR}/${OUTBASE}_HD.mp4"

# After (OUTBASE = "Show_s01e06_PRT261023120000"):
ffmpeg -i "${IN_PROGRESS}/${SOURCE}" \
  -map "[vidHD]" -c:v libx264 ... "${TARGET_DIR}/${OUTBASE}_HD.mp4"
  # resolves to: Show_s01e06_PRT261023120000_HD.mp4
```

The `OUTBASE` shell variable is computed from `BaseName + NameTag + "_PRT" + timestamp`, so all output files automatically inherit the PRT stamp.
