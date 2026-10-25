# Bounded Context: CallName

> **Status:** active  
> **Owner:** pipeline engineering  
> **Entropy level:** Medium

## 1. Purpose

The callname domain constructs self-documenting filenames for both input (source) files and output (rendered) files in the media processing pipeline. It produces **call names** — human-readable, metadata-enriched filenames that encode the product identity, season/episode information, pipeline run timestamp, and per-file attributes (language, layout, codec quality). The domain is split into two responsibilities: **input naming** (`ScriptInput`) for renaming source files before rendering, and **output prediction** (`PredictedOutput`) for determining the filenames the rendering pipeline will produce.

**Out of scope:** The actual file renaming operations (handled by the rendering script), stream-level processing (belongs to the *Ingest* domain), product metadata resolution (belongs to the *Metadata* domain), and script template generation (belongs to the *Script* domain). The domain knows nothing about execution — only about naming.

## 2. Context & Environment

- **Users / Actors:** The automated pipeline is the sole actor. No human directly interacts with this domain; it is invoked by the pipeline orchestrator during the script configuration phase, which calls `BuildInputNames()` for source files and `PredictOutputs()` / `PredictScriptName()` for rendered outputs.
- **Adjacent domains and systems:** 
  - **Ingest** — provides the `MediaUnit` (streams, filenames, PRT timestamp) consumed by `PredictOutputs()`.
  - **Metadata** — provides the `Product` record (SeriesTitle, SeasonNumber, EpisodeNumber, Kind) used by both input and output naming logic.
  - **Script** — consumes `PredictedOutput` and `CallNames` from callname to build rendering scripts that reference the correct filenames.
  - **Translit (internal)** — a self-contained Cyrillic transliteration subsystem (`callname/translit/`) that converts Unicode titles into ASCII-safe filename components.
- **Key business rules:** Input call names follow the pattern `{Base}_{sN}{eM}_PRT{timestamp}_id{ID}[-{file_tags}]{ext}`. Output predictions follow `{Base}{seasonDir}/{Base}{nameTag}_PRT{timestamp}_HD.{ext}` for video, `{Base}{nameTag}_PRT{timestamp}_HD_AUDIO{index}_{lang}{layout}.m4a` for audio, and `{Base}{nameTag}_PRT{timestamp}_HD.srt` for subtitles. Products without season/episode numbers (movies) omit the `s`/`e` fragments. Cyrillic titles are transliterated using a configurable transliteration engine with a default Russian→Latin literal map.

## 3. Boundaries & Interactions

- **Inbound interactions:** 
  - `BuildInputName(Product, originalFilename, timestamp) (ScriptInput, error)` — constructs a call name for a single source file.
  - `BuildInputNames(Product, []string, timestamp) (map[string]string, error)` — batch version: maps each original filename to its call name. Fails fast if any file fails.
  - `PredictOutputs(Product, MediaUnit, targetRoot, hardsubTag, prodID) []PredictedOutput` — predicts all output files the rendering pipeline will produce for a media unit.
  - `PredictScriptName(Product, MediaUnit, targetRoot, hardsubTag, timestamp) string` — predicts the rendering script filename.
  - `NewPRT() string` — generates a pipeline run timestamp (`PRTYYMMDDHHmmss`).
  - `ExtractID(filename) (int64, error)` — parses the product ID from a filename (leading digits of the stem).
- **Outbound interactions:** 
  - Imports `domain/metadata` (for `Product` type) and `domain/ingest` (for `MediaUnit`, `Stream`, `Layout`).
  - Returns DTOs (`ScriptInput`, `PredictedOutput`) and maps to the application layer.
- **Domain Events:** None. The domain is synchronous and returns values directly.
- **Dependencies:** 
  - `callname/translit` — internal sub-package, no external dependencies.
  - `domain/script` imports `callname` (for `PredictedOutput` type in `ScriptConfig`).

## 4. Ubiquitous Language

