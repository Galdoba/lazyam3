# Bounded Context: Script

> **Status:** active  
> **Owner:** pipeline engineering  
> **Entropy level:** Medium

## 1. Purpose

The script domain generates bash/ffmpeg rendering scripts from structured inputs produced by the ingest and metadata domains. It composes naming, video filters, audio filter chains, subtitle handling (external SRT + hardsub burn-in), and output file management into a single executable bash script.

**Out of scope:** Actual file rendering or ffmpeg execution (belongs to the platform/external tools), file ingestion or probing (belongs to the *Ingest* domain), product metadata resolution (belongs to the *Metadata* domain), call-name prediction for output filenames (belongs to the *Callname* subdomain), and any user-facing interface. The domain knows nothing about disk I/O or pipeline orchestration — only about composing the right ffmpeg command lines and file-moving logic from structured inputs.

## 2. Context & Environment

- **Users / Actors:** The automated pipeline is the sole actor. No human directly interacts with this domain; it is invoked by the pipeline orchestrator (`Pipeline.processUnit`) which assembles a `ScriptConfig` from the ingest unit and metadata, then calls `appscript.Build` → `BuildScript`.
- **Adjacent domains and systems:**
  - **Ingest** — provides the parsed streams and filenames via `ScriptConfig.Streams` and `ScriptConfig.Filenames`.
  - **Metadata** — provides product information (series title, season/episode numbers, kind) via `ScriptConfig.Product`.
  - **Callname** — provides predicted output filenames via `ScriptConfig.Predictions` and `ScriptConfig.CallNames`.
  - **ffbash/ffmpeg (external)** — the generated bash script is executed by an external system; the domain only produces the script text.
- **Key business rules:** A rendering script must contain at least one video stream. Audio tracks are resampled to 48 kHz with tempo correction. Interlaced video gets the `yadif` deinterlace filter. External subtitles (`.srt`, `.ass`, `.ssa`) are moved through the pipeline stages (buffer → in_progress → done). Hardsub files (`.srt` tagged via HSUB config) are burned into video using ffmpeg's `subtitles=` filter. Output filenames are derived from call-name predictions, with a fallback to stream-based derivation when predictions are unavailable.

## 3. Boundaries & Interactions

- **Inbound interactions:**
  - `BuildScript(ScriptConfig) (Script, error)` — pure function that validates inputs, assembles the bash script from template, and returns the rendered script.
  - `ScriptConfig` — assembled by the application layer (`pipeline.buildScriptConfig`) from ingest unit, metadata product, call-name predictions, and environment paths.
- **Outbound interactions:**
  - Returns a `Script` DTO to the application layer.
  - The domain port `Renderer.Render(Script) error` is implemented by infrastructure; the domain only defines the interface.
  - The domain itself makes no external calls; all I/O is delegated to infrastructure.
- **Domain Events:** None. The domain is synchronous and returns a DTO directly.
- **Dependencies:**
  - `domain/script` imports `domain/ingest` (for `Stream`, `StreamKind`, `Layout` types).
  - `domain/script` imports `domain/metadata` (for `Product` type).
  - `domain/script` imports `callname` (for `PredictedOutput` type).

## 4. Ubiquitous Language

| Term | Definition | Synonyms/Notes |
|------|------------|----------------|
| **ScriptConfig** | Input DTO for building a rendering script. Aggregates ingest streams, metadata, predictions, and environment paths. | "config", "build config" |
| **Script** | The rendered bash script ready to be written to disk. Contains `Destination` path and `Content` (bash text). | "rendering script", "output script" |
| **Environment** | Filesystem paths used by the generated script (buffer, in_progress, done, notifications, target root, archive). | "env", "paths" |
| **StreamSelection** | Rules for selecting specific streams from input files. | "stream filter", "selection" |
| **PredictedOutput** | A predicted output file with `SourceFile`, `Directory`, `Filename`, and `IsVideo` fields. Produced by callname. | "prediction", "output prediction" |
| **audioTrack** | Internal representation of a selected audio track for filter generation. | "audio track" (internal) |
| **Renderer** | Domain port interface: `Render(Script) error`. Implemented by infrastructure. | "renderer port", "materializer" |
| **Placeholder** | Template substitution token in `|=key=|` format. All placeholders must be resolved before the script is returned. | "template token", "variable" |
| **HSUB** | Hardsub subtitle — an `.srt` file burned into video using ffmpeg's `subtitles=` filter. Detected via config tag. | "hardsub", "burned subtitle" |
| **SRT** | External subtitle file (`.srt`, `.ass`, `.ssa`) copied alongside rendered video. Not burned in. | "soft subtitle", "external subtitle" |
| **Layout** | Audio channel configuration: `stereo`, `surround51`, `surround71`, `mono`. Maps to a numeric suffix for output filenames. | "channel layout", "audio layout" |
| **YADIF** | "Yet Another DeInterlaced Filter" — applied when any video stream is interlaced. | "deinterlace", "yadif filter" |
| **Ready file** | A `.ready` marker file placed in the output directory after rendering (e.g., `Show_HD.ready`). | "marker", "completion signal" |

