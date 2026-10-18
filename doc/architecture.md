# lazyam3 Architecture

## Overview

lazyam3 is a media processing pipeline tool that scans input directories for media files, resolves product metadata, and generates rendering scripts (e.g., ffmpeg commands for extracting audio tracks).

The architecture follows **Domain-Driven Design (DDD)** principles with a clean separation between domain logic, application orchestration, and infrastructure concerns.

## Architecture Layers

```
┌───────────────────────────────────────────────────────────────┐
│  Presentation Layer                                           │
│  cmd/lazyam3 (wiring), internal/infra/commands (CLI)          │
│  internal/presentation/tui (bubbletea TUI, not wired yet)     │
└──────────────────────────┬────────────────────────────────────┘
                           │ uses
                           ▼
┌───────────────────────────────────────────────────────────────┐
│  Application Layer (internal/app/)                            │
│  Pipeline orchestrator, use cases, AppModules                 │
│  Works exclusively with DTOs and interfaces                   │
└───────┬───────────────────────────────┬───────────────────────┘
        │ calls                         │ calls
        ▼                               ▼
┌─────────────────┐ ┌───────────────────┐ ┌─────────────────────┐
│ domain/ingest   │ │ domain/metadata   │ │ domain/script       │
│ MediaUnit       │ │ ProductRepository │ │ Script, Text        │
│ Stream          │ │ Root, Found, Kind │ │ ScriptConfig        │
│ IngestPort      │ │ Descendants       │ │ StreamSelection     │
└────────┬────────┘ └─────────┬─────────┘ │ Renderer            │
         │                    │           └──────────┬──────────┘
         │                    │                      │ imports
         │                    │            ┌─────────▼──────────┐
         │                    │            │ domain/ingest      │
         │                    │            │ (Stream in         │
         │                    │            │  ScriptConfig)     │
         │                    │            └────────────────────┘
         │ implements         │ implements
         ▼                    ▼
┌───────────────────────────────────────────────────────────────┐
│  Infrastructure Layer (internal/infra/)                       │
│  Concrete implementations of domain ports                     │
│  - app/ingest.Ingest (stub) implements IngestPort             │
│  - metadata.Repository implements ProductRepository           │
│  - fs.ScriptWriter implements Renderer                        │
│  - ffprobe.Client / ProbeClient (probing support)             │
│  Also: config, logging, cache, persistence/jsonstore,         │
│  commands (CLI entry points)                                  │
└───────────────────────────────────────────────────────────────┘
```

## Domain Layer

The domain layer contains pure business logic with zero dependencies on external packages (only the Go standard library). It defines **ports** (interfaces) that application/infrastructure code implements.

### Domain Packages

| Package | Responsibility | Key Types |
|---------|---------------|-----------|
| `internal/domain/ingest` | Physical file identity and stream description | `MediaUnit`, `Stream`, `StreamKind`, `Layout`, `Language`, `Rational`, `ValidationIssue`, `IngestPort` |
| `internal/domain/metadata` | Product metadata from counterparty | `Root`, `Found`, `Kind`, `Descendants`, `SeriesItem`, `SeasonsItem`, `EpisodesItem`, `MoviesItem`, `ProductRepository` |
| `internal/domain/script` | Rendering script generation | `Script`, `Text`, `ScriptConfig`, `StreamSelection`, `Renderer` |

### Domain Port Interfaces

```go
// domain/ingest/port.go
type IngestPort interface { Ingest(dir string) ([]MediaUnit, error) }

// domain/metadata/port.go
type ProductRepository interface { FindByID(id int64) (*Found, error) }

// domain/script/port.go
type Renderer interface { Write(dest string, content Text) error }
```

### Pure Functions

```go
// domain/ingest
func ExtractID(filename string) (int64, error)                        // "63603.mp4" → 63603
func ParseStreams(ffprobeJSON string) ([]Stream, error)               // ffprobe JSON → []Stream
func ValidateMediaUnit(streams []Stream) ValidationIssues             // domain validation rules
func (s Stream) Coordinate() string                                   // ffmpeg selector "[0:a:1]"

// domain/metadata
func (r *Root) FindByID(id int64, d Descendants) (*Found, bool)       // tree search, returns self-contained copy
```

