---
tags:
  - project
  - tool
---

## Workflow

1. Scan input directory for media files
2. Collect technical metadata — run ffprobe to extract stream information
3. Extract product ID from filename (`63603.mp4` → `63603`)
4. Resolve content metadata — look up product info by ID from counterparty's metadata
5. Build processing script — generate bash/ffmpeg commands based on streams and metadata
6. Render script to output directory

**Implementation status (task 1 complete):** the ingest domain is defined with types, pure functions, and 24 unit tests. `Pipeline.Run` currently performs ingest only; `app/ingest` is a stub. Metadata resolution, script generation, and rendering are wired at the type/port level but not yet connected (see `doc/tasks.json`).

## Domain Modules

### Ingest (physical files)

Language of physical files and streams. Pure data and functions — no IO.

Types:
- `MediaUnit` — aggregated product info: `ID`, `Filepaths`, `Streams`, `Validation`
- `Stream` — single stream: kind, codec, video/audio/subtitle/data attributes
- `StreamKind` (`video`, `audio`, `subtitle`, `data`), `Layout`, `Language`, `Rational`
- `ValidationIssue` / `ValidationIssues` — problems found during ingest

Pure functions:
- `ExtractID(filename)` — product ID from filename (`63603.mp4` → `63603`)
- `ParseStreams(ffprobeJSON)` — ffprobe JSON output → `[]Stream`
- `ValidateMediaUnit(streams)` / `ValidateMediaUnitWithFilepaths(unit)` — domain validation rules
- `Stream.Coordinate()` — ffmpeg stream selector (`[0:a:1]`, `[1:s:0]`)

Port: `IngestPort` — `Ingest(dir string) ([]MediaUnit, error)`. Implemented in `app/ingest` (currently a stub; real scan/probe/parse logic is task 2).

### Metadata (product catalog)

Works with media product metadata from the counterparty.

Definitions:
- **id** — unique identifier written to the file name. Can refer to a series, season, series episode, movie, or movie episode.
- **Kind** — type of object in the catalog that has an `id`.

```go
type Kind string

const (
	KindMovie         Kind = "movie"
	KindMovieEpisode  Kind = "movie.episode"
	KindSeries        Kind = "series"
	KindSeason        Kind = "series.season"
	KindSeriesEpisode Kind = "series.season.episode"
)
```

Types: `Root`, `SeriesItem`, `SeasonsItem`, `EpisodesItem`, `MoviesItem`, `PersonsItem`, `Found`, `Descendants`.

Implemented:
- Tree search: `Root.FindByID(id, WithDescendants)` returns a self-contained `*Found` copy
- Port: `ProductRepository` — `FindByID(id int64) (*Found, error)`

JSON parsing of `metadata.json` currently lives in `infra/metadata` (`json.Unmarshal` into `Root`).

### Script (rendering)

Language of rendering script generation.

Types: `Script`, `Text` (script body value object), `ScriptConfig`, `StreamSelection`.

Port: `Renderer` — `Write(dest string, content Text) error`. Implemented by `infra/fs.ScriptWriter`.

Logic: "Pass config (DTO) → receive file text". The pure builder (`BuildScript`) is not implemented yet (task 5). Note: `ScriptConfig.Files` is `[]ingest.Stream` — the script domain currently imports the ingest domain.

---

## Repository Tree

