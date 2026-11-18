# Tasks

## 1. Reachability evidence before deletion

- [x] 1.1 Create the branch for this change and verify `git branch --show-current` reports it, so the implementation lands outside `main` as the project workflow requires
- [x] 1.2 Collect gopls symbol-reference evidence for `ValidateMediaUnit`, `validateVideo`, `validateAudio`, `validateDurations`, `rationalMeanFloat`, `ValidationIssues`, `ValidationIssue`, `MediaUnit.Validation`, `TagMap`, and `buildStreams`' second result, and verify that every reference list contains only in-package declarations plus `internal/domain/ingest` test call sites, confirming nothing outside the package depends on them

## 2. Remove the validation subsystem

- [x] 2.1 Delete `ValidateMediaUnit`, `validateVideo`, `validateAudio`, `validateDurations` and `rationalMeanFloat` from `internal/domain/ingest/unit.go`, and verify `go build ./...` and `go vet ./...` report nothing for the package
- [x] 2.2 Delete the `ValidationIssues` and `ValidationIssue` types and the `MediaUnit.Validation` field, and verify `go build ./...` succeeds, which proves no consumer outside the package referenced them
- [x] 2.3 Delete `TestValidateMediaUnit`, `TestValidateMediaUnit_Details` and `TestValidateMediaUnit_Comprehensive` from `internal/domain/ingest/unit_test.go`, and verify `go test -race -cover ./internal/domain/ingest/` passes with the coverage percentage of the remaining code unchanged within a point

## 3. Remove the remaining unreachable declarations

- [x] 3.1 Delete the `TagMap` type from `internal/domain/ingest/parse.go`, and verify `go build ./...` and `go vet ./...` report nothing
- [x] 3.2 Delete the orphaned `// resolveID extracts product IDs …` comment in `internal/domain/ingest/unit.go`, and verify the file's remaining doc comments each sit directly above the declaration they describe
- [x] 3.3 Reduce `buildStreams` to a single return value and update its only call site in `NewMediaUnit`, and verify `go test -race ./internal/domain/ingest/` passes, including `TestNewMediaUnit` and `TestNewMediaUnit_Empty`

## 4. Make the ffprobe fixtures exercise what they claim

- [x] 4.1 Change `"tag"` to `"tags"` in the `TestParseStreams` fixture (`internal/domain/ingest/unit_test.go:85`) and the `videoProbe` fixture in `TestNewMediaUnit` (`unit_test.go:265`), and verify `TestParseStreams` passes, then temporarily make `detectInterlaced` return `true` unconditionally, confirm `TestParseStreams` now fails on its interlaced assertion, and restore `detectInterlaced` unchanged
- [x] 4.2 Confirm the helper edge-case suites `TestParseDuration`, `TestParseRational`, `TestParseBitRate`, `TestStreamKind` and `TestDetectInterlaced` are still present and covering the `N/A`, `0/0`, empty and malformed branches, and verify `go test -race ./internal/domain/ingest/` passes with them running

## 5. Record the decision

- [x] 5.1 Write `doc/adr/0020-ingest-carries-stream-facts-it-does-not-validate.md` stating that ingest does not validate media quality, that the descriptive `Stream` fields (`Codec`, `Width`, `Height`, `FrameRate`, `PixelFormat`, `SampleRate`, `SampleFormat`, `Forced`, `Duration`, `BitRate`) are model vocabulary rather than dead code, and that validation returns only as its own change wired through the pipeline's error handling, and verify the record follows the shape of the neighbouring records such as `doc/adr/0019-input-name-grammar-owned-by-ingest.md`
- [x] 5.2 Add the matching row to the `doc/adr/INDEX.md` catalog and verify the row's scope and date columns match the new file's front matter

## 6. Integration gate

- [x] 6.1 Run `./scripts/full_test.sh` and verify every reported step is PASS, including both live pipeline runs, so ingest's end-to-end behavior is unchanged with the removed code
- [x] 6.2 Run `golangci-lint run ./...` and `go_vulncheck ./...` and verify both report no issues in `internal/domain/ingest`

## Workflow follow-up

- Verify the change with the `openspec-verify-change` skill before archiving; note that this change carries no delta specs, so nothing merges into `openspec/specs/` at archive
- Implement on the change's own branch, commit the work, and do not push
- Archive with the `openspec-archive-change` skill once the gate passes
