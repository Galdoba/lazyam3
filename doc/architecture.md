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
│ MediaUnit       │ │ ProductRepository │ │ Script              │
│ Stream          │ │ Root, Found, Kind │ │ ScriptConfig        │
│ Organizer       │ │ Descendants       │ │ StreamSelection     │
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
│  - app/ingest.Ingest implements Organizer             │
│  - metadata.Repository implements ProductRepository           │
│  - script.Renderer: infra/script (usable artifact: executable file) │
│  - fso.Prober (ffprobe execution and JSON parsing)             │
│  Also: config, logging, cache, persistence/jsonstore,         │
│  commands (CLI entry points)                                  │
└───────────────────────────────────────────────────────────────┘
```

## Domain Layer

The domain layer contains pure business logic with zero dependencies on external packages (only the Go standard library). It defines **ports** (interfaces) that application/infrastructure code implements.

### Domain Packages

| Package | Responsibility | Key Types |
|---------|---------------|-----------|
| `internal/domain/ingest` | Physical file identity and stream description | `MediaUnit`, `Stream`, `StreamKind`, `Layout`, `Language`, `Rational`, `ValidationIssue`, `Organizer` |
| `internal/domain/metadata` | Product metadata from counterparty | `Product`, `Kind`, `ProductRepository` |
| `internal/domain/script` | Rendering script generation | `Script`, `ScriptConfig`, `StreamSelection`, `Renderer` |

### Domain Port Interfaces

```go
// domain/ingest/port.go
type Organizer interface { Ingest(dir string) (MediaUnit, error) }

// domain/metadata/port.go
type ProductRepository interface { FindByID(id int64) (*Product, error) }

// domain/script/port.go
type Renderer interface { Render(sc Script) error }
```

### Pure Functions

```go
// domain/ingest
func ExtractID(filename string) (int64, error)                        // "63603.mp4" → 63603
func ParseStreams(ffprobeJSON []byte) ([]Stream, error)               // ffprobe JSON → []Stream
func ValidateMediaUnit(streams []Stream) ValidationIssues             // domain validation rules
func (s Stream) Coordinate() string                                   // ffmpeg selector "[0:a:1]"

// domain/metadata — no pure functions: types + ProductRepository port only
// (JSON tree search lives in infra/metadata: Root.FindByID(id, Descendants) → *Found)
```

### Design Principles

- **Domains are pure** — no imports from infrastructure or application layers
- **Ports define contracts** — implementations live in application/infrastructure
- **Types are data** — `MediaUnit`, `Found`, `Script` are data structures; behavior lives in package-level functions
- **Logic stays in domain** — `ValidateMediaUnit()` rules are domain functions, not infrastructure
- **One accepted cross-domain dependency** — `domain/script` imports both `domain/ingest` and `domain/metadata` (`ScriptConfig.Streams []ingest.Stream`). `domain/ingest` and `domain/metadata` import no other domain.

## Application Layer

The application layer orchestrates domain services to accomplish use cases. It knows nothing about how domains work internally.

### Pipeline

```go
// internal/app/pipeline
type Pipeline struct {
    logger      *slog.Logger
    in          ingest.Organizer
    md          metadata.ProductRepository
    rd          script.Renderer
    sourceDir   string
    destination string
    dryRun      bool
    events      chan PipelineEvent
}

func NewPipeline(opts Options) *Pipeline
func (p *Pipeline) Run(ctx context.Context) (RunOutput, error)    // loops RunOnce until ctx cancelled
func (p *Pipeline) RunOnce(ctx context.Context) (RunOutput, error) // one scan pass
```

**Responsibilities (current):**
- `RunOnce`: scans `sourceDir`, per writable unit directory runs ingest → metadata (`FindByID`) → `app/script.Build` (unless `DryRun`), emits `PipelineEvent` (UnitDone/UnitFailed/ScanDone), returns `RunOutput{Processed, Failed}`
- Assembles `ScriptConfig` by convention: streams sorted by `(FileIndex, Index)`, SRT = first `.srt` filename, env paths derived under `Destination` (`buffer`, `in_progress`, `done`, `notifications`, `target`), script artifact at `<Destination>/scripts/<id>.sh`; `EnsureEnvDirs` creates that tree (called from cmd_process at startup)
- Per-unit failures are counted and emitted, not returned; the error return covers scan failure and context cancellation

**What Pipeline does NOT do:**
- Does not scan files directly (delegates to `Organizer`)
- Does not query metadata directly (delegates to `ProductRepository`)
- Does not generate script content (delegates to `domain/script` via `app/script.Build`)
- Does not write files directly (delegates to `Renderer`)

### Use Cases

```go
// internal/app/ingest
type Ingest struct {
	ffprobeBinary string
}
func (ing *Ingest) Ingest(dir string) (ingest.MediaUnit, error)
// real: scan → filter → probe → parse → MediaUnit

