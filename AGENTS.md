# lazyam3 — Agent Guide

Go 1.26 CLI application: a media processing pipeline tool (ingest → metadata resolution → script generation → rendering).

**Current State**: Tasks 1-10 complete. Pipeline fully wired end-to-end with HSUB auto-detection, movie naming fix, TOML config, and config-driven paths.

## Current Feature

> **Work on: `C1` — Config package refactoring** (`doc/feature_list.json`)
>
> Status: `proposed` · blocked_by: `["A4" — done, in_rewiew]` · effort: 30 min · priority: 3

Selection rule: pick the highest-priority feature in `doc/feature_list.json` whose `blocked_by` list is empty and whose `status` is `proposed`. When a feature is done, set its `status` to `in_rewiew` in `doc/feature_list.json` before moving on.

## Documentation Index

| File | Purpose |
|---|---|
| `doc/architecture.md` | Full architecture: layers, domains, ports, dependency rules, development guidelines |
| `doc/lazyam3.md` | Brief overview: workflow, domain modules, repository tree |
| `doc/ingest.md` | Ingest domain: types, pure functions, validation rules, test data |
| `doc/metadata.md` | Metadata domain: Product/Kind, ProductRepository, tree lookup, catalog test data |
| `doc/script.md` | Script domain: BuildScript, naming derivation, audio filters, SRT/HSUB handling, template |
| `doc/callname.md` | CallName bounded context: input/output filename construction, transliteration, naming patterns |
| `doc/feature_list.json` | Feature backlog: status, priority, blocked_by, implementation notes, tests per feature |
| `doc/logging_suggestions.md` | Logging improvement catalog (A1–A4 quick wins, B1–B5 larger items) with trade-offs |
| `doc/BACKLOG.md` | Deferred items: language translation, source file naming hook |
| `AGENTS.md` | This file: current feature, commands, rules, quick reference |

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
| `github.com/Galdoba/appcontext` | Config loading via configmanager (TOML; `[hardsub] tag` for HSUB detection) |
| `github.com/tidwall/gjson` | JSON querying (`scrips/asses` tool) |

## Project Rules

### Workflow
- Work one feature at a time; the current feature is defined in **Current Feature** above
- Every feature: analyse → plan → implement → test → `go build` + `go test -race -cover` + `go vet` + `golangci-lint run` → update `status` in `doc/feature_list.json` and other related documentation
- Every fixed bug MUST produce a test
- New features MUST list their tests in `doc/feature_list.json`
- Update **Current Feature** in this file when starting/finishing work

### Code
- Domains are pure — no imports from infra or app
- Pipeline works only with interfaces and DTOs
- Infrastructure implements domain ports
- Commands wire dependencies and call Pipeline
- Infrastructure failures fail fast
- Simples solutions SHOULD examined first
- Analyze the problem before solving it
- Make only minimal code edits
- No comments in code unless really needed
- Do NOT fix the bugs which unrelated to current feature
