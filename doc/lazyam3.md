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

**Implementation status (tasks 1-9 complete):** ingest, metadata, and script domain boundaries are defined; `app/ingest`, `infra/metadata.Repository`, `domain/script.BuildScript`, `infra/script.Renderer`, and `app/script.Build` are real implementations. `Pipeline.Run`/`RunOnce` wires the full chain: scan → ingest → metadata → script build → render (env paths derived under `--out` root; `RunOutput{Processed, Failed}`). Task 9 (live testing) verified with synthetic sources and real config — scripts render to `/home/galdoba/workspace`, HSUB auto-detection via `[hardsub] tag`, naming fix (no `_se` for movies). Task 10 (TOML config) deferred.

## Domain Modules

### Ingest (physical files)

Language of physical files and streams. Pure data and functions — no IO. Implements ID extraction, ffprobe stream parsing, and domain validation rules. Port: `Organizer`, implemented in `app/ingest` (real scan/probe/parse logic; configurable ffprobe binary). See `doc/ingest.md`.

### Metadata (product catalog)

Language of media products from the counterparty. `id` identifies a catalog object (series, season, series episode, movie, or movie episode); `Kind` says which. Domain types: `Product` (flat DTO for the pipeline), `Kind`. Port: `ProductRepository`, implemented by `infra/metadata.Repository` (JSON storage types, tree search, flattening to `Product`). See `doc/metadata.md`.

### Script (rendering)

Language of rendering script generation. Types: `Script`, `ScriptConfig`, `StreamSelection`. Port: `Renderer` (`Render(Script) error` — materialize a usable artifact). Logic: "Pass config (DTO) → receive file text" — pure builder `BuildScript` implemented (template rendering, naming derivation with `deriveNameTag`/`deriveSeasonDir`, audio filters, SRT/HSUB handling; see `doc/script.md`). `ScriptConfig` composition: ingest fields (`Filenames`, `Streams`, `SRT`, `HSUBFile`) + a single metadata `Product` (with `SeriesTitle` set by repo) + `Environment` paths. `app/script.Build` = `BuildScript` + `Renderer.Render`; `infra/script.Renderer` writes the executable file (MkdirAll, WriteFile, Chmod; default 0o755). Pipeline calls Build in every `RunOnce` pass.

---

## Repository Tree

```
.
├── cmd/
│   └── lazyam3/
│       └── main.go                 # entry point: AppModules → config/logger → commands
│
├── configs/                        # placeholder YAML configs (TOML planned — task 10)
│
├── doc/
│   ├── architecture.md             # architecture overview and development guidelines
│   ├── ingest.md                   # ingest domain details
│   ├── lazyam3.md                  # this file
│   ├── metadata.md                 # metadata domain details
│   ├── script.md                   # script domain details
│   ├── tasks.json                  # implementation tasks and status
│   └── BACKLOG.md
│
├── internal/
│   ├── app/                        # APPLICATION LAYER: orchestration, works on ports and DTOs
│   │   ├── ingest/                 # Ingest use case (scan/probe/parse)
│   │   ├── pipeline/               # Pipeline orchestrator + DTOs
│   │   └── script/                 # script Build use case
│   │
│   ├── domain/                     # DOMAIN LAYER: models and rules. Pure Go. No os/net.
│   │   ├── ingest/                 # physical files and streams (see doc/ingest.md)
│   │   ├── metadata/               # product catalog (see doc/metadata.md)
│   │   └── script/                 # rendering script generation
│   │       └── translit/           # Cyrillic→Latin transliteration (script subdomain)
│   │
│   ├── infra/                      # PORT IMPLEMENTATIONS. Knows about the outside world. Depends on domain.
│   │   ├── cache/                  # in-memory cache
│   │   ├── commands/               # CLI entry points (wires app + domain)
│   │   ├── config/                 # Config via appcontext configmanager (TOML)
│   │   ├── fso/                    # Prober: runs ffprobe
│   │   ├── logging/                # structured logging (slog facade)
│   │   ├── metadata/               # Repository implements ProductRepository
│   │   ├── script/                 # Renderer: usable-artifact materialization
│   │   └── persistence/jsonstore/  # JSON key-value store
│   │
│   ├── platform/                   # STATELESS SERVICES. No domain dependencies.
│   │   ├── clock/                  # time abstraction
│   │   └── retry/                  # backoff, jitter
│   │
│   └── presentation/
│       ├── cli/                    # empty — CLI commands currently live in infra/commands
│       └── tui/                    # bubbletea TUI model (not wired into cmd yet)
│
├── scrips/                         # standalone helper tools (native name is misspelled)
│   ├── asses/                      # JSON querying tool (gjson)
│   └── find/
│
├── testdata/
│   ├── metadata.json               # product catalog snapshot
│   ├── examples/                   # ffprobe output dumps and sample media
│   └── synthetic/                  # synthetic test cases (2s SD video, various stream configs)
│
├── .golangci.yml
├── AGENTS.md
├── go.mod
├── go.sum
└── README.md
```

## Dependency Notes

Full dependency rules, layer descriptions, and development guidelines: `doc/architecture.md`.

Summary: `cmd` → app, infra, domain (wiring only); `infra` → domain (implements ports); `app` → domain; `platform` → no domain deps; domains are pure. Only accepted cross-domain dependency: `domain/script` → `domain/ingest` (via `ScriptConfig`).
