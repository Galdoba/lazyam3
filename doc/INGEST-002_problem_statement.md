---
problem_id: INGEST-002
title: bypass non-media files during ingest
repository: github.com/Galdoba/lazyam3
baseline_commit: cf2d9c5f51346bfe213423c8da10573468275eee
problem_type: improvement
status: CONFIRMED
---

## current_behavior: |
  The ingest process scans a directory and probes each regular file with ffprobe via fso.Prober.ProbeFile(). Currently, only files named "metadata.json" are explicitly skipped (ingest.go:45-46). Subtitle files (.srt, .ass, .ssa) that ffprobe cannot probe are recorded as sidecars (ingest.go:51-53). Any other non-media file that ffprobe cannot probe causes ProbeFile() to return an error, which aborts the entire directory ingest (ingest.go:55). The dirty fix commit cf2d9c5 added a metadata.json skip but does not address the general case of arbitrary non-media files in the directory.

## desired_behavior: |
  During ingestion, the program should bypass ALL non-media files. Files that ffprobe cannot probe (and are not subtitle sidecars) should be skipped with a logged warning, and the ingest should continue processing the remaining files in the directory. Only genuine media files (and subtitle sidecars) should be recorded in the MediaUnit.

## rationale: |
  Directories passed to ingest often contain non-media files (metadata.json, logs, READMEs, temp files, etc.). Currently, any such file aborts the entire ingest, making the pipeline fail on directories that contain non-media artifacts. Bypassing non-media files makes ingestion robust and idempotent regardless of directory contents.

## evidence:
  - type: code_review
    reference: internal/app/ingest/ingest.go:45-55
  - type: code_review
    reference: internal/infra/fso/ffprobe.go:41-61
  - type: log_analysis
    reference: doc/dirty_fix_problems.md

## acceptance_criteria:
  - criterion: ingest skips non-media files and continues processing the rest of the directory
    verification_method: unit test for Ingest with a directory containing a mix of media, subtitle, and non-media files
  - criterion: ingest logs a warning when a non-media file is skipped
    verification_method: verify log output contains warning for skipped non-media file
  - criterion: live test is passing
    verification_method: run `live_test.sh`

## non_goals:
  - detecting or extracting metadata from non-media files
  - changing subtitle sidecar handling behavior
  - modifying the ffprobe binary or Prober implementation

## constraints:
  - domain purity: ingest logic in internal/app/ingest may use infra fso and logging ports
  - existing subtitle sidecar handling must be preserved
  - non-media files must be skipped, not cause ingest failure

## risks:
  - silently skipping files that should be media could hide corrupted media files; a logged warning should be emitted for visibility
  - changing error handling in Ingest() could affect callers that expect failure on probe errors

## open_questions:
  - should non-media files that are neither subtitles nor media be logged at warn level or debug level? (Yes, log message with WARN level is REQUIRED)
