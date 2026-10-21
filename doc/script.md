# Script Domain

The script domain generates bash/ffmpeg rendering scripts from ingest and metadata inputs. It composes naming, video/audio filters, subtitle handling (external SRT + hardsub burn-in), and output file management into a single executable bash script.

## Overview

```
ScriptConfig → [naming derivation] → [video/audio filter assembly] → [SRT/HSUB sections] → Script (bash)
```

The domain owns all rendering logic: pure functions that produce a bash script ready to be materialized by the `Renderer` port. It knows nothing about disk I/O or pipeline orchestration — only about composing the right ffmpeg command lines and file-moving logic from structured inputs.

## Domain Layer

### Types

**`ScriptConfig`** — the domain input for building a rendering script.

```go
type ScriptConfig struct {
    // From ingest domain
    Filenames      []string          // input filenames in unit order; Filenames[0] is the primary media file
    Streams        []ingest.Stream   // all parsed streams
    SRT            string            // external subtitle filename (empty if none)
    HSUBFile       string            // hardsub subtitle filename (empty if none)
    Product        metadata.Product  // episode/season/movie product (with SeriesTitle set)

    // Optional stream selections (if nil/empty, all audio auto-selected)
    StreamSelections []StreamSelection

    // Naming components (populated from Product in BuildScript)
    BaseName   string
    SeasonTag  string
    EpisodeTag string
    NameTag    string // ready-to-use suffix: "_s01e06", "_s01", "_e06", "" (movie)
    SeasonDir  string // ready-to-use target-dir suffix: "_s01", "" (movie)

    // Environment paths (filled from app config)
    Env Environment

    // Output path for the .sh file
    Destination string
}
```

**`Environment`** — filesystem paths used by the generated script.

```go
type Environment struct {
    Buffer        string // <root>/buffer
    InProgress    string // <root>/in_progress
    Done          string // <root>/done
    Notifications string // <root>/notifications
    TargetRoot    string // <root>/target
    Priority      int
}
```

**`StreamSelection`** — rules for selecting specific streams.

```go
type StreamSelection struct {
    InputIndex  int
    StreamIndex int
    Kind        string // "v", "a", "s"
}
```

**`Script`** — the rendered bash script ready to be written to disk.

```go
type Script struct {
    Destination string
    Content     string
}
```

### Pure Functions

**`BuildScript(cfg ScriptConfig) (Script, error)`**

The main entry point. Pure function: no IO, no side effects, fully deterministic.

**Steps:**
1. **Validate inputs** — at least one video stream, selections reference existing streams, destination non-empty
2. **Derive naming** from `Product`:
    - `BaseName` — transliterated series title, first char uppercase (e.g., `"Arliss"`)
    - `SeasonTag` / `EpisodeTag` — zero-padded season/episode numbers (e.g., `"01"` / `"06"`)
    - `NameTag` — ready-to-use suffix for `OUTBASE`: `"_s01e06"`, `"_s01"`, `"_e06"`, or `""` for movies (no `_se` artifact)
    - `SeasonDir` — ready-to-use suffix for `TARGET_DIR`: `"_s01"` or `""` for movies
    - The template composes `OUTBASE="${BASENAME}${NAME_TAG}"` and `TARGET_DIR="${TargetRoot}/${BASENAME}${SEASON_DIR}"` via shell variables
3. **Build video section** — `yadif` filter if interlaced; `subtitles=` filter if `HSUBFile` set (absolute path inside `IN_PROGRESS`)
4. **Build audio section** — filter_complex and map outputs for each selected audio track
5. **Build SRT sections** — shell lines for moving/copying external subtitles (empty if no SRT)
6. **Build HSUB sections** — shell lines for moving/burning hardsub subtitles (empty if none)
7. **Fill template placeholders** — replace `|=key=|` with computed values
8. **Validate** — no unresolved placeholders remain
9. **Return** `Script{Destination, Content}`

### Naming Derivation

Uses `translit.String(title, translit.RegisterLow(), translit.Short_SE_Form())` for filesystem-safe names.

