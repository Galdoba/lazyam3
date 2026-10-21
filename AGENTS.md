# lazyam3 — Agent Guide

Go 1.26 CLI application: a media processing pipeline tool (ingest → metadata resolution → script generation → rendering).

**Current State**: Tasks 1-10 complete. Pipeline fully wired end-to-end with HSUB auto-detection, movie naming fix, TOML config, and config-driven paths.

## Documentation Guide

| File | Purpose |
|---|---|
| `doc/architecture.md` | Full architecture: layers, domains, ports, dependency rules, development guidelines |
| `doc/lazyam3.md` | Brief overview: workflow, domain modules, repository tree |
| `doc/ingest.md` | Ingest domain: types, pure functions, validation rules, test data |
| `doc/metadata.md` | Metadata domain: Product/Kind, ProductRepository, tree lookup, catalog test data |
| `doc/script.md` | Script domain: BuildScript, naming derivation, audio filters, SRT/HSUB handling, template |
| `doc/tasks.json` | Implementation tasks: items with reasons, verification, blocking deps, status tracking |
| `doc/BACKLOG.md` | Deferred items: language translation, source file naming hook |
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

## External Dependencies

| Package | Purpose |
|---|---|
| `github.com/urfave/cli/v3` | CLI framework (direct dependency) |
| `charm.land/bubbletea/v2` | TUI framework (`internal/presentation/tui`, not wired yet) |
| `github.com/Galdoba/appcontext` | Config loading via configmanager (TOML; `[hardsub] tag` for HSUB detection) |
| `github.com/tidwall/gjson` | JSON querying (`scrips/asses` tool) |



## Development Rules

- Domains are pure — no imports from infra or app
- Pipeline works only with interfaces and DTOs
- Infrastructure implements domain ports
- Commands wire dependencies and call Pipeline
- Make only minimal code edits
- Infrastructure failures fail fast

