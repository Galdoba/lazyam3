# Ingest Domain

The ingest domain handles physical media files: extracting product IDs, parsing stream metadata from ffprobe output, and assembling coherent MediaUnits.

## Overview

```
Directory → [filter files] → [probe each file] → [parse streams] → MediaUnit
```

The ingest domain transforms raw ffprobe JSON data into structured domain types. It knows nothing about how the data was acquired — only that it arrived as ffprobe JSON output.

## Domain Layer

### Types

**`MediaUnit`** — aggregated product info from one or more files on disk.

```go
type MediaUnit struct {
    ID         int64             // Product ID extracted from filenames
    Filenames  []string          // Files ordered by kind rank (video, audio, subtitle, other; lexicographic within a group); Filenames[0] is the primary media file
    Streams    map[string]Stream // Keyed by stream coordinate (e.g. "[0:v:0]")
    Validation ValidationIssues  // Problems found during ingest
}
```

**`Stream`** — describes a single media stream within a file.

```go
type Stream struct {
    FileIndex int           // Which file in the unit (0-based)
    Index     int           // Stream index within the file
    Kind      StreamKind    // video, audio, subtitle, data
    Codec     string        // e.g. "h264", "aac", "mov_text"
    // Video fields
    Width, Height int
    FrameRate   Rational
    PixelFormat string
    Interlaced  bool        //not used yet
    // Audio fields
    ChannelLayout Layout
    Language      Language
    SampleRate    int
    SampleFormat  string
    // Subtitle fields
    Forced bool
    // Common
    BitRate  int64
    Duration Rational
}
```

**`StreamKind`** — identifies stream type.

```go
const (
    StreamKindVideo    StreamKind = "video"
    StreamKindAudio    StreamKind = "audio"
    StreamKindSubtitle StreamKind = "subtitle"
    StreamKindData     StreamKind = "data"
)
```

**`Rational`** — represents fractional values (frame rate, duration).

```go
type Rational struct {
    Num   int
    Denom int
}
```

**`ValidationIssue`** — records a problem found during ingest.

```go
type ValidationIssue struct {
    StreamIndex int    // -1 if file-level, >= 0 if stream-level
    Field       string // e.g. "resolution", "bitrate", "duration_consistency"
    Problem     string // human-readable description
}
```

### Pure Functions

**`ExtractID(filename string) (int64, error)`**

Parses the product ID from a filename stem (name without extension). The stem must start with a digit sequence.

```go
ExtractID("63603.mp4")        // → 63603, nil
ExtractID("63604_PRT251205.mp4") // → 63604, nil
ExtractID("Copy of 63603.mp4")   // → 0, error (doesn't start with digit)
```

**`ParseStreams(ffprobeJSON []byte) ([]Stream, error)`**

Parses ffprobe JSON output into domain `Stream` objects. The JSON must match the format produced by:

```bash
ffprobe -v quiet -print_format json -show_streams -show_format <file>
```

Only streams with `codec_type` of `video`, `audio`, or `subtitle` are returned. Other types (data, attachment) are mapped to `StreamKindData`.

**`ValidateMediaUnit(streams []Stream) ValidationIssues`**

Runs domain-level validation rules and returns a list of issues:

| Rule | Threshold | When triggered |
|---|---|---|
| Resolution | ≥ 720×576 (SD) | Video resolution below SD |
| Video bitrate | ≥ 100 kbps | Suspiciously low bitrate |
| Audio bitrate | ≥ 32 kbps | Suspiciously low bitrate |
| Duration | ≥ 0 | Negative duration for video/audio |
| Duration consistency | ≤ 10% variance | Streams' durations vary by more than 10% |
| Duplicate detection | Unique coordinates | Same stream coordinate appears twice |

**`Stream.Coordinate() string`**

Returns the ffmpeg-style stream selector string.

```go
Stream{FileIndex: 0, Index: 0, Kind: "video"}.Coordinate() // → "[0:v:0]"
Stream{FileIndex: 1, Index: 2, Kind: "audio"}.Coordinate() // → "[1:a:2]"
```

### `NewMediaUnit(data map[string][]byte) (MediaUnit, error)`

The main entry point. Takes raw ffprobe JSON data keyed by filename and produces a `MediaUnit`.

**Process:**
1. Parse streams from ffprobe JSON for every file
2. Order files by kind rank: video-containing → audio-only → subtitle-only → other;
   lexicographic within a group (`sortUnitFiles`). `Filenames[0]` is therefore the
   primary media file whenever the unit contains one — alphabetical order alone is
   not enough (e.g. `12094-rus.srt` sorts before `12094.mp4`)
