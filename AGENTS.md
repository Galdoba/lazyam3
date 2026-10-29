# lazyam3 — Agent Guide

Go 1.26 CLI application: a media processing pipeline tool (ingest → metadata resolution → script generation → rendering).

**Current State**: Tasks 1-10 complete. Pipeline fully wired end-to-end with HSUB auto-detection, movie naming fix, TOML config, config-driven paths, empty-dir sanitation, and cross-platform path resolution (PathBuilder, canonical metadata markers). Marker metadata path bug fixed. Entering maintenance phase.

## Current Feature

> **STORAGE-002** — persist .done processing history to jsonstore before deletion
>
> **Status**: Discovery complete, solution-search skill created. Ready for solution search.
> Problem Statement: `doc/STORAGE-002_problem_statement.md` (landing gate: PASS).


## Live Test

`scripts/live_test.sh` — resets the testing ground and runs a full pipeline pass:
- Wipes all files in `~/workspace/lazyam3/` (files only, keeps directories)
- Copies `testdata/synthetic/` → `incoming/`
- Builds lazyam3 and runs `process --once` (no flags — uses config)
- Prints all files in the test ground

Run before/after any pipeline change to verify end-to-end behavior.

## Documentation

Full catalog: `doc/INDEX.md`. 

## Commands

```bash
go build ./...                       # Compile
go test -race -cover ./...           # Run tests
go vet ./...                         # Static analysis
golangci-lint run ./...              # Lint (strict gate: exit 1 on any issue; config in .golangci.yml)
go run ./cmd/lazyam3                 # Run CLI
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
| `github.com/Galdoba/appcontext` | Config loading via configmanager (TOML; `[script_generation.hardsub.tag]` for HSUB detection) |
| `github.com/tidwall/gjson` | JSON querying (`scrips/asses` tool) |

## Project Rules

### Workflow
- Work one feature at a time; the current feature is defined in **Current Feature** above
- Every feature: analyse → plan → implement → test → `go build` + `go test -race -cover` + `go vet` + `golangci-lint run` → update `status` in `doc/feature_list.json` and other related documentation
- Every fixed bug MUST produce a test
- New features MUST list their tests in `doc/feature_list.json`
- Update **Current Feature** in this file when starting/finishing work
- HANDOFF.md is always to be REWRITTEN completly with your personal findings
- PROGRESS.md is APPEND only

### Code
- Domains are pure — no imports from infra or app
- Pipeline works only with interfaces and DTOs
- Infrastructure implements domain ports
- Commands wire dependencies and call Pipeline
- Infrastructure failures fail fast
- Simple solutions MUST examined first
- Analyze the problem before solving it
- Make only minimal code edits
- No comments in code unless really needed
- Do NOT fix the bugs which unrelated to current feature
- Be honest
- Be carefull on your judjements
- If unsure: ASK!!!