// internal/app/script
func Build(ctx context.Context, config script.ScriptConfig, renderer script.Renderer) error
// real: BuildScript(config) → renderer.Render(sc) — materializes a usable artifact
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
| `Options` | Pipeline construction: Logger, Events channel, ports (Ingest, Metadata, Renderer), SourceDir, Destination, DryRun, HardsubTag |
| `RunOutput` | Result summary: Processed (completed chain), Failed (ingest/metadata/script failures) |
| `PipelineEvent` | Signal emitted to CLI: Type (UnitDone/UnitFailed/ScanDone), ID, Error |

## Infrastructure Layer

Infrastructure provides concrete implementations of domain ports and the outside-world integrations (filesystem, ffprobe, config, logging). It depends on domains but domains know nothing about infrastructure.

### Infrastructure Packages

| Package | Contents | Depends On |
|---------|----------|------------|
| `internal/infra/fso` | `Prober` (runs ffprobe, returns JSON) | — |
| `internal/infra/metadata` | `Repository` (implements `ProductRepository`) | `domain/metadata` |
| `internal/infra/script` | `Renderer` (implements `domain/script.Renderer`; MkdirAll + WriteFile + Chmod, options) | `domain/script` |
| `internal/infra/config` | `Config` struct, `Load()` via appcontext configmanager | — |
| `internal/infra/logging` | `Logger` (slog facade) | — |
| `internal/infra/persistence/jsonstore` | JSON key-value store | — |
| `internal/infra/commands` | `CMDProcess` (CLI command; wires app + domain) | `app`, `domain/ingest` |

### Key Implementation: metadata.Repository

```go
type Repository struct {
    mu   sync.RWMutex
    root *Root
}
```

**Workflow:**
1. `New(path)` — reads and unmarshals `metadata.json` into `Root`
2. `FindByID(id)` — delegates to `Root.FindByID(id, WithDescendants)`, converts to `*Product`
3. Returns `*Product` (flat domain DTO)

### Current Ingest Path

The scan/probe/parse implementation is complete:

1. `commands.CMDProcess` validates `--source-root` and `--ffprobe-path` (fallback: flag → `paths.ffprobe_binary` → `ffprobe`), loads metadata (`--metadata-path` or config)
2. Creates `app/ingest.Ingest` with ffprobe binary path and `metadata.Repository` with catalog path
3. `Pipeline.Run` scans directories, calls `Organizer.Ingest(dir)` per directory
4. Each directory is probed, parsed, and validated into a `MediaUnit`
5. Events (`EventUnitDone`, `EventUnitFailed`) are emitted per unit

By default `Pipeline.Run` polls every second until SIGINT/SIGTERM: the context is
cancelled, the run stops after the current unit and the command logs
`pipeline stopped` with the processed/failed totals and exits 0. `--once` runs a
single scan pass (`RunOnce`) and exits the same way.

`infra/fso.Prober` handles ffprobe execution and JSON parsing.

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
        Flags: []cli.Flag{ /* --source-root, --target-root, --ffprobe-path, --metadata-path, --once */ },
        Action: cmdProcessAction,
    }
}

func cmdProcessAction(ctx context.Context, cmd *cli.Command) error {
    // 1. Validate flags (--source-root required, --metadata-path or config)
    // 2. Create dependencies (app/ingest.Ingest stub, Pipeline)
    // 3. Run pipeline (--once: single pass; otherwise poll until SIGINT)
    // 4. Output results (processed/failed summary, exit 0 on cancellation)
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

- Loaded via `github.com/Galdoba/appcontext/configmanager` with `WithFormat(configmanager.TOML)`
- Defaults are hardcoded (`defaultCFG()`); app name constant is `lazyam3`
- `configs/config.yaml` and `configs/config.local.yaml` are empty placeholders

## Workflow

```
1. cmd/lazyam3 starts
2. AppModules created (empty)
3. Root Before: config loaded, logger created (modules.Init); SIGINT/SIGTERM cancel the run context
4. commands.CMDProcess action:
   a. Validate --source-root flag; resolve metadata/ffprobe paths (flag → config → default)
   b. Create app/ingest.Ingest, metadata.Repository, ensure work dirs, create Pipeline
   c. --once → RunOnce (single pass); default → Run (polls until the context is cancelled)
   d. Log "pipeline finished/stopped" with processed/failed totals (exit 0 on signal)
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