### Design Principles

- **Domains are pure** — no imports from infrastructure or application layers
- **Ports define contracts** — implementations live in application/infrastructure
- **Types are data** — `MediaUnit`, `Found`, `Script` are data structures; behavior lives in package-level functions
- **Search logic stays in domain** — `Root.FindByID()` is a domain method, not infrastructure
- **One accepted cross-domain dependency** — `domain/script` imports `domain/ingest` (`ScriptConfig.Files []ingest.Stream`). `domain/ingest` and `domain/metadata` import no other domain.

## Application Layer

The application layer orchestrates domain services to accomplish use cases. It knows nothing about how domains work internally.

### Pipeline

```go
// internal/app/pipeline
type Pipeline struct {
    Ingest ingest.IngestPort
}

func NewPipeline(in ingest.IngestPort) *Pipeline
func (p *Pipeline) Run(input RunInput) (RunOutput, error)
```

**Responsibilities (current):**
- Calls `Ingest.Ingest(sourceDir)` and returns `FilesProcessed`
- Metadata resolution, script generation, and rendering are TODO (tasks 2–8)

**What Pipeline does NOT do:**
- Does not scan files directly (delegates to `IngestPort`)
- Does not query metadata directly (delegates to `ProductRepository` — not yet wired)
- Does not generate script content (delegates to domain/script — not yet wired)
- Does not write files directly (delegates to `Renderer` — not yet wired)

### Use Cases

```go
// internal/app/ingest
type Ingest struct{}
func (s *Ingest) Ingest(dir string) ([]ingest.MediaUnit, error)  // stub: returns nil, nil

// internal/app/script
func Build(ctx context.Context, config script.ScriptConfig, renderer script.Renderer) error
// currently writes a placeholder Text via Renderer
```

### AppModules

```go
// internal/app
type AppModules struct {
    Config config.Config
    Logger *logging.Logger
    // lazy: Cache(), Store()
}
```

Shared dependency container. Only `Config` and `Logger` are eagerly initialized (in root command `Before`); `Cache` and `Store` are created on first access. `Shutdown` closes what was created.

### DTOs

| Type | Purpose |
|------|---------|
| `RunInput` | Runtime parameters from CLI: `SourceDir`, `Destination`, `DryRun` |
| `RunOutput` | Results returned to CLI: `FilesProcessed`, `ScriptWritten` |

## Infrastructure Layer

Infrastructure provides concrete implementations of domain ports and the outside-world integrations (filesystem, ffprobe, config, logging). It depends on domains but domains know nothing about infrastructure.

### Infrastructure Packages

| Package | Contents | Depends On |
|---------|----------|------------|
| `internal/infra/fs` | `ScriptWriter` (implements `script.Renderer`) | `domain/script` |
| `internal/infra/ffprobe` | `Client` (runs ffprobe, `RawData`), `Mapper` (stub), `ProbeClient` (`[]ingest.Stream`) | `domain/ingest` |
| `internal/infra/metadata` | `Repository` (implements `ProductRepository`) | `domain/metadata` |
| `internal/infra/config` | `Config` struct, `Load()` via appcontext configmanager | — |
| `internal/infra/logging` | `Logger` (slog facade) | — |
| `internal/infra/cache` | in-memory cache | — |
| `internal/infra/persistence/jsonstore` | JSON key-value store | — |
| `internal/infra/commands` | `CMDProcess` (CLI command; wires app + domain) | `app`, `domain/ingest` |

### Key Implementation: metadata.Repository

```go
type Repository struct {
    mu   sync.RWMutex
    root *metadata.Root
}
```

**Workflow:**
1. `New(path)` — reads and unmarshals `metadata.json` into `Root`
2. `FindByID(id)` — delegates to `Root.FindByID(id, WithDescendants)`
3. Returns `*Found` with a self-contained copy of the product tree

### Current Ingest Path

The real scan/probe/parse implementation does not exist yet. Today:

1. `commands.CMDProcess` validates `--source`
2. Creates `app/ingest.Ingest` (stub) and `pipeline.NewPipeline`
3. `Pipeline.Run` calls `Ingest.Ingest(sourceDir)` → `nil`
4. Returns `RunOutput{FilesProcessed: 0}`

`infra/ffprobe.ProbeClient` exists but is not yet wired into any ingest implementation.

## Presentation Layer

CLI commands are the entry point. Wiring lives in `cmd/lazyam3/main.go`:

1. `app.New()` — empty `AppModules` container
2. Root command `Before`: `modules.Init(configPath)` — loads config, creates logger
3. Subcommand `commands.CMDProcess()` runs the pipeline
4. Root command `After`: `modules.Shutdown()` — closes store/logger

### Command Pattern

```go
func CMDProcess() *cli.Command {
    return &cli.Command{
        Name:  "process",
        Flags: []cli.Flag{ /* --source, --output */ },
        Action: cmdProcessAction,
    }
}

func cmdProcessAction(ctx context.Context, cmd *cli.Command) error {
    // 1. Validate flags (--source required)
    // 2. Create dependencies (app/ingest.Ingest stub, Pipeline)
    // 3. Run pipeline
    // 4. Output results
}
```

`internal/presentation/cli` is currently empty. `internal/presentation/tui` contains a bubbletea TUI model that is not yet wired into the binary.

## Dependency Rules

```
cmd                → app, infra, domain        (wiring only)
infra/commands     → app, domain               (presentation entry points)
app                → domain                    (never infra)
infra              → domain                    (implements ports)
domain             → (script → ingest only)
platform           → nobody                    (stateless services)
```

**Rules:**
- **Domains have zero external dependencies** — pure Go, no `os`/`net`/YAML/TOML
- **Application depends on domain interfaces** — never on infrastructure (exception: `infra/commands` currently lives in infra while acting as presentation wiring)
- **Infrastructure depends on domains** — implements domain ports
- **cmd is the only full-wiring place** — creates and wires dependencies
- **Cross-domain imports** — only `domain/script → domain/ingest` is allowed; all others are forbidden

## Configuration

Intended canonical format: **TOML**. The config subsystem is currently a stub and will be finalized once all domain implementations are done (tracked as task 10 in `doc/tasks.json`).

Current state (`internal/infra/config`):

```go
type Config struct {
    Paths   Paths   // InputDirectory, OutputDirectory, MetadataPath
    Logger  Logging // Level, Path, Terminal
    Storage Storage // Path (jsonstore)
}
```

- Loaded via `github.com/Galdoba/appcontext/configmanager` with `WithFormat(configmanager.JSON)` — a minimal/stub selection, not the intended format
- Defaults are hardcoded (`defaultCFG()`); app name constant is `lazyam3`
- `configs/config.yaml` and `configs/config.local.yaml` are empty placeholders

## Workflow

```
1. cmd/lazyam3 starts
2. AppModules created (empty)
3. Root Before: config loaded, logger created (modules.Init)
4. commands.CMDProcess action:
   a. Validate --source flag
   b. Create app/ingest.Ingest (stub) and Pipeline
   c. Pipeline.Run(input):
      - Ingest.Ingest(sourceDir) → []MediaUnit   (currently nil)
      - TODO: metadata resolution, script generation, rendering
   d. Print "processed N files"
5. Root After: modules.Shutdown() closes store and logger
```

## Development Guidelines

When adding new functionality:

1. **Identify the domain** — Which domain owns this concept? (ingest, metadata, script)
2. **Define the pure logic** — Add types and pure functions to `internal/domain/<name>/`
3. **Define the port** — Add interface to `domain/<name>/port.go` if new IO is needed
4. **Implement** — Create concrete type in `internal/infra/<name>/` or use case in `internal/app/<name>/`
5. **Wire in application** — Add to `Pipeline` as interface dependency
6. **Use in command** — Command creates implementation and passes to Pipeline
7. **Never leak infrastructure** — Domains must not know about files, APIs, or databases
8. **Update docs and tasks** — Keep `doc/lazyam3.md`, `doc/architecture.md`, and `doc/tasks.json` in sync with the code
