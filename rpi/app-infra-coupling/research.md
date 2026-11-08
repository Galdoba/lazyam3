# App-to-Infra Coupling Research

## Problem Context

Restated problem: The codebase review (`doc/code_review.md`) reports that the
application layer couples to infrastructure, violating the documented dependency
rule "application depends on domain, never infra". The question under research is
whether that claim is valid, whether the code can be reorganized to satisfy the
rule, or whether the claim should be rejected.

Business/functional intent: The architecture doc (`doc/architecture.md`) mandates
strict layering so the domain stays pure and infrastructure stays swappable. The
rule exists to keep the domain abstraction intact and avoid concrete-type leakage
between layers.

Current vs desired behavior:
- Current: The application layer uses domain interfaces for its three domain
  capabilities (ingest, metadata, script), but has four cited sites where it
  imports or takes concrete `internal/infra/*` types.
- Desired: No application package imports or function signatures depend on concrete
  infra types for its domain logic.

Constraints:
- Must not break the passing build, vet, or tests.
- The three domain ports (Organizer, ProductRepository, Renderer) must remain
  satisfied by their infra implementations.
- Any change must respect ADR 0001, which already governs the logger
  type-assertion pattern.

## Affected Files

```
doc/architecture.md:267
doc/architecture.md:275
doc/adr/0001-fso-logger-injection.md
internal/app/pipeline/pipeline.go:105
internal/app/pipeline/pipeline.go:159-160
internal/app/ingest/ingest.go:9
internal/app/ingest/ingest.go:17
internal/app/ingest/ingest.go:39
internal/app/script/build.go:13
internal/app/pipeline/dto.go:16
internal/domain/metadata/port.go
internal/domain/ingest/port.go
internal/domain/script/port.go
internal/infra/fso/filescanner.go:22
internal/infra/logging/logger.go:20
```

- `doc/architecture.md:267` and `doc/architecture.md:275` define the rule
  `app -> domain (never infra)` and restate it as "Application depends on domain
  interfaces - never on infrastructure". This is the rule the claim cites; it is
  an import-direction rule.
- `doc/adr/0001-fso-logger-injection.md` is an accepted ADR that explicitly blesses
  the `pipeline.Logger` to `*logging.Logger` type-assertion conversion used in
  `pipeline.go` and `ingest.go`. The logger case is a documented, reviewed
  decision, not an accidental violation.
- `internal/app/pipeline/pipeline.go:105` type-asserts
  `p.md.(*metadatainfra.Repository)`; the field `p.md` is the domain interface
  `metadata.ProductRepository`. The infra import exists only to name the concrete
  type for an optional `Reload()` optimization.
- `internal/app/pipeline/pipeline.go:159-160` type-asserts
  `p.logger.(*logging.Logger)` to thread the logger into `fso.Options`. Same
  escape-hatch shape as ADR 0001.
- `internal/app/ingest/ingest.go:9` and `:10` import `infra/fso` and
  `infra/logging`; `:17` stores `*logging.Logger`; `:39` calls `fso.NewProber`.
  This is genuine app-to-infra implementation coupling, not a type assertion.
- `internal/app/script/build.go:13` takes `*logging.Logger` in its signature.
  Genuine signature-level coupling.
- `internal/app/pipeline/dto.go:16` defines the `pipeline.Logger` interface
  (Debug/Info/Warn/Error). It duplicates the exact four-method surface of
  `infra/logging.Logger`.
- `internal/domain/metadata/port.go`, `internal/domain/ingest/port.go`, and
  `internal/domain/script/port.go` are the three domain ports; the app already
  depends on all three as interfaces.
- `internal/infra/fso/filescanner.go:22` shows `fso` (infra) already imports
  `infra/logging`, so the logger interface already lives in infra.
- `internal/infra/logging/logger.go:20` is the concrete `*logging.Logger` facade.

## Code Examples

Current shape (escaped coupling):

```go
// internal/app/pipeline/pipeline.go:105
if repo, ok := p.md.(*metadatainfra.Repository); ok {
    if changed, err := repo.Reload(); err == nil && changed { ... }
}

// internal/app/script/build.go:13
func Build(logger *logging.Logger, ctx context.Context, config script.ScriptConfig, renderer script.Renderer) error
```

Proposed shape (interface / port driven):

