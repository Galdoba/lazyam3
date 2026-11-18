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

`scripts/full_test.sh` — one gate that runs build, vet, `go test -race -cover ./...`, `golangci-lint run ./...`, then both live tests in order. It reports PASS or FAIL per step and keeps each step's log in a temp directory. Run it before archiving a change.

## Documentation

Full catalog: `doc/adr/INDEX.md` for decisions, plus the topic docs in `doc/`.

## Go Tooling (gopls MCP)

The `gopls_mcp` server is the preferred way to inspect Go code. Agent MUST use it instead of `grep`/`glob`/`view` scanning whenever the question is about Go symbols, and fall back to text search only for prose, config, or non-Go files. These tools are the main agent's own responsibility — delegating Go work to a subagent is not an acceptable substitute.

| Task | Use |
|---|---|
| Workspace/package overview | `go_workspace`, `go_package_api` |
| Find a symbol or API surface | `go_search`, `go_file_context` |
| Where is it used / blast radius before refactoring | `go_symbol_references` |
| Edit errors, vet, lint feedback | `go_diagnostics` |
| Rename across the module | `go_rename_symbol` (never manual multi-file rename) |
| Dependency/security audit | `go_vulncheck` |

Rules:
- Read a file only after `go_file_context` or `go_search` has identified it as relevant
- Call `go_diagnostics` after every edit to a Go file and fix issues in files you changed
- Use `go_rename_symbol` for any rename; verify the reported edits before running tests
- `go_vulncheck ./...` before archiving a change
- Never hand a Go search, reference lookup, or rename to a subagent
- MCP Go tools do not replace `go build`, `go test`, or `golangci-lint` — they complement, not gate, the shell gate in **Commands** below

## Commands

```bash
go build ./...                       # Compile
go test -race -cover ./...           # Run tests
go vet ./...                         # Static analysis
golangci-lint run ./...              # Lint (strict gate: exit 1 on any issue; config in .golangci.yml)
go run ./cmd/lazyam3                 # Run CLI
go run ./cmd/lazyam3 process --help  # Process command help
./scripts/full_test.sh               # Full gate: unit suite plus both live runs
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
- Subagents are forbidden: no `agent_dispatch`, no `agent_send`, no delegated searches or reviews
- The main agent MUST do every Go inspection, edit, and rename itself through `gopls_mcp`

### Code
- Domains are pure — no imports from infra or app
- Pipeline works only with interfaces and DTOs
- Infrastructure implements domain ports
- Commands wire dependencies and call Pipeline
- Infrastructure failures fail fast
- Simple solutions MUST examined first
- Analyze the problem before solving it
- Symbol-level Go work MUST go through `gopls_mcp` (see **Go Tooling (gopls MCP)**); text search is the fallback, not the default
- Make only minimal code edits
- No comments in code unless really needed
- Do NOT fix the bugs which unrelated to current feature
- Be honest
- Be carefull on your judjements
- If unsure: ASK!!!



