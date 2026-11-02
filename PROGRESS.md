# PROGRESS

## INGEST-002: bypass non-media files during ingest
- **Status**: Implemented and verified.
- **Change**: `internal/app/ingest/ingest.go:50-58` now logs WARN and continues on probe error for non-media, non-subtitle files instead of returning an error and aborting the directory ingest. Removed unused `fmt` import.
- **Test**: `TestIngest_NonMediaFilesSkipped` added to `internal/app/ingest/ingest_test.go`.
- **Verification**: unit tests pass, lint clean, live test passes (processed=9 failed=0).