```
.
├── cmd/
│   └── lazyam3/
│       └── main.go                 # entry point: AppModules → config/logger → commands
│
├── configs/
│   ├── config.yaml                 # placeholder — real format will be TOML (task 10)
│   └── config.local.yaml
│
├── doc/
│   ├── architecture.md             # architecture overview and development guidelines
│   ├── lazyam3.md                  # this file
│   └── tasks.json                  # implementation tasks and status
│
├── internal/
│   ├── app/                        # APPLICATION LAYER: orchestration, works on ports and DTOs
│   │   ├── app.go                  # AppModules: shared deps (Config, Logger; lazy Cache, Store)
│   │   ├── ingest/
│   │   │   └── ingest.go           # Ingest — stub implementing domain ingest.IngestPort
│   │   ├── pipeline/
│   │   │   ├── dto.go              # RunInput / RunOutput — DTOs at presentation ↔ app boundary
│   │   │   └── pipeline.go         # Pipeline{Ingest}; Run = ingest only (TODO: metadata, script)
│   │   └── script/
│   │       └── build.go            # Build — use case; currently writes placeholder text
│   │
│   ├── domain/                     # DOMAIN LAYER: models and rules. Pure Go. No os/net.
│   │   ├── ingest/                 # language of physical files and streams
│   │   │   ├── unit.go             # MediaUnit, ValidationIssue(s), ValidateMediaUnit*
│   │   │   ├── stream.go           # Stream, StreamKind, Layout, Language, Rational, Coordinate
│   │   │   ├── parse.go            # ParseStreams: ffprobe JSON → []Stream
│   │   │   ├── extract.go          # ExtractID: filename → product ID
│   │   │   ├── port.go             # port: IngestPort
│   │   │   └── unit_test.go        # 24 unit tests
│   │   │
│   │   ├── metadata/               # language of media products from counterparty
│   │   │   ├── types.go            # Root, Found, Kind, Descendants, Series/Season/Episode/Movies/Persons
│   │   │   ├── search.go           # Root.FindByID(id, Descendants) — tree search logic
│   │   │   └── port.go             # port: ProductRepository
│   │   │
│   │   ├── script/                 # language of rendering script generation
│   │   │   ├── script.go           # Script, StreamSelection
│   │   │   ├── text.go             # type Text string — value object with script body
│   │   │   ├── config.go           # ScriptConfig — domain input (uses ingest.Stream)
│   │   │   └── port.go             # port: Renderer
│   │
│   ├── infra/                      # PORT IMPLEMENTATIONS. Knows about the outside world. Depends on domain.
│   │   ├── cache/
│   │   │   └── cache.go            # in-memory cache
│   │   ├── commands/
│   │   │   └── cmd_process.go      # process command: wires app/ingest → Pipeline
│   │   ├── config/
│   │   │   └── config.go           # Config via appcontext configmanager (JSON stub; TOML planned)
│   │   ├── ffprobe/
│   │   │   ├── client.go           # runs ffprobe -show_streams -print_format json → RawData
│   │   │   ├── mapper.go           # Mapper.MapStream → domain ingest.Stream (stub)
│   │   │   └── probe_client.go     # ProbeClient: ffprobe → []ingest.Stream
│   │   ├── fs/
│   │   │   └── script_writer.go    # ScriptWriter implements script.Renderer
│   │   ├── logging/
│   │   │   └── logger.go           # structured logging (slog facade)
│   │   ├── metadata/
│   │   │   └── repository.go       # implements ProductRepository (loads metadata.json)
│   │   └── persistence/
│   │       └── jsonstore/          # JSON key-value store
│   │
│   ├── platform/                   # STATELESS SERVICES. No domain dependencies.
│   │   ├── clock/
│   │   │   └── clock.go            # time abstraction
│   │   └── retry/
│   │       └── retry.go            # backoff, jitter
│   │
│   └── presentation/
│       ├── cli/                    # empty — CLI commands currently live in infra/commands
│       └── tui/
│           └── tui.go              # bubbletea TUI model (not wired into cmd yet)
│
├── scrips/                         # standalone helper tools (note: directory name is misspelled)
│   ├── asses/                      # JSON querying tool (gjson)
│   └── find/
│
├── testdata/
│   ├── metadata.json               # product catalog snapshot
│   └── examples/                   # ffprobe output dumps and sample media
│
├── .golangci.yml
├── AGENTS.md
├── go.mod
├── go.sum
└── README.md
```

## Dependency Notes

**Dependency direction**
```
cmd                → app, infra, domain (only place that wires dependencies)
infra/commands     → app, domain (acts as CLI presentation entry point)
app                → domain
infra              → domain (implements ports)
platform           → no domain dependencies
domain             → (ingest, metadata: no internal deps; script → ingest)
```

**Who doesn't know whom**
- `domain/ingest` doesn't know `domain/metadata` or `domain/script`.
- `domain/metadata` doesn't know any other domain.
- `domain/script` imports `domain/ingest` (`ScriptConfig.Files []ingest.Stream`) — current, accepted cross-domain dependency.
- `infra/*` only knows about the ports of its corresponding domain (plus `infra/commands`, which wires app and domain together).

**Where DTOs, where domain types**

| Boundary | What's passed |
|---|---|
| presentation ↔ app | DTOs (`RunInput`, `RunOutput`) |
| app ↔ domain | domain types (`MediaUnit`, `Found`, `ScriptConfig`, `Text`) |
| app ↔ infra | domain types through ports |
| domain → domain | only `script → ingest` (via `ScriptConfig`); otherwise none |

**Where `os`, `net`, `database/sql`**
Only in `internal/infra/*` and `internal/platform/*`.

**Where loops, port calls, filtering**
Only in `internal/app/*` (Pipeline orchestrator, use cases).

**Where pure rules and models**
Only in `internal/domain/*`.
