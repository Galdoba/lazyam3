# Codebase Review — lazyam3

Date: 2026-11-08
Scope: full Go codebase review (`cmd/`, `internal/`, `scripts/`, tests, lint, tests)

## Project overview

Go 1.26 CLI media processing pipeline (`lazyam3`): scan input dir → ffprobe ingest →
metadata resolve → ffmpeg script generation → render.

- ~62 Go files, ~13.3k LOC including tests.
- Layering: `cmd → app → domain → infra`, plus `platform` (stateless services) and
  `presentation/tui` (bubbletea, not wired).
- Architecture docs: `doc/architecture.md`; ADRs: `doc/adr/0001–0009`.

## Verification status (2026-11-08)

| Check | Result |
|---|---|
| `go build ./...` | ✅ pass |
| `go vet ./...` | ✅ pass |
| `go test -race -cover ./...` | ❌ 2 packages failing (same root cause) |
| `golangci-lint run ./...` | ❌ 3 issues (strict gate = fail) |

## Findings

### Blockers (current failures)

1. Nil-logger panic:
   - `internal/infra/script/script.go:58` — `Renderer.Render` calls `r.logger.Info(...)`
     unconditionally; `Renderer.New()` leaves `logger` nil unless `WithLogger` is used.
   - `internal/app/script.Build` forwards a possibly-nil logger into the renderer.
   - Failing tests: `internal/infra/script.TestRenderer_Render`,
     `internal/app/script.TestBuildAndExecute_RealInputs`.

2. Lint gate failures:
   - `internal/app/pipeline/config.go:144` and `internal/domain/callname/shared.go:9` —
     `time.Sleep(time.Second)` forbidden (forbidigo); both carry `// TODO: need to make it
     more go linter friendly and idiomatic`.
   - `internal/app/pipeline/pipeline.go:237` — gofumpt formatting violation.

### Design findings

- App→infra coupling violates the documented rule ("application depends on domain,
  never infra"):
  - `internal/app/pipeline/pipeline.go:105` type-asserts `p.md.(*metadatainfra.Repository)`.
  - `internal/app/pipeline/pipeline.go:159-160` type-asserts `p.logger.(*logging.Logger)`.
  - `internal/app/ingest/ingest.go` imports `internal/infra/fso`.
  - `internal/app/script.Build` takes `*logging.Logger` (infra concrete) instead of the
    `pipeline.Logger` port.
- Stale TODOs/comments:
  - `internal/app/pipeline/pipeline.go:237` — script-name logic TODO.
  - `internal/domain/ingest/parse.go` — `TODO: rework`.
  - `internal/infra/commands/cmd_process.go` — stale "TODO: start event consumer goroutine".
- Semantics questions:
  - `StatusReport.Status` hard-coded `"ok"` (`internal/app/pipeline/pipeline.go:438`).
  - `fallbackProduct` hard-codes `KindMovie` (`internal/app/pipeline/pipeline.go:~310`).
  - `prepareFiles` not transactional.

### Positives

- Domain stays pure (stdlib-only); validation centralized (`ValidateMediaUnit`).
- Script builder self-verifies via unresolved-placeholder check.
- fso error-handling principle and logging facade (custom text handler, color writer,
  multi-handler) are solid.
- Test coverage healthy where tests run (domain 95–100%, infra/logging 93%, pipeline 83%).

## Recommended next steps

1. Fix nil-logger handling in `Renderer.Render` (nil-check or default in `New`).
2. Replace both `time.Sleep` PRT-timestamp stubs with context/timer-based generation.
3. Run gofumpt / fix `pipeline.go:237`.
4. Remove stale TODOs; reduce app→infra concrete coupling.
5. Decide fate of unused `clock`/`retry`/`cache` packages and wire or retire them.

---

Saved from the codebase review session on 2026-11-08.