| Term | Definition | Synonyms/Notes |
|------|------------|----------------|
| **ScriptInput** | DTO representing the components of an input call name. | "input name", "source name" |
| **PredictedOutput** | A predicted output file produced by the rendering pipeline. | "prediction", "output file" |
| **CallName** | The self-documenting filename string derived from `ScriptInput.String()`. | "renamed filename", "call filename" |
| **PRT** | Pipeline Run Timestamp — a 12-digit UTC timestamp prefixed with "PRT" (e.g. `PRT260928200000`). | "pipeline timestamp", "run ID" |
| **Base** | The transliterated `SeriesTitle` of the product, first character uppercase. | "show name", "title base" |
| **SeasonTag** | Zero-padded season number (e.g. `"s01"`), or empty for movies. | "season identifier" |
| **EpisodeTag** | Zero-padded episode number (e.g. `"e06"`), or empty for movies/seasons. | "episode identifier" |
| **FileTags** | Per-file suffixes extracted from the original filename (e.g. `["rus", "Russkii", "Forsirovannye"]`). | "filename tags", "file descriptors" |
| **HSUB** | Hardsub file — a subtitle file whose name contains the configured hardsub tag. | "hard subtitle", "burned subtitle" |
| **layoutTag** | Channel-layout suffix used in output filenames ("20"=stereo, "51"=5.1, "70"=7.1, "10"=mono). | "audio layout", "channel suffix" |
| **Translit** | Cyrillic-to-Latin transliteration subsystem. Self-contained, no external dependencies. | "transliteration", "Cyrillic converter" |

## 5. Key Scenarios / Use Cases

### Scenario 1: Build an input call name for a series episode
- **Given** a `Product` with `SeriesTitle="Игра лицевых"`, `SeasonNumber=1`, `EpisodeNumber=26`
- **And** a source file `12345.mp4` with timestamp `260928200000`
- **When** `BuildInputName(product, "12345.mp4", "260928200000")` is called
- **Then** a `ScriptInput` is returned with `Base="Igra_licevyh"`, `SeasonTag="s01"`, `EpisodeTag="e26"`, `PRT="PRT260928200000"`
- **And** `String()` produces `"Igra_licevyh_s01e26_PRT260928200000_id12345.mp4"`

### Scenario 2: Build input names for a multi-file unit (video + external subtitle)
- **Given** a `Product` and filenames `["12345.mp4", "12345-rus_Russkii.srt"]`
- **When** `BuildInputNames(product, filenames, "260928200000")` is called
- **Then** a `map[string]string` is returned mapping each original filename to its call name
- **And** the video maps to `"Igra_licevyh_s01e26_PRT260928200000_id12345.mp4"`
- **And** the subtitle maps to `"Igra_licevyh_s01e26_PRT260928200000_id12345-rus_Russkii.srt"`

### Scenario 3: Predict output files for a series episode with multi-audio
- **Given** a `Product` (series, season 1, episode 1) and a `MediaUnit` with video, 2 audio streams (stereo Russian, 5.1 English), and 1 subtitle stream
- **When** `PredictOutputs(product, unit, "/mnt/ROOT/EDIT", "", 12345)` is called
- **Then** 4 `PredictedOutput` entries are returned:
  - Video: `/mnt/ROOT/EDIT/Show_s01/Show_s01_01_PRT261001120000_HD.mp4`
  - Audio 0: `/mnt/ROOT/EDIT/Show_s01/Show_s01_01_PRT261001120000_HD_AUDIO00_RUS20.m4a`
  - Audio 1: `/mnt/ROOT/EDIT/Show_s01/Show_s01_01_PRT261001120000_HD_AUDIO01_ENG51.m4a`
  - Subtitle: `/mnt/ROOT/EDIT/Show_s01/Show_s01_01_PRT261001120000_HD.srt`

### Scenario 4: Predict output files for a movie (no season/episode)
- **Given** a `Product` with `Kind="movie"`, `SeriesTitle="Movie Title"`
- **When** `PredictOutputs(product, unit, "/mnt/ROOT/EDIT", "", 999)` is called
- **Then** output paths contain `"Movie_title"` but **no** `_s01` or episode tags
- **And** the directory is `/mnt/ROOT/EDIT/Movie_title` (no season subdirectory)

