# HANDOFF

## Summary
Implemented INGEST-002: bypass non-media files during ingest.

## What was done
- Modified `internal/app/ingest/ingest.go:50-58` to log WARN and continue on probe error for non-media, non-subtitle files (removed the `fmt.Errorf` return and `fmt` import).
- Added `TestIngest_NonMediaFilesSkipped` in `internal/app/ingest/ingest_test.go` verifying mixed media/subtitle/non-media directory ingestion succeeds with a video stream.
- Updated `doc/feature_list.json` (INGEST-002 marked done with test list).
- Updated `AGENTS.md` current feature to INGEST-002 complete.

## Verification
- `go test -race -cover ./internal/app/ingest/` — passes (74.2% coverage).
- `golangci-lint run ./internal/app/ingest/` — 0 issues.
- `scripts/live_test.sh` — processed=9 failed=0, including the "With_non_media" directory which contains `something-unrelated.md`.

## Notes
- Lint error in `internal/domain/callname/shared.go:9:2` (forbidigo `time.Sleep`) is pre-existing and unrelated to this feature.
- Non-goals respected: subtitle sidecar handling preserved, ffprobe binary/Prober not modified, domain purity maintained.
