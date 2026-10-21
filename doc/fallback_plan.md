# Fallback Script Generation Plan

## Problem

When `metadata.FindByID(unit.ID)` fails (product not found in metadata.json), the pipeline **skips the unit entirely**:

```go
// internal/app/pipeline/pipeline.go:130-133
if prod == nil {
    p.emit(EventUnitFailed, unit.ID, fmt.Errorf("metadata for %d: product not found", unit.ID))
    return fmt.Errorf("metadata for %d: product not found", unit.ID)
}
```

**Consequence:** No script is generated, and the unit is lost even if it could be processed.

## Root Cause

Script generation (`script.BuildScript`) depends on `metadata.Product`:
- `SeriesTitle` → drives `deriveOutBase()` → basename for output paths
- `SeasonNumber`/`EpisodeNumber` → for season/episode tags (movie mode if nil)

Without metadata, there's no `SeriesTitle` to derive a basename.

## Solution: Directory-Based Fallback Naming

When metadata is missing, use the **source directory basename** as the fallback SeriesTitle. The unit directory name is already the most reliable human-meaningful label available — it's where the files live and is typically set by the operator or upstream system.

### Architecture

```
Pipeline.processUnit(dir)
    ├── p.md.FindByID(unit.ID)  →  nil (metadata missing)
    ├── fallbackProduct(dir)    →  Product{SeriesTitle: filepath.Base(dir), ...}
    ├── buildScriptConfig(unit, fallbackProduct, ...)
    │   ├── callname.PredictOutputs(fallbackProduct, unit, ...)
    │   │   └── deriveOutBase(fallbackProduct)  →  filepath.Base(dir) (transliterated)
    │   └── callname.BuildInputNames(fallbackProduct, ...)
    │       └── deriveOutBase(fallbackProduct)  →  filepath.Base(dir) (transliterated)
    └── script.BuildScript(cfg)  →  generates script normally
```

### 1. Pipeline Layer (App)

**File:** `internal/app/pipeline/pipeline.go`

**Change** in `processUnit()` at lines 130-133:

```go
if prod == nil {
    p.logger.Warn("metadata not found, using fallback naming", "id", unit.ID, "dir", dir)
    // Build a minimal Product from the unit directory basename.
    prod = fallbackProduct(filepath.Base(dir))
    // Continue to build script config with fallback Product.
}
```

**New function:**

```go
// fallbackProduct creates a minimal Product from the source directory basename.
// It treats the unit as a movie (no season/episode numbers).
func fallbackProduct(seriesTitle string) *metadata.Product {
    return &metadata.Product{
        SeriesTitle: seriesTitle,
        Kind:        metadata.KindMovie,
    }
}
```

### 2. Callname — Output Naming (Domain)

**File:** `internal/domain/callname/names_output.go`

**Function:** `deriveOutBase` (line 170) — already returns `""` when `SeriesTitle` is empty. Since the fallback Product sets `SeriesTitle` from `filepath.Base(dir)`, this function works as-is with no changes needed.

**Verify:** The transliteration path is:
```
deriveOutBase(product)
  → product.SeriesTitle != ""  (set from dir basename)
  → translit.String(seriesTitle, ...)  (transliterate if Cyrillic)
  → TitleCase(first char)
```

### 3. Callname — Input Naming (Domain)

**File:** `internal/domain/callname/names_input.go`

**Function:** `BuildInputName` (line 90) — calls `deriveOutBase(product)` at line 110. Since the fallback Product has `SeriesTitle` set, `deriveOutBase` returns the transliterated directory basename. No changes needed.

**Verify:** The call name construction:
```
ScriptInput{
    Base:         deriveOutBase(product),   // → transliterated dir basename
    SeasonTag:    deriveNameTag(product),    // → "" (no season/episode)
    PRT:          "PRT" + timestamp,
    OriginalName: filename,
    FileTags:     extractFileTags(stem),
}
// String(): "Base_PRT123456_idN-original_tags.ext"
```

### 4. Script Domain (Domain)

**File:** `internal/domain/script/script.go`

`BuildScript` calls `deriveNamesFromPredictions()` (line 45) which extracts baseName, nameTag, seasonDir from the predictions. Since predictions are derived from the fallback Product's SeriesTitle, the output paths use the directory basename. No changes needed.