### Scenario 5: Hardsub detection
- **Given** a `MediaUnit` with filenames `["12345.mp4", "12345-hardsub.srt"]`
- **When** `PredictOutputs(product, unit, "/mnt/ROOT/EDIT", "hardsub", 12345)` is called
- **Then** the video `PredictedOutput.HsubFile` is `"12345-hardsub.srt"`

### Scenario 6: Missing language defaults to "und"
- **Given** an audio stream with no `Language` field set
- **When** `PredictOutputs` processes the stream
- **Then** the output filename uses `"UND"` as the language suffix (e.g. `AUDIO00_UND20.m4a`)

### Scenario 7: Movie input without season/episode
- **Given** a movie `Product` with `SeriesTitle="Movie Title"` and file `999.mp4`
- **When** `BuildInputName(product, "999.mp4", "261001120000")` is called
- **Then** the resulting call name is `"Movie_title_PRT261001120000_id999.mp4"` (no `s`/`e` fragments)

### Scenario 8: PRT timestamp generation
- **When** `NewPRT()` is called
- **Then** a string like `"PRT261023153045"` is returned (current UTC time in `YYMMDDHHmmss` format)
- **And** the format matches the regex `^PRT\d{12}$`

## 6. High-Level Data Model (Key Entities)

- **ScriptInput** — DTO for building input call names.
  - Fields: `ID` (int64), `Base` (string, transliterated SeriesTitle), `SeasonTag` (string, e.g. `"s01"`), `EpisodeTag` (string, e.g. `"e06"`), `PRT` (string, pipeline run timestamp), `OriginalName` (string, source filename), `FileTags` ([]string, per-file descriptors).
  - Key invariants: `ID > 0`, `Base != ""` (unless empty SeriesTitle), `PRT` matches `^PRT\d{12}$`, `FileTags` derived from filename stem after ID.
  - `String()` assembles the call name: `{Base}_{SeasonTag}{EpisodeTag}_PRT{id}_id{ID}[-{FileTags}]{ext}`.

- **PredictedOutput** — DTO for a predicted rendering output file.
  - Fields: `Path` (string, full absolute path), `MediaType` (string: "video", "audio", "subtitle"), `Language` (string, IANA language tag), `Layout` (string, channel-layout suffix), `AudioIndex` (int, zero-based among audio outputs), `HsubFile` (string, hardsub subtitle filename for video).
  - Key invariants: `Path` uses forward slashes for cross-platform compatibility; `Layout` maps from `ingest.Layout` enum ("20"=stereo, "51"=5.1, "70"=7.1, "10"=mono).

- **PRT** — Pipeline Run Timestamp, a string matching `^PRT\d{12}$` (e.g. `"PRT260928200000"`). Generated by `NewPRT()` using UTC time.

- **Translit subsystem** (`callname/translit/`) — Cyrillic transliteration engine.
  - `String(input, ...TranslitterOption) string` — main entry point, transliterates input using configurable options.
  - `TranslitterOption` — functional options: `RegisterLow()` (lowercase, no title case), `RegisterHigh()` (uppercase), `KeepRegister()` (preserve case), `Short_SE_Form()` (convert `_1_sezon` → `_s1`), `KeepUnderscoreRepetitions()` (preserve consecutive underscores).
  - `literals.go` — default map converting Cyrillic glyphs to Latin (а→a, б→b, в→v, etc.) and punctuation to underscores.
  - `options.go` — functional option pattern for configuring transliteration behavior.

- **ExtractID** — pure function that parses the leading-digit product ID from a filename stem. Returns an error if no leading digits exist.

**Relationships:**
- ScriptInput 1—1 PredictedOutput (both derive from Product + MediaUnit)
- PredictedOutput 1—1 ingest.Stream (audio predictions map one-to-one from audio streams)
- PredictedOutput 0—1 PredictedOutput (video's HsubFile references another output's filename string)
- Product → ScriptInput.Base (Base is derived from Product.SeriesTitle via transliteration)

## 7. Non-Functional Requirements & Constraints

