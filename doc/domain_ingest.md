# Bounded Context: Ingest

> **Status:** active  
> **Owner:** pipeline engineering  
> **Entropy level:** Medium

## 1. Purpose

The ingest domain transforms raw physical media files into structured, validated product units. It scans directories for media files, probes each file with ffprobe to extract stream metadata, groups files by product ID, and produces a `MediaUnit` — a coherent DTO representing one media product ready for downstream metadata resolution and script generation.

**Out of scope:** Product metadata lookup (belongs to the *Metadata* domain), rendering script generation (belongs to the *Script* domain), file delivery or notification, and any user-facing interface. The domain knows nothing about *what* the media is — only that it exists as files on disk with measurable properties.

## 2. Context & Environment

- **Users / Actors:** The automated pipeline is the sole actor. No human directly interacts with this domain; it is invoked by the pipeline orchestrator (`app.Pipeline.RunOnce`) which scans source directories and delegates to `Organizer.Ingest(dir)` per unit.
- **Adjacent domains and systems:** 
  - **Metadata** — resolves the product ID from the ingest unit into a `Product` record.
  - **Script** — consumes the ingest unit's streams and filenames to build rendering scripts.
  - **ffprobe (external)** — the only external system; its JSON output is the raw input to the domain.
  - **Filesystem (OS)** — the source of physical files; accessed through infrastructure, never by the domain.
- **Key business rules:** A product ID is extracted from the filename stem (digits at the start). All files within a single unit must share the same ID. Streams are parsed from ffprobe JSON and validated against quality thresholds (resolution, bitrate, duration consistency). Files that contain only non-media streams (e.g. attachment data) are silently filtered out.

## 3. Boundaries & Interactions

- **Inbound interactions:** 
  - `Organizer.Ingest(dir string) (MediaUnit, error)` — the domain port. Infrastructure scans a directory, probes files with ffprobe, and passes the raw JSON (`map[string][]byte`) to `NewMediaUnit()`.
  - `NewMediaUnit(map[string][]byte) (MediaUnit, error)` — pure function that parses, orders, and validates.
- **Outbound interactions:** 
  - Returns a `MediaUnit` DTO to the application layer.
  - The domain itself makes no external calls; all I/O is delegated to infrastructure.
- **Domain Events:** None. The domain is synchronous and returns a DTO directly.
- **Dependencies:** 
  - `domain/script` imports `domain/ingest` (`ScriptConfig.Streams []ingest.Stream`) — the only permitted cross-domain dependency.
  - `domain/metadata` does not depend on ingest.

## 4. Ubiquitous Language

| Term | Definition | Synonyms/Notes |
|------|------------|----------------|
| **MediaUnit** | Aggregated representation of one media product across one or more physical files. | "unit", "product unit" |
| **Stream** | A single media stream (video, audio, subtitle, or data) within a file. | "track" (informal) |
| **StreamKind** | Classification of a stream: `video`, `audio`, `subtitle`, or `data`. | "codec_type", "stream type" |
| **Coordinate** | ffmpeg-style stream selector string, e.g. `"[0:v:0]"`. | "stream selector", "ffmpeg key" |
| **Rational** | A fraction (numerator / denominator) used for frame rate and duration. | "fraction", "ratio" |
| **ValidationIssue** | A quality concern found during ingest (resolution, bitrate, duration, duplicates). | "warning", "alert" |
| **Organizer** | Domain port interface for scanning a directory into a MediaUnit. | "ingester" (informal) |
| **Product ID** | Numeric identifier extracted from the beginning of a filename stem. | "ID", "product number" |
| **File rank** | Ordering priority: video-containing=0, audio-only=1, subtitle-only=2, other=3. | "kind rank", "priority" |
| **Sidecar** | External subtitle file (`.srt`, `.ass`, `.ssa`) that ffprobe cannot probe. | "external subtitle" |

## 5. Key Scenarios / Use Cases

### Scenario 1: Ingest a single-file product
- **Given** a directory containing one media file (e.g. `63603.mp4`) with video and audio streams
- **When** `Organizer.Ingest(dir)` is called
- **Then** a `MediaUnit` is returned with `ID=63603`, one filename, and parsed streams (video + audio)
- **And** no validation issues are reported

### Scenario 2: Ingest a multi-file product (video + external subtitle)
- **Given** a directory containing `65003.mp4` and `65003-eng.srt`
- **When** `Organizer.Ingest(dir)` is called
- **Then** the `MediaUnit` includes both files in `Filenames` (video first, subtitle second)
- **And** the subtitle file is recorded as a sidecar (app layer handles ffprobe probing failure gracefully)
- **And** streams from both files are aggregated in the `Streams` map

### Scenario 3: Multi-audio unit with language tags
- **Given** a file with multiple audio tracks (e.g. English stereo + French 5.1)
- **When** streams are parsed from ffprobe JSON
- **Then** each audio stream carries its `Language` field (from the `tags.language` attribute)
- **And** each stream retains its `ChannelLayout` (stereo, 5.1, etc.)

### Scenario 4: Validation — low-resolution video
- **Given** a video stream with resolution below 720×576
- **When** `ValidateMediaUnit(streams)` is called
- **Then** a `ValidationIssue` with `Field="resolution"` is returned
- **And** the problem message describes the actual resolution (e.g. "320x240")

