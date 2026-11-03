# Contributing to lazyam3

Thank you for your interest in contributing!

## Prerequisites

- Go 1.26 or later

## Quick Start

```bash
# Clone the repository
git clone https://github.com/Galdoba/lazyam3.git
cd lazyam3

# Build
go build ./...

# Run tests
go test -race -cover ./...

# Static analysis
go vet ./...

# Lint (strict gate: exit 1 on any issue; config in .golangci.yml)
golangci-lint run ./...
```

## Live Test

`scripts/live_test.sh` — resets the testing ground and runs a full pipeline pass:

- Wipes all files in `~/workspace/lazyam3/` (files only, keeps directories)
- Copies `testdata/synthetic/` → `incoming/`
- Builds lazyam3 and runs `process --once` (no flags — uses config)
- Prints all files in the test ground

Run before/after any pipeline change to verify end-to-end behavior.

## Development Workflow

1. Work one feature at a time; the current feature is defined in **Current Feature** in `AGENTS.md`.
2. Analyse the problem, plan, implement, and test.
3. Every feature: `go build ./...`, `go test -race -cover ./...`, `go vet ./...`, `golangci-lint run ./...`.
4. Every fixed bug MUST produce a test.
5. New features MUST list their tests in `doc/feature_list.json`.
6. Update `status` in `doc/feature_list.json` and related documentation.

## Code Guidelines

- Domains are pure — no imports from infra or app.
- Pipeline works only with interfaces and DTOs.
- Infrastructure implements domain ports.
- Commands wire dependencies and call Pipeline.
- Infrastructure failures fail fast.
- Simple solutions MUST be examined first.
- Make only minimal code edits.
- No comments in code unless really needed.
- Do NOT fix bugs unrelated to the current feature.

## Reporting Issues

Use [GitHub Issues](https://github.com/Galdoba/lazyam3/issues). Include:

- Go version (`go version`)
- OS and architecture
- Steps to reproduce
- Expected vs actual behavior