- **Authorization:** N/A — the domain runs as part of an automated pipeline with no user-facing access control.
- **Performance/Scale:** Designed for batch processing of hundreds to thousands of media units. Input naming is O(n) where n is the number of files per unit (typically ≤ 10). Output prediction iterates over streams once, sorting them O(m log m) where m is the number of streams (typically ≤ 20). Transliteration is O(k) where k is the length of the SeriesTitle (typically ≤ 100 characters).
- **Data:** No persistence. All types are DTOs passed through the pipeline and discarded. No PII is stored.
- **External constraints:** Depends on ffprobe JSON output format (via ingest.MediaUnit) and metadata JSON catalog format (via metadata.Product). The transliteration subsystem is self-contained and has no external dependencies beyond the Go standard library.

## 8. Acceptance Criteria

- [ ] `BuildInputName` correctly constructs call names for series episodes, seasons, and movies, with proper season/episode tag combinations.
- [ ] `BuildInputNames` maps all files in a MediaUnit, failing fast if any file produces an error.
- [ ] `extractFileTags` correctly parses tags from filename stems (e.g. `"12345-rus_Russkii"` → `["rus", "Russkii"]`), handles empty stems, missing tags, and trailing underscores.
- [ ] `ScriptInput.validate` rejects inputs with missing ID, empty Base, missing PRT, or invalid PRT format (not matching `^PRT\d{12}$`).
- [ ] `ScriptInput.String()` assembles the correct call name format: `{Base}_{SeasonTag}{EpisodeTag}_PRT{id}_id{ID}[-{file_tags}]{ext}`.
- [ ] `PredictOutputs` produces the correct number and type of outputs (1 video, N audio, 0/1 subtitle) for any MediaUnit.
- [ ] `PredictOutputs` sorts streams deterministically by `(FileIndex, Index)` before generating predictions.
- [ ] `PredictOutputs` correctly applies layout tags to audio filenames ("20", "51", "70", "10") based on `ingest.Layout`.
- [ ] `PredictOutputs` defaults missing audio language to `"und"` and uppercases it in the output filename.
- [ ] `PredictOutputs` correctly detects hardsub files via tag substring matching (case-insensitive, with legacy "hardsub" fallback).
- [ ] `PredictScriptName` generates the correct script filename path with forward slashes.
- [ ] `PredictOutputs` for movies omits season/episode tags from directory and filename.
- [ ] `deriveOutBase` correctly transliterates Cyrillic SeriesTitle to ASCII (e.g. "Игра лицевых" → "Igra_licevyh"), preserves Latin titles with first-character uppercase ("Test Show" → "Test_show").
- [ ] `deriveNameTag` returns correct season/episode tag combinations: "s01e06", "s01", "e06", or "" (movie).
- [ ] `ExtractID` parses leading digits from filenames, handles extensions, and rejects non-digit-prefixed names.
- [ ] `NewPRT` generates timestamps in `PRTYYMMDDHHmmss` format matching the regex `^PRT\d{12}$`.
- [ ] Translit subsystem: `String()` correctly transliterates Cyrillic titles, applies title case by default, squashes consecutive underscores, and respects functional options.
- [ ] Integration tests pass for all synthetic scenarios: series episode, movie, season-only, multi-audio, hardsub, empty base.

## 9. Open Questions

- [ ] Should `PredictOutputs` produce separate subtitle outputs per language stream, or is the current single-output approach (first subtitle stream only) sufficient?
- [ ] Should `deriveOutBase` handle edge cases where the SeriesTitle is entirely numeric or contains characters not covered by the transliteration map?
- [ ] The `ScriptInput` struct is used both as a builder (via `BuildInputName`) and as a DTO (via `String()`). Should these be separate types to clarify intent?
- [ ] Should `PredictScriptName` accept a `MediaUnit` parameter for the PRT timestamp, or keep the separate `timestamp` argument (which can be redundant with `unit.PRTTimestamp`)?
- [ ] The transliteration subsystem is duplicated from `script/translit` — the duplication was intentional for domain purity. Should it be moved to a shared `platform/translit` package instead?
- [ ] Should `PredictOutputs` include a prediction for the ready-file (the marker file written after rendering completes), or is that entirely within the Script domain's responsibility?
- [ ] The `prodID` parameter in `PredictOutputs` is used to append `_idN` to output base names. Should this be derived from `unit.ID` instead of passed separately?