```go
// internal/domain/metadata/port.go: add Reload as a real port method
type ProductRepository interface {
    FindByID(id int64) (*Product, error)
    Reload() (bool, error)
}

// internal/app/script/build.go:13
func Build(logger pipeline.Logger, ctx context.Context, config script.ScriptConfig, renderer script.Renderer) error
```

## External Research Findings

- Clean / hexagonal architecture dependency rule: higher-level layers depend on
  abstractions, not concretions; the direction of dependency points inward toward
  the domain. Infrastructure implements ports defined in the domain or
  application layer. This is the principle behind the project's
  `app -> domain (never infra)` rule.
- Go interface ownership guidance (blog.golang.org): interfaces belong to the
  package that consumes them; a provider package should not define interfaces for
  consumers. The inverse problem here is that the logging interface is defined in
  the app layer, but a second consumer (infra) needs the same surface and cannot
  import app. The resolution is to place the shared logging interface in a layer
  both can depend on (the stateless platform package or the domain).
- Type assertions as optional-behavior probes are idiomatic Go for detecting a
  richer implementation (`if f, ok := w.(Flusher); ok`), but they require importing
  the concrete type, which is exactly the coupling cost in play here.

## FAR Scale Output

### Factual Score: 5
The claim is directly evidenced in code (four cited import/type-assertion sites)
and in docs (the architecture.md rule plus ADR 0001). Every cited location was
read and confirmed.

### Actionable Score: 4
Every case has a concrete remediation. The trivial fix (script.Build signature)
can be done immediately; the logger interface move needs a one-line decision on
where the shared interface lives (platform vs domain).

### Relevant Score: 4
Relevant to architecture hygiene, but the domain abstraction is intact and the
only current failures (nil logger, lint) are unrelated. This is a hygiene
finding, not a blocker.

### Mean: 4.33

## Testing Strategy

- Unit: `internal/app/script` test `Build` with a mock logger implementing the port
  interface; assert the concrete `*logging.Logger` is no longer required.
- Unit: `internal/app/ingest` test with a stub prober once a `Prober` interface is
  introduced.
- Integration: run `scripts/live_test.sh` end-to-end after each change to confirm
  the full pipeline still processes a unit.
- Build gates: `go build ./...`, `go vet ./...`, `go test -race -cover ./...`, and
  `golangci-lint run ./...` must all pass.
- Add compile-time interface checks (`var _ script.Renderer = ...`) wherever new
  interfaces are introduced.

## Potential Design Pattern Recommendations

- Port-and-adapter (hexagonal): expose `Reload()` on the `metadata.ProductRepository`
  domain port so the pipeline calls it through the interface, removing the metadata
  type assertion.
- Interface where consumed, lifted to a shared layer: define the logging surface
  (Debug/Info/Warn/Error) in the stateless `platform` package (or the domain),
  have `infra/logging`, `infra/fso`, and `app/script` all depend on it, and drop
  the `pipeline.Logger` duplicate plus both type assertions. This is the
  highest-value change and also removes the ADR 0001 known limitation.
- Optional-behavior type assertion (keep if justified): if `Reload()` is truly
  infra-internal and not a domain capability, keep the comma-ok assertion but move
  the `metadatainfra` import to a single adapter file and document it, rather than
  promoting it to a domain port.
- Constructor interface injection: `app/script.Build` should accept the port
  interface, not the concrete logger (accept interfaces, return structs).

## Assumptions

- The `pipeline.Logger` interface and `*logging.Logger` share the identical
  four-method surface (Debug/Info/Warn/Error); confirmed by reading both.
- `fso` uses only the Debug/Info/Warn logging surface, so a shared logging
  interface would satisfy it without widening.
- The `metadata.Repository.Reload()` method is infra-internal (it reads file mtime)
  and is only ever called on the concrete type in the pipeline loop.
- The `config.Config` and `jsonstore.Store` fields on `Options` are treated as
  acceptable composition-root wiring, not part of this finding, unless the team
  decides the rule is a strict import ban.
- The three domain ports remain the single source of truth for how the app
  consumes domain capabilities.

## Out of Scope

- The unrelated nil-logger panic in `infra/script.Renderer` and the lint/build
  failures cited as blockers.
- The `config.Config` and `jsonstore.Store` concrete fields on `Options`
  (composition wiring), unless the rule is reinterpreted as a strict import ban.
- Moving `Reload()` out of `infra/metadata` if it is promoted to a domain port.
- Refactoring the `fso` snapshot/traversal logic itself.