- **BaseName:** `deriveBaseName(Product)` — `"Arliss"` (from `Product.SeriesTitle`)
- **SeasonTag/EpisodeTag:** `deriveSeasonTag`/`deriveEpisodeTag` — `"01"`/`"06"` (from `Product.SeasonNumber`/`EpisodeNumber`)
- **NameTag:** `deriveNameTag(SeasonTag, EpisodeTag)` — `"_s01e06"`, `"_s01"`, `"_e06"`, or `""` for movies (no `_se` artifact)
- **SeasonDir:** `deriveSeasonDir(SeasonTag)` — `"_s01"` or `""` for movies
- **deriveOutBase(Product):** combined `"Arliss_s01e06"` helper (used in tests; the template composes the same value via shell variables)

A single Product has everything needed: `SeriesTitle` (set by repo), `SeasonNumber`, `EpisodeNumber`, `Title` (episode title).

### Audio Section

Each selected audio track becomes:
- **Filter:** `[0:N]aresample=48000,atempo=25/(25/1)[audK]` (absolute `[0:N]` indexes)
- **Output:** `-map "[audK]" -c:a alac -compression_level 0 "${TARGET_DIR}/${OUTBASE}_HD_AUDIO_KK_LL.m4a"`

Layout mapping: `"stereo"→"20"`, `"5.1"→"51"`, `"7.1"→"70"`, `"mono"→"10"`

### SRT Handling

- **Move:** `mv "${BUFFER}/${SRT}" "${IN_PROGRESS}/${SRT}"`
- **Copy:** `cp "${IN_PROGRESS}/${SRT}" "${TARGET_DIR}/${OUTBASE}.srt"`
- **Cleanup:** `mv "${IN_PROGRESS}/${SRT}" "${DONE}/${SRT}"`

Empty strings if `cfg.SRT` is empty.

### HSUB Handling

When `cfg.HSUBFile` is set, `buildHSUB` generates an ffmpeg `subtitles=` filter with the path resolved to an absolute path inside `IN_PROGRESS` (where the script moves the file before rendering). The filter is prepended to the video filter chain. The script also moves the hardsub file from buffer to in_progress (`|=hsub_move=|`) and after rendering to done (`|=hsub_cleanup=|`). Empty string if no hardsub.

## Template

`AmediaGeneric` is a string constant in `template.go`. Placeholders use `|=key=|` format.

### Placeholders

| Placeholder | Source |
|---|---|
| `|=source=|` | `Filenames[0]` |
| `|=srt=|` | `SRT` field |
| `|=hsubfile=|` | `HSUBFile` field |
| `|=basename=|` | Derived from Product (`BaseName`) |
| `|=name_tag=|` | Derived from Product (`NameTag`) |
| `|=season_dir=|` | Derived from Product (`SeasonDir`) |
| `|=yadif=|` | `"yadif,"` if interlaced, `""` otherwise |
| `|=hsub=|` | `subtitles=` filter if `HSUBFile` set, `""` otherwise |
| `|=audio_filter=|` | Built from selected audio tracks |
| `|=audio_outputs=|` | Built from selected audio tracks |
| `|=srt_move=|` | SRT move shell line |
| `|=srt_section=|` | SRT copy shell line |
| `|=srt_cleanup=|` | SRT cleanup shell line |
| `|=hsub_move=|` | HSUB move shell line |
| `|=hsub_cleanup=|` | HSUB cleanup shell line |
| `|=priority=|` | `Env.Priority` |
| `|=buffer=|` | `Env.Buffer` |
| `|=in_progress=|` | `Env.InProgress` |
| `|=done=|` | `Env.Done` |
| `|=notifications=|` | `Env.Notifications` |
| `|=target_root=|` | `Env.TargetRoot` |

`OUTBASE` is not a placeholder — the template composes it via shell variables: `OUTBASE="${BASENAME}${NAME_TAG}"`.

## Renderer Port

```go
// port.go
type Renderer interface {
    Render(sc Script) error
}
```