## 5. Key Scenarios / Use Cases

### Scenario 1: Build a basic episode rendering script
- **Given** a `ScriptConfig` with video + audio streams, a Product (series title, season 1, episode 6), predictions, and environment paths
- **When** `BuildScript(cfg)` is called
- **Then** a bash script is returned with ffmpeg command, audio filter, SRT/HSUB sections (if any), and all placeholders resolved
- **And** the script contains correct `OUTBASE` (e.g., `"Arliss_s01e06"`) and `TARGET_DIR` (e.g., `"/mnt/ROOT/EDIT/Arliss"`)

### Scenario 2: Build a movie rendering script (no season/episode)
- **Given** a Product with `Kind="movie"` (no season/episode numbers)
- **When** `BuildScript(cfg)` is called
- **Then** `NAME_TAG` is `""` (no `_se` artifact) and `SEASON_DIR` is `""`
- **And** `OUTBASE` is just the base name (e.g., `"MovieTitle"`)
- **And** `TARGET_DIR` is `"/mnt/ROOT/EDIT/MovieTitle"` (no season subdirectory)

### Scenario 3: Multi-audio track with layout-based naming
- **Given** streams with two audio tracks: Russian stereo and English 5.1
- **When** `BuildScript(cfg)` is called
- **Then** the audio filter contains two `aresample=48000` chains
- **And** output files are named with layout suffixes: `_HD_AUDIO_00_RUS20.m4a` and `_HD_AUDIO_01_ENG51.m4a`

### Scenario 4: Interlaced video with yadif filter
- **Given** a video stream where `Interlaced=true`
- **When** `BuildScript(cfg)` is called
- **Then** the video filter chain includes `"yadif,"` before any other filters
- **And** the placeholder `|=yadif=|` resolves to `"yadif,"`

### Scenario 5: External SRT subtitle handling
- **Given** a `ScriptConfig` with `SRT="english.srt"` and predictions
- **When** `BuildScript(cfg)` is called
- **Then** the script contains: move from buffer to in_progress, copy to target dir with prediction filename, cleanup to done
- **And** the SRT section uses prediction directory and prediction filename for the copy destination

### Scenario 6: Hardsub (HSUB) burn-in
- **Given** a `ScriptConfig` with `HSUBFile="hardsub.srt"` and environment paths
- **When** `BuildScript(cfg)` is called
- **Then** the video filter chain includes a `subtitles=` filter with absolute path to the hardsub file in `IN_PROGRESS`
- **And** the script contains move and cleanup shell lines for the hardsub file
- **And** an `HSUBFILE` bash variable is set

### Scenario 7: Prediction fallback (derive from streams)
- **Given** a `ScriptConfig` with empty `Predictions` slice
- **When** `BuildScript(cfg)` is called
- **Then** predictions are derived from `Product` + `Streams` as a fallback
- **And** the script renders correctly with derived output paths

### Scenario 8: Validation — missing video stream
- **Given** a `ScriptConfig` with only audio streams (no video)
- **When** `BuildScript(cfg)` is called
- **Then** an error is returned indicating that at least one video stream is required

### Scenario 9: Stream selection references non-existent stream
- **Given** a `ScriptConfig` with `StreamSelections` that reference a stream index not present in `Streams`
- **When** `BuildScript(cfg)` is called
- **Then** an error is returned indicating the invalid selection reference

### Scenario 10: Call-name-based source file
- **Given** a `ScriptConfig` with `CallNames` mapping original filename to a call name
- **When** `BuildScript(cfg)` is called
- **Then** the `SOURCE` variable uses the call name, not the original filename
- **And** the `|=source=|` placeholder resolves to the call name

## 6. High-Level Data Model (Key Entities)

- **ScriptConfig** — aggregate input DTO. Assembled by the application layer from ingest, metadata, callname, and environment sources.
  - Key invariants: at least one video stream required; selections must reference existing streams; destination and environment paths must be non-empty.
  - Fields: `Filenames` ([]string), `Streams` ([]ingest.Stream), `SRT` (string), `HSUBFile` (string), `Product` (metadata.Product), `StreamSelections` ([]StreamSelection), `Predictions` ([]callname.PredictedOutput), `CallNames` (map[string]string), `NameTag` (string), `SeasonDir` (string), `PRTTimestamp` (string), `Env` (Environment), `Destination` (string).

- **Script** — output DTO. The rendered bash script.
  - Fields: `Destination` (string), `Content` (string).

- **Environment** — filesystem paths used by the generated script.
  - Fields: `Buffer`, `InProgress`, `Done`, `Notifications`, `Archive`, `TargetRoot` (all string), `Priority` (int).