3. **Filter**: skip files with only `data` streams (not media files)
4. Assign `FileIndex` to each stream based on the ordered position
5. Build coordinate keys and collect streams into a map
6. Extract product ID from first filename (validate consistency across files)
7. Run `ValidateMediaUnit` and store issues

**Error cases:**
- Invalid ffprobe JSON → returns error with filename
- Inconsistent IDs across files → returns error
- All files filtered out → returns empty `MediaUnit` (not an error)

## Implementation Layer

### `Organizer` Interface

```go
type Organizer interface {
    Ingest(dir string) (MediaUnit, error)
}
```

Defines the contract for turning a directory of files into a `MediaUnit`.

### `app/ingest.Ingest` Implementation

```go
type Ingest struct {
    ffprobeBinary string
}

func New(ffprobeBinaryPath string) *Ingest
func (ing *Ingest) Ingest(dir string) (MediaUnit, error)
```

**Process:**
1. Read directory entries with `os.ReadDir`
2. **Filter**: skip non-regular files (directories, symlinks, etc.)
3. Create a single `Prober` instance for reuse
4. For each regular file:
   - Probe with ffprobe (`fso.Prober.ProbeFile(path)`)
   - **Fail fast**: if probing fails, return error immediately
   - Collect raw JSON into `map[string][]byte`
5. Pass collected data to `domain.ingest.NewMediaUnit`

**Error handling:**
- Directory read error → returned directly
- Probe failure → wrapped error (`failed to probe file "path": <cause>`)
- Parsing/validation errors → returned from `NewMediaUnit`

### CLI Integration

```bash
lazyam3 process \
  --source-root /path/to/media \
  --target-root /path/to/output \
  --ffprobe-path /usr/bin/ffprobe \
  --once
```

The `--ffprobe-path` flag configures which ffprobe binary to use. Fallback chain: flag → config field `paths.ffprobe_binary` → `ffprobe` (from `$PATH`). The `--once` flag runs a single scan pass and exits (useful for live testing). Without `--once`, the pipeline polls every second until SIGINT/SIGTERM.

## Test Data

Synthetic test files in `testdata/synthetic/` cover common scenarios:

| Product | Files | Streams | Purpose |
|---|---|---|---|
| **10001** (simple-unit) | `10001.mp4` (88K) | 1 video (h264), 1 audio (aac) | Basic case |
| **10002** (multi-audio) | `10002.mp4` (105K) | 1 video, 2 audio (eng, fra) | Multiple audio tracks |
| **10003** (with-subtitles) | `10003.mp4` + `.srt` (180K) | 1 video, 1 audio, external srt | External subtitle file |
| **10004** (multi-lang-audio) | `10004.mp4` + `.srt` (236K) | 1 video, 6 audio (3 langs × stereo/5.1) | Complex audio routing |
| **10005** (embedded-sub) | `10005.mp4` (88K) | 1 video, 1 audio, 1 subtitle stream | Embedded subtitle track |

All files are 2 seconds, SD (720×576), 500kbps video / 64kbps audio. Used by integration tests.

## Validation Rules

### Resolution Check
Video streams must have resolution ≥ 720×576 (SD). Files with lower resolution trigger:
```
Field: "resolution"
Problem: "video resolution 320x240 is below SD (720x576)"
```

### Bitrate Sanity
Video streams below 100 kbps and audio below 32 kbps are flagged:
```
Field: "bitrate"
Problem: "video bitrate 50000 bps is suspiciously low"
```

### Duration Checks
- Video/audio streams with negative duration trigger:
  ```
  Field: "duration"
  Problem: "video has negative duration"
  ```
- Subtitle streams may have zero duration (normal for SRT)

### Duration Consistency
When a MediaUnit has multiple streams with durations, they must be within 10% of each other. This catches mismatched or corrupted split files:
```
Field: "duration_consistency"
Problem: "stream durations vary by more than 10%"
```

### Duplicate Detection
Each stream's coordinate (file_index + kind + index) must be unique. Duplicates trigger:
```
Field: "coordinate"
Problem: "duplicate stream coordinate"
```

## File Structure

```
internal/domain/ingest/
├── port.go         # Organizer interface (11 lines)
├── unit.go         # MediaUnit, ValidationIssue, NewMediaUnit, ValidateMediaUnit
├── stream.go       # Stream, StreamKind, Rational, Layout, Language
├── parse.go        # ParseStreams, helper functions
├── extract.go      # ExtractID
└── unit_test.go    # 70 unit tests (12 test functions)

internal/app/ingest/
├── ingest.go       # Ingest struct, New, Ingest method (real implementation)
└── ingest_test.go  # Integration tests with synthetic testdata
```