Materializes a built `Script` as a usable artifact. The domain only defines the contract; the concrete implementation is `internal/infra/script.Renderer` (creates parent directories, writes the file with mode `0o755` by default via chmod so umask cannot strip the executable bit; options pattern for future hooks). `app/script.Build` = `BuildScript` → `Renderer.Render`.

## Pure Functions Reference

| Function | File | Purpose |
|---|---|---|
| `BuildScript(cfg)` | `script.go` | Main entry point |
| `validateConfig(cfg)` | `script.go` | Input validation |
| `deriveBaseName(p)` | `naming.go` | Transliterated series title |
| `deriveSeasonTag(p)` / `deriveEpisodeTag(p)` | `naming.go` | Zero-padded s/e tags |
| `deriveOutBase(p)` | `naming.go` | Combined output base (tested helper) |
| `deriveNameTag(seasonTag, episodeTag)` | `naming.go` | Ready-to-use name suffix |
| `deriveSeasonDir(seasonTag)` | `naming.go` | Ready-to-use target-dir suffix |
| `selectAudioTracks(cfg)` | `audio.go` | Selected audio from streams |
| `buildAudioFilter(tracks)` | `audio.go` | ffmpeg filter_complex segment (absolute `[0:N]` indexes, `;`-prefixed block) |
| `buildOutputFiles(tracks)` | `audio.go` | ffmpeg -map output block: video line + one ALAC line per audio track; `\` continuation on all but the last line |
| `layoutSuffix(layout)` | `audio.go` | Layout→suffix mapping |
| `getInterlaced(streams)` | `audio.go` | Check if any video is interlaced |
| `buildVideoFilter(interlaced)` | `audio.go` | yadif filter or empty |
| `buildHSUB(hsubFile, inProgress)` | `script.go` | `subtitles=` filter with absolute path, empty |
| `buildHSUBMove(hsubFile)` | `script.go` | HSUB buffer→in_progress move, empty |
| `buildHSUBCleanup(hsubFile)` | `script.go` | HSUB in_progress→done move, empty |
| `buildSRTMove/Section/Cleanup(srt)` | `script.go` | SRT shell lines |
| `findUnresolved(content)` | `script.go` | Unresolved placeholder check |
| `replaceAll(template, key, value)` | `script.go` | Placeholder substitution |

## Testing

Golden file test (`TestBuildScript_GoldenFile`) verifies end-to-end rendering with:
- 2 audio tracks (rus/stereo, eng/stereo)
- Interlaced video (1920×1080)
- SRT subtitle
- Single Product with `SeriesTitle: "Белая королева"`, `SeasonNumber: 1`, `EpisodeNumber: 1`
- Env paths rendered as bash-valid plain assignments

Error cases test: missing video, empty streams, empty destination, missing selection reference.

HSUB tests (`TestBuildScript_WithHSUB`, `TestBuildHSUB`) verify the hardsub filter with absolute path and `HSUBFILE` variable; synthetic files live in `testdata/synthetic/hardsub-with-subtitle/` (now contains `Forsirovannye`-tagged hardsub srt alongside external srt).

Naming tests verify Cyrillic→Latin transliteration for various Product configurations, including movie (no season/episode → `NAME_TAG=""`, `SEASON_DIR=""`).

## File Structure

```
internal/domain/script/
├── script.go       # BuildScript, validateConfig, buildHSUB/Move/Cleanup, buildSRT*
├── template.go     # AmediaGeneric template (|=key=| placeholders)
├── naming.go       # deriveBaseName, deriveSeasonTag, deriveEpisodeTag, deriveNameTag, deriveSeasonDir
├/audio.go          # selectAudioTracks, buildAudioFilter, buildOutputFiles, layoutSuffix, getInterlaced, buildVideoFilter
├── translit/       # Cyrillic→Latin transliteration subdomain
└── script_test.go  # 17 test functions: golden, errors, HSUB, audio, naming, translations

internal/infra/script/
├── renderer.go     # Renderer: MkdirAll, WriteFile, Chmod (0o755 default)
└── renderer_test.go  # 3 unit tests
```