- **StreamSelection** — rules for selecting specific streams.
  - Fields: `InputIndex` (int), `StreamIndex` (int), `Kind` (string: "v", "a", or "s").

- **audioTrack** — internal helper for audio filter generation.
  - Fields: `StreamIndex` (int), `Language` (string), `Layout` (string), `Suffix` (string, e.g., "AUDIO_00_RUS20").

- **Renderer** — domain port interface: `Render(Script) error`. Implemented by infrastructure (creates parent directories, writes file, sets permissions to 0o755).

**Relationships:**
- ScriptConfig 1—1 Product (metadata)
- ScriptConfig 1—N Stream (ingest)
- ScriptConfig 1—N PredictedOutput (callname)
- ScriptConfig 1—1 Environment
- ScriptConfig 0—N StreamSelection
- Script 1—1 Destination (string path)

## 7. Non-Functional Requirements & Constraints

- **Authorization:** N/A — the domain runs as part of an automated pipeline with no user-facing access control.
- **Performance/Scale:** `BuildScript` is a pure function with no I/O. Template filling uses `strings.ReplaceAll` — O(m×n) where m is the number of placeholders and n is the average value length. Audio filter assembly is O(a) where a is the number of audio tracks (typically ≤ 5). Validation runs in O(s) where s is the number of streams.
- **Data:** No persistence. Scripts are DTOs passed through the pipeline and written to disk by infrastructure. No PII is stored.
- **External constraints:** Depends on ffmpeg command-line syntax and filter graph format. The generated bash script assumes POSIX shell compatibility and ffmpeg availability on the rendering host. Template placeholders use `|=key=|` format, which must not appear in any substituted value.

## 8. Acceptance Criteria

- [ ] Config validation rejects empty destination, empty streams, missing video stream, and invalid stream selection references.
- [ ] `BuildScript` produces a valid bash script with correct shebang, shell options, ffmpeg command, and all `|=key=|` placeholders resolved.
- [ ] Prediction fallback generates deterministic predictions from Product + Streams when `Predictions` is not set.
- [ ] Audio track selection maps predictions to audio tracks with correct language, layout, and suffix.
- [ ] Audio filter assembly generates correct `aresample=48000,atempo=...` chains with `;`-separated segments and `[0:N]` absolute indexes.
- [ ] Layout suffix mapping: stereo→"20", surround51→"51", surround71→"70", mono→"10", unknown→"20".
- [ ] Video filter returns `"yadif,"` when any video stream is interlaced, `""` otherwise.
- [ ] Interlaced detection returns true only when at least one video stream has `Interlaced=true` (ignores non-video streams).
- [ ] Hardsub filter generates a `subtitles=` filter with absolute path resolved to `IN_PROGRESS`; returns empty when `HSUBFile` is empty.
- [ ] SRT move/section/cleanup generate correct shell lines; section uses prediction filename when available.
- [ ] Target directory creation generates `mkdir -p` with prediction directory; falls back to first prediction's directory.
- [ ] Ready file generation creates `touch` with `.ready` suffix derived from the video prediction filename.
- [ ] Unresolved placeholder detection correctly identifies `|=key=|` patterns; returns nil when no placeholders remain.
- [ ] Placeholder substitution replaces `|=key=|` with value; no partial matches or double-substitution.
- [ ] `BuildScript` with call names uses the call name as `SOURCE`, not the original filename.
- [ ] Movie Product (no season/episode) produces `NAME_TAG=""` and `SEASON_DIR=""` (no `_se` artifact).
- [ ] Infrastructure `Renderer.Render` creates parent directories, writes the file, and sets mode to 0o755 (or custom mode via `WithMode` option).
- [ ] Integration test passes: real ffprobe ingest → metadata lookup → script build → bash syntax check → ffmpeg execution → output verification.

## 9. Open Questions

- [ ] Should `validateConfig` enforce a minimum audio sample rate (e.g., 44100 Hz) or bitrate for selected audio tracks? Currently only the presence of streams is validated.
- [ ] The `PRTTimestamp` field on `ScriptConfig` is set by the pipeline, not the domain. Should this field live in the domain DTO or be carried separately by the application layer?
- [ ] Should `PredictedOutput` be moved into the domain as a first-class type, or should the dependency on the `callname` package be kept as-is (external import)?
- [ ] The `StreamSelection` fields (`InputIndex`, `StreamIndex`, `Kind`) use raw types. Should they be wrapped in a dedicated `StreamKey` type for stronger typing?
- [ ] Should the domain expose a `Script.Validate()` method for post-generation validation (e.g., checking for unresolved placeholders, invalid ffmpeg syntax), or is the pre-generation `validateConfig` sufficient?
- [ ] Should `Environment` include paths for temporary files or log output, or is the current set of paths sufficient for the pipeline's needs?
