# Tasks

## 1. Reachability evidence before deletion

- [x] 1.1 Create the branch for this change and verify `git branch --show-current` reports it, so the implementation lands outside `main` as the project workflow requires
- [x] 1.2 Collect gopls symbol-reference evidence for `IsMovie`, `IsSeries`, `IsSeason`, `IsEpisode`, `IsMovieEpisode`, and `IsSeriesEpisode`, and verify each reference list contains only the declaration in `internal/domain/metadata/types.go` plus lines of `internal/domain/metadata/types_test.go`, confirming nothing outside the package depends on them
- [x] 1.3 Collect gopls symbol-reference evidence for `Product`, `Kind`, the five `Kind` constants, `ProductRepository`, and `FindByID`, and verify they are all still referenced from `internal/app/pipeline` and `internal/infra/metadata`, confirming that deleting the predicate layer cannot disturb them

## 2. Remove the predicate layer

- [x] 2.1 Delete the six `Kind` predicate methods from `internal/domain/metadata/types.go` (`types.go:31-60`), and verify `go build ./...` and `go vet ./...` report nothing for the package
- [x] 2.2 Delete `internal/domain/metadata/types_test.go` together with the code it tested, and verify `go test -race -cover ./internal/domain/metadata/` passes and reports a package with no statements left to cover
- [x] 2.3 Verify `internal/infra/metadata`, `internal/app/pipeline`, `internal/domain/callname`, and `internal/domain/script` are untouched by the change (`git status --short` lists only `internal/domain/metadata`), and verify `go test -race ./internal/infra/metadata/ ./internal/app/pipeline/ ./internal/domain/...` passes with their assertions on `Kind` and `Product` unchanged
- [x] 2.4 Confirm `types.go` keeps `Product` with all thirteen fields, `Kind`, the five `Kind` constants, and the existing doc comments intact, and that `port.go` is unchanged, and verify `go doc ./internal/domain/metadata` still lists `Product`, `Kind`, the constants, and `ProductRepository`

## 3. Record the decision

- [x] 3.1 Write `doc/adr/0021-metadata-carries-product-facts-it-does-not-classify.md` stating that the metadata domain performs no kind classification, that the six `Kind` predicates were removed rather than left dormant, that the `Product` fields nothing outside `internal/infra/metadata` assigns and no consumer reads (`Kind`, `ParentID`, `SeasonEp`, `Description`, `Genres`, `Countries`, `KinopoiskID`, `IMDBID`) are model vocabulary rather than dead code, and that kind-based branching returns only as its own change with a named consumer, and verify the record follows the shape of `doc/adr/0020-ingest-carries-stream-facts-it-does-not-validate.md`
- [x] 3.2 Add the matching row to the `doc/adr/INDEX.md` catalog and verify the row's scope and date columns match the new file's front matter

## 4. Integration gate

- [x] 4.1 Run `./scripts/full_test.sh` and verify every reported step is PASS, including both live pipeline runs, so product resolution end to end is unchanged with the removed helpers
- [x] 4.2 Run `golangci-lint run ./...` and `go_vulncheck ./...` and verify both report no issues in `internal/domain/metadata`

## Workflow follow-up

- [x] Verify the change with the `openspec-verify-change` skill before archiving; note that this change carries no delta specs, so nothing merges into `openspec/specs/` at archive
- [x] Implement on the change's own branch, commit the work, and do not push
- [x] Archive with the `openspec-archive-change` skill once the gate passes
