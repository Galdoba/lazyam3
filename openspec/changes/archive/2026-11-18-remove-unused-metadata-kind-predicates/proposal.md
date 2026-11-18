# Proposal

## Why

`internal/domain/metadata` carries a `Kind` classification helper layer — six predicate methods — that nothing outside the package calls, together with a test file that exists only to assert against that layer. The package therefore reports 100% coverage while the pipeline exercises none of it, which hides the fact that kind-based classification is not how the pipeline tells a movie from an episode.

## What Changes

- Remove the six `Kind` predicates from `internal/domain/metadata/types.go:31-60`: `IsMovie`, `IsSeries`, `IsSeason`, `IsEpisode`, `IsMovieEpisode`, `IsSeriesEpisode`. gopls symbol references list, for each one, exactly the declaration plus two lines of `internal/domain/metadata/types_test.go` — no production caller exists in `internal/app/pipeline`, `internal/domain/callname`, `internal/domain/script`, or `internal/infra/metadata`.
- Delete `internal/domain/metadata/types_test.go` (43 lines) together with the code it tests; it asserts nothing else in the package.
- Keep `Product`, `Kind`, the five `Kind` constants, and `ProductRepository` exactly as they are. The classification the pipeline actually performs is structural: `callname` decides movie-versus-episode from `SeasonNumber`/`EpisodeNumber` being nil (`internal/domain/callname/names_input.go:129-146`), never from a kind predicate.
- Keep the `Product` fields that `internal/infra/metadata/types.go:284-367` assigns but no consumer ever reads — `Kind`, `ParentID`, `SeasonEp`, `Description`, `Genres`, `Countries`, `KinopoiskID`, `IMDBID` — as model vocabulary rather than dead code, and record that as a deliberate decision in an ADR rather than leaving it implicit.
- Record the decision in `doc/adr/` and add the matching row to `doc/adr/INDEX.md`.

Not changing: `ProductRepository` and its `FindByID`, the infra mapping from `Found` to `Product`, how `callname` derives names, how the pipeline resolves a product for a unit, or any file outside `internal/domain/metadata`.

## Capabilities

### New Capabilities

- none

### Modified Capabilities

- none. No standing requirement changes: none of `openspec/specs/execution-signals`, `openspec/specs/product-identity`, or `openspec/specs/script-generation` mentions `Kind`, product classification, or any of these predicates. This change removes code that no spec ever claimed, so it opts out of spec deltas via `skip_specs: true` in the change's `.openspec.yaml`. The durable outcome is recorded as an ADR (`doc/adr/`), which is where the project keeps decisions rather than requirements.

## Impact

- `internal/domain/metadata`: `types.go` loses its 30-line predicate block (60 lines to 30), `types_test.go` disappears entirely. The package drops from 108 lines to 35 across `port.go` and `types.go`, and both remaining files are pure declarations.
- Coverage: the six predicate bodies are the package's only executable statements, so the 100% figure goes with them. `go test -cover ./internal/domain/metadata/` then reports a package with no statements to cover, which is the honest result for a package that holds a data model and one port.
- No call sites change. Nothing outside the package referenced the removed symbols, so `internal/app/pipeline`, `internal/domain/callname`, `internal/domain/script`, and `internal/infra/metadata` are not edited.
- `doc/adr/` gains one record: the metadata domain classifies nothing, and the unread `Product` facts are vocabulary rather than dead code. `doc/adr/INDEX.md` gains the matching row.
- Risk: if kind-based branching is wanted later it returns as its own change with a named consumer, not as resurrected helpers.
- Gate: `./scripts/full_test.sh` must pass, since both live pipeline runs resolve a product for every unit.