### 5. Output Prediction (Domain)

**File:** `internal/domain/callname/names_output.go`

**Function:** `PredictOutputs` (line 35) — uses `deriveOutBase(product)` at line 48. With fallback Product having `SeriesTitle` set, this works as-is.

**Output path layout with fallback:**
```
// dir = "/path/to/units/Some Show"
// fallback Product: Product{SeriesTitle: "Some Show", Kind: KindMovie}

baseName    = "Some_Show"           // transliterated, TitleCase
seasonTag   = ""                    // no SeasonNumber
nameTag     = ""                    // no EpisodeNumber
seasonDir   = ""                    // derived from empty seasonTag
targetDir   = targetRoot/Some_Show
video       = Some_Show_HD.mp4
output      = targetRoot/Some_Show/Some_Show_HD.mp4
```

### 6. Template Variables

When metadata is missing (fallback mode), template variables resolve as:

| Variable | Fallback Value |
|---|---|
| `BASENAME` | `filepath.Base(dir)` transliterated (e.g. `Some_Show`) |
| `name_tag` | `""` (empty — movie mode) |
| `season_dir` | `""` (empty — movie mode) |
| `SeasonNumber` | `nil` |
| `EpisodeNumber` | `nil` |

### 7. Naming Convention

**With Metadata:**
```
BASENAME   = SeriesTitle (transliterated)
name_tag   = _s01e06 (season + episode)
season_dir = _s01
Output     = <root>/<BASENAME><season_dir>/<BASENAME><name_tag>_HD.mp4
```

**Without Metadata (Fallback):**
```
BASENAME   = filepath.Base(dir) transliterated (e.g. "Some Show" → "Some_Show")
name_tag   = ""
season_dir = ""
Output     = <root>/<BASENAME>/<BASENAME>_HD.mp4
```

**Example:**
- Source dir: `/mnt/SOURCE/units/Какие_то_фильмы`
- Fallback BASENAME: `Kakie_to_filmy` (transliterated from Cyrillic dir name)
- Output: `/mnt/ROOT/EDIT/Kakie_to_filmy/Kakie_to_filmy_HD.mp4`

### 8. Events

Log a warning when fallback is used. No new event type needed — `EventUnitDone` still fires after successful processing.

```go
if prod == nil {
    p.logger.Warn("metadata not found, using fallback naming",
        "id", unit.ID, "dir", dir, "fallback_base", filepath.Base(dir))
    prod = fallbackProduct(filepath.Base(dir))
}
```

### 9. Configuration (Optional, Future)

Add config flag to control fallback behavior:

```toml
# config.toml
[paths]
# Options: "fail", "warn_and_fallback" (default)
fallback_on_missing_metadata = "warn_and_fallback"
```

## Implementation Order

1. **Pipeline layer** — Add `fallbackProduct()` helper; change `prod == nil` to warning + fallback (pipeline.go:130-133)
2. **Verify callname** — Confirm `deriveOutBase` works with fallback Product (names_output.go:170)
3. **Verify input names** — Confirm `BuildInputName` works with fallback Product (names_input.go:90)
4. **Verify script domain** — Confirm `deriveNamesFromPredictions` extracts correct values (script.go:103)
5. **Tests** — Add unit test for fallback Product in callname and integration test in pipeline

## Edge Cases

| Scenario | Behavior |
|---|---|
| Directory name is empty | `filepath.Base("")` → `"."` — unlikely in practice (scan dirs are real paths) |
| Directory name has special chars | Transliteration handles Cyrillic → Latin; other chars pass through as-is |
| Multiple files, different IDs | Unit already groups files by consistent ID (ingest.go:54-58), fallback Product doesn't affect ID |
| Metadata present for some files | Each unit is independent — fallback only applies when that unit's Product is nil |
| Directory name already transliterated | No harm — transliteration on non-Cyrillic is a no-op |

## Benefits

1. **No data loss** — Units processed even without metadata
2. **Backward compatible** — Existing behavior unchanged when metadata present
3. **Consistent output** — Same naming rules, just different SeriesTitle source
4. **Explicit warning** — Operator aware of missing metadata (log warning)
5. **Simple** — Uses directory basename (already available), no filename parsing needed
