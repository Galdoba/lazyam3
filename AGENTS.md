# lazyam3 — Agent Guide

Go 1.26 CLI application: a media processing pipeline tool (ingest → metadata resolution → script generation → rendering).

**Current State**: Tasks 1-10 complete. Pipeline fully wired end-to-end with HSUB auto-detection, movie naming fix, TOML config, and config-driven paths.

## Current Feature

> **Work on**: refactoring
>
> Status: `planning` 

> **Just finished: `S3` — Configurable hardsub parameters** (`doc/feature_list.json`) — status: `done` · effort: 2-3 hours · priority: 4

Selection rule: pick the highest-priority feature in `doc/feature_list.json` whose `blocked_by` list is empty and whose `status` is `proposed`. When a feature is done, set its `status` to `in_rewiew` in `doc/feature_list.json` before moving on.

## Live Test

`scripts/live_test.sh` — resets the testing ground and runs a full pipeline pass:
- Wipes all files in `~/workspace/lazyam3/` (files only, keeps directories)
- Copies `testdata/synthetic/` → `incoming/`
- Builds lazyam3 and runs `process --once` (no flags — uses config)
- Prints all files in the test ground

Run before/after any pipeline change to verify end-to-end behavior.

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
| `doc/feature_index.md` | Synced feature table from feature_list.json |
| `doc/logging_suggestions.md` | Logging improvement catalog (A1–A4 quick wins, B1–B5 larger items) with trade-offs |
| `doc/HANDOFF.md` | Session handoff: decisions, known issues, what to watch for |
| `doc/PROGRESS.md` | Session progress log: work completed, findings, metrics |
| `doc/BACKLOG.md` | Known issues/footguns and deferred items |
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
- Simples solutions MUST examined first
- Analyze the problem before solving it
- Make only minimal code edits
- No comments in code unless really needed
- Do NOT fix the bugs which unrelated to current feature
- Be honest
- Be carefull on your judjements

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.
