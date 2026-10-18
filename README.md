# LazyAM3

A media processing pipeline tool (ingest → metadata resolution → script generation → rendering).

## Architecture

```
cmd + infra/commands → app → domain
infra                → domain (implements ports)
platform             → nobody (stateless services)
```

See `doc/architecture.md` for the full architecture and `doc/lazyam3.md` for the repository tree and domain overview.

## Directory Layout

| Path | Purpose |
|---|---|
| `cmd/lazyam3/` | Entry point — wires AppModules → config → logger → commands |
| `internal/app/` | Application orchestration (pipeline, ingest/script use cases, AppModules) |
| `internal/domain/` | Pure domain models and rules (ingest, metadata, script) |
| `internal/infra/` | Port implementations (config, logging, fs, ffprobe, metadata, cache, persistence, commands) |
| `internal/platform/` | Stateless services (retry, clock) |
| `internal/presentation/` | TUI (bubbletea; not wired yet) |
| `scrips/` | Standalone helper tools (note: directory name is misspelled) |
| `configs/` | Config placeholders (real format will be TOML — task 10) |
| `testdata/` | Test fixtures (metadata snapshot, ffprobe dumps) |
| `doc/` | Architecture docs and implementation tasks |

## Building

```bash
go build ./...
go test ./...
go vet ./...
```

## Usage

```bash
go run ./cmd/lazyam3 process --source ./incoming --output ./output
```

Note: the pipeline currently runs ingest only and `app/ingest` is a stub — see `doc/tasks.json` for implementation status.