### Scenario 5: Validation — duration inconsistency across files
- **Given** a MediaUnit with streams whose durations vary by more than 10%
- **When** `ValidateMediaUnit(streams)` is called
- **Then** a `ValidationIssue` with `Field="duration_consistency"` is returned
- **And** the issue is flagged at file-level (`StreamIndex=-1`)

### Scenario 6: Data-only files are filtered
- **Given** a directory with a media file and a data-only file (e.g. `.dat` with no media streams)
- **When** `NewMediaUnit(data)` is called
- **Then** the data-only file is excluded from `Filenames` and its streams are not included
- **And** the MediaUnit is returned successfully (not an error)

### Scenario 7: Inconsistent product IDs across files
- **Given** files `63603.mp4` and `99999.mp4` in the same directory
- **When** `NewMediaUnit(data)` is called
- **Then** an error is returned indicating inconsistent IDs

## 6. High-Level Data Model (Key Entities)

- **MediaUnit** — aggregate root. Represents one product across its physical files.
  - Key invariants: all files share the same Product ID; `Filenames[0]` is the primary media file (highest file rank).
  - Fields: `ID` (int64), `Filenames` ([]string), `Streams` (map[string]Stream), `Validation` (ValidationIssues), `PRTTimestamp` (string, pipeline-provided).
  
- **Stream** — describes a single media stream.
  - Key invariants: `coordinate()` uniquely identifies a stream within a unit; duration is stored as a Rational (milliseconds).
  - Fields: `FileIndex`, `Index`, `Kind`, `Codec`, `Width`, `Height`, `FrameRate`, `PixelFormat`, `Interlaced`, `ChannelLayout`, `Language`, `SampleRate`, `SampleFormat`, `Forced`, `BitRate`, `Duration`.

- **StreamKind** — enum: `video`, `audio`, `subtitle`, `data`.

- **Rational** — fraction (Num/Denom). Used for frame rate and duration. Zero-value (`Num=0, Denom=0`) means "unavailable."

- **ValidationIssue** — a quality concern with `StreamIndex` (-1 for file-level, ≥0 for stream-level), `Field` (category), and `Problem` (human-readable description).

- **Organizer** — port interface: `Ingest(dir string) (MediaUnit, error)`. Implemented by infrastructure.

**Relationships:**
- MediaUnit 1—N Stream (streams map, keyed by coordinate string)
- MediaUnit 1—N ValidationIssue (issues list)
- MediaUnit 1—N File (Filenames slice)

## 7. Non-Functional Requirements & Constraints

- **Authorization:** N/A — the domain runs as part of an automated pipeline with no user-facing access control.
- **Performance/Scale:** Designed for batch processing of hundreds to thousands of media units. Stream parsing and validation are pure functions with no I/O or allocations beyond the input data. File ordering uses `slices.SortFunc` — O(n log n) where n is the number of files per unit (typically ≤ 10).
- **Data:** No persistence. MediaUnits are DTOs passed through the pipeline and discarded. No PII is stored.
- **External constraints:** Depends on ffprobe JSON output format (produced by `ffprobe -v quiet -print_format json -show_streams -show_format`). The domain assumes ffprobe is available as an external tool but never invokes it directly.

## 8. Acceptance Criteria

- [ ] `ExtractID` correctly parses product IDs from filenames with various formats (numeric, underscore suffix, hyphen suffix, no extension) and rejects non-digit-prefixed stems.
- [ ] `ParseStreams` correctly maps ffprobe JSON to domain `Stream` objects for video, audio, and subtitle streams.
- [ ] `ValidateMediaUnit` detects all validation rules: resolution below SD, low bitrate (video <100 kbps, audio <32 kbps), negative duration, duration inconsistency (>10%), and duplicate coordinates.
- [ ] `NewMediaUnit` orders files by kind rank (video > audio > subtitle > data), filters data-only files, assigns `FileIndex`, and validates ID consistency.
- [ ] `sortUnitFiles` produces deterministic ordering: video-containing files first, lexicographic within each rank group.
- [ ] `Stream.coordinate()` produces correct ffmpeg-style selectors (e.g. `[0:v:0]`, `[1:a:2]`).
- [ ] Subtitle sidecar files (`.srt`, `.ass`, `.ssa`) are handled gracefully by infrastructure (probing failure is not a fatal error).
- [ ] Integration tests pass for all synthetic test scenarios (simple, multi-audio, subtitles, multi-lang, embedded-sub, hardsub).

## 9. Open Questions

- [ ] Should `ValidateMediaUnit` enforce a minimum audio sample rate (e.g. 44100 Hz)? Currently only bitrate is checked.
- [ ] Should the domain expose a `MediaUnit.WithPrimaryFile() Stream` convenience method, or should the application layer derive this from `Streams` by selecting the first video stream?
- [ ] The `PRTTimestamp` field on `MediaUnit` is set by the pipeline, not the domain. Should this field live in the domain DTO or be carried separately by the application layer?
- [ ] Should `Interlaced` be used by downstream domains (currently marked "not used yet"), or should it be removed to keep the model lean?
- [ ] Should `Rational` support arbitrary-precision arithmetic for edge-case frame rates (e.g. 29.97 as 30000/1001)? Current implementation uses `int` which may lose precision for very large denominators.
