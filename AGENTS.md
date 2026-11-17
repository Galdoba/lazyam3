# lazyam3 — Agent Guide

Go 1.26 CLI application: a media processing pipeline tool (ingest → metadata resolution → script generation → rendering).

## Feature Workflow

Work is tracked as OpenSpec changes. Run `openspec list --json` for active changes and `openspec status --change "<name>" --json` for the next artifact. Load the matching `openspec-*` skill for the phase: propose, continue, apply-change, verify-change, archive-change.

- `openspec/changes/<name>/` holds the change's proposal, design, tasks, and delta specs
- `openspec/specs/` holds the standing capability specs; the delta specs merge there at archive
- `doc/adr/` holds durable decisions (`doc/adr/INDEX.md` is the catalog)

## Live Test

`scripts/live_test.sh` — resets the testing ground and runs a full pipeline pass:
- Wipes all files in `~/workspace/lazyam3/` (files only, keeps directories)
- Copies `testdata/synthetic/` → `incoming/`
- Builds lazyam3 and runs `process --once` (no flags — uses config)
- Prints all files in the test ground

Run before/after any pipeline change to verify end-to-end behavior.

`scripts/live_test_2.sh` — same test ground, but plants an error log beside three of the nine scripts before one `process --once`, then asserts what each store and directory received. Run it for anything that touches error reporting or storage.

## Documentation

Full catalog: `doc/adr/INDEX.md` for decisions, plus the topic docs in `doc/`.

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
- Work one feature at a time; the current change is defined in **Feature Workflow** above
- An OpenSpec change is the unit of work: propose it before implementing, verify it before archiving
- Implementation of the current change MUST be done in it's own branch
- Agent SHOULD commit, but MUST NOT push

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



