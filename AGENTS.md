# lazyam3 — Agent Guide

Go 1.26 CLI application: a media processing pipeline tool (ingest → metadata resolution → script generation → rendering).

**Current State: TASK 1 COMPLETE — ingest domain defined (types + pure functions, 24 unit tests). `app/ingest` is a stub; pipeline wires ingest only. See `doc/tasks.json`.**

## Documentation Guide

| File | Purpose |
|---|---|
| `doc/architecture.md` | Full architecture: layers, domains, ports, dependency rules, development guidelines |
| `doc/lazyam3.md` | Brief overview: workflow, domain modules, repository tree |
| `doc/tasks.json` | Implementation tasks: items with reasons, verification, blocking deps, status tracking |
| `AGENTS.md` | This file: commands, dependencies, quick reference |

## Commands

```bash
go build ./...          # Compile
go test ./...           # Run tests
go vet ./...            # Static analysis
golangci-lint run ./... # Lint (10 linters, issues-exit-code: 0)
go run ./cmd/lazyam3    # Run CLI
go run ./cmd/lazyam3 process --help  # Process command help
```

## Architecture Overview

```
presentation (cmd/, infra/commands/)  ← CLI wiring, flags, output
    ↓
application (internal/app/)   ← Pipeline orchestrator, use cases, AppModules
    ↓
domain (internal/domain/*)    ← Pure logic, zero external deps
    ↑
infrastructure (internal/infra/*)  ← Concrete implementations of ports
platform (internal/platform/*)     ← Stateless services (no domain deps)
```

**Key principles:**
- Domains are pure — no imports from infra or app (exception: `domain/script` imports `domain/ingest`)
- Pipeline works only with interfaces and DTOs
- Infrastructure implements domain ports
- Commands wire dependencies and call Pipeline

## External Dependencies

| Package | Purpose |
|---|---|
| `github.com/urfave/cli/v3` | CLI framework (direct dependency) |
| `charm.land/bubbletea/v2` | TUI framework (`internal/presentation/tui`, not wired yet) |
| `github.com/Galdoba/appcontext` | Config loading via configmanager (JSON stub; TOML planned — task 10) |
| `github.com/tidwall/gjson` | JSON querying (`scrips/asses` tool) |

Note: `go mod tidy` is pending — `appcontext` and `gjson` are imported directly but still marked `// indirect` in go.mod.

## Key Types

| Type | Package | Purpose |
|---|---|---|
| `Pipeline` | `internal/app/pipeline` | Orchestrator (currently `Ingest` only) |
| `RunInput` / `RunOutput` | `internal/app/pipeline` | DTOs at presentation ↔ app boundary |
| `MediaUnit` | `internal/domain/ingest` | Aggregated file info (ID, filepaths, streams, validation) |
| `Stream` | `internal/domain/ingest` | Single stream (kind, codec, layout, language, …) |
| `IngestPort` | `internal/domain/ingest` | Port for file ingestion |
| `ProductRepository` | `internal/domain/metadata` | Port for metadata lookup |
| `Found` / `Root` | `internal/domain/metadata` | Product tree and search result |
| `Renderer` | `internal/domain/script` | Port for script rendering |
| `AppModules` | `internal/app` | Shared dependency container (Config, Logger; lazy Cache, Store) |
