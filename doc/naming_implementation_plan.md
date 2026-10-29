---
plan_id: PLAN-2026-1029-cross-platform-path-resolution
problem_id: P-2026-1029-cross-platform-path-resolution
selected_option_id: OPT-3
baseline_commit: a109189
---

# Cross-Platform Path Resolution — Implementation Plan

Selected solution: **OPT-3 — Architectural Change: Config-driven paths with platform abstraction**

## Tasks

- task_id: T1
  title: Extend and validate TOML paths section
  description: Ensure the Paths.Executor map (buffer, in_progress, done, notifications, archive, target) and Paths.Generator are loaded and validated from TOML. Add schema validation so generator and executor resolve to the same canonical form.
  affected_paths:
    - internal/infra/config/config.go
    - internal/infra/config/config_test.go
  depends_on: []
  acceptance_criteria: Paths section loads from TOML; required generator paths validated; defaultCFG() executor map present and non-empty.
  verification_command: go test -race -cover ./internal/infra/config/...
  risk_level: medium
  requires_human_review: true

- task_id: T2
  title: Introduce PathBuilder service
  description: Platform-agnostic path builder that uses filepath.Join for the host OS and exposes canonical forward-slash strings for metadata markers.
  affected_paths:
    - internal/platform/pathbuilder.go
    - internal/platform/pathbuilder_test.go
  depends_on: [T1]
  acceptance_criteria: PathBuilder.native() uses filepath; PathBuilder.canonical() returns forward-slash strings; unit tests cover native vs canonical output per OS.
  verification_command: go test -race -cover ./internal/platform/...
  risk_level: low
  requires_human_review: false

- task_id: T3
  title: Wire PathBuilder into AppModules
  description: Wire PathBuilder through app module wiring and pipeline options so domain layers receive it via ports.
  affected_paths:
    - internal/app/pipeline/config.go
    - internal/app/pipeline/dto.go
    - internal/app/*.go (AppModule wiring)
  depends_on: [T2]
  acceptance_criteria: PathBuilder instance available to domain layers; app builds; wiring tests pass.
  verification_command: go build ./... && go test -race -cover ./internal/app/...
  risk_level: medium
  requires_human_review: true

- task_id: T4
  title: Update domain/script.go for script name construction
  description: Consume PathBuilder for script name construction; emit forward-slash paths in generated naming scripts.
  affected_paths:
    - internal/domain/script/script.go
    - internal/domain/script/template.go
  depends_on: [T2, T3]
  acceptance_criteria: Generated script paths use forward slashes; script tests pass.
  verification_command: go test -race -cover ./internal/domain/script/...
  risk_level: low
  requires_human_review: false

- task_id: T5
  title: Update metadata markers to canonical form
  description: Generator writes and executor reads .processing/.done markers using canonical forward-slash form from config.
  affected_paths:
    - internal/domain/metadata/port.go
    - internal/domain/metadata/types.go
    - infrastructure metadata implementations
  depends_on: [T2, T3]
  acceptance_criteria: Markers constructed via PathBuilder canonical(); marker tests pass.
  verification_command: go test -race -cover ./internal/domain/metadata/...
  risk_level: low
  requires_human_review: false

- task_id: T6
  title: Update domain/ingest for source file paths
  description: Consume PathBuilder for source file path handling in ingest domain.
  affected_paths:
    - internal/domain/ingest/unit.go
    - internal/domain/ingest/port.go
  depends_on: [T2, T3]
  acceptance_criteria: Source paths canonical; ingest tests pass.
  verification_command: go test -race -cover ./internal/domain/ingest/...
  risk_level: low
  requires_human_review: false

- task_id: T7
  title: Add path existence validation before file operations
  description: Validate path existence before move/detect operations; fail fast per requirements.
  affected_paths:
    - internal/infra/* (file mover logic, script renderer)
  depends_on: [T2, T3, T5]
  acceptance_criteria: Missing source dir fails fast with clear error; validation tests pass.
  verification_command: go test -race -cover ./...
  risk_level: medium
  requires_human_review: true

## Test Plan

- unit_tests:
    - PathBuilder unit tests: native vs canonical output per OS (Linux/Windows simulation)
    - Config schema test: paths section loads and normalizes correctly
- integration_tests:
    - Generate script on Linux, run marker logic, assert forward-slash paths and successful .processing → .done transition
- regression_tests:
    - scripts/live_test.sh with config-driven paths
    - Existing .processing → .done transition tests across domain modules
- manual_checks:
    - Explicit path-validation failures (missing source dir fails fast)
    - Cross-platform verification: generate on Windows simulator, execute on Linux

## Rollback Plan

| If the feature is reverted to baseline commit `a109189`, the previous TOML schema (Paths.Generator + Paths.Executor map without PathBuilder) is restored. Domain ports and metadata marker logic revert to pre-implementation state. Config rollback restores previous path resolution; no metadata markers are lost since marker format reverts to the original host-native form.

## Definition of Done

- All tasks T1–T7 completed and verified
- `go build ./...` passes
- `go test -race -cover ./...` passes
- `go vet ./...` passes
- `golangci-lint run ./...` passes
- `scripts/live_test.sh` passes end-to-end with config-driven paths
- Path markers use canonical forward-slash form on both generator and executor
- Path validation present and fails fast on missing source directories
- `doc/feature_list.json` updated with naming feature entry
- `doc/INDEX.md` updated with `naming_implementation_plan.md` entry
- `doc/HANDOFF.md` rewritten with findings
