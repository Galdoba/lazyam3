---
problem_id: INGEST-002
title: bypass non-media files during ingest
generated_at: 2026-11-02T00:00:00Z
verdict: APPROVED
---

## approaches:

### Approach 1: Change error handling in ingest.go (minimal)

**Description:** In `internal/app/ingest/ingest.go`, replace the probe-error return on line 55 (`return ingest.MediaUnit{}, fmt.Errorf("failed to probe file %q: %w", path, err)`) with a logged warning and `continue`. Subtitle sidecar handling (lines 51-53) is preserved as-is. The ingest loop continues processing remaining files. Files: `internal/app/ingest/ingest.go`.

**Pros:**
- Minimal change: 3-4 lines modified in a single file.
- Preserves subtitle sidecar handling behavior exactly.
- Domain purity maintained: ingest logic uses only `fso` and `logging` ports.
- No changes to ffprobe binary or Prober implementation.
- Easy to unit-test: directory with mixed media/subtitle/non-media files.

**Cons:**
- Changes `Ingest()` semantics from "fail fast on probe error" to "skip non-media files". Callers (pipeline) do not rely on probe errors for non-media files, so this is safe.

**Effort:** 15-30 minutes.

### Approach 2: Classify probe errors via ffprobe stderr inspection

**Description:** Parse the error returned by `fso.Prober.ProbeFile()` to determine whether it represents a non-media file, logging a warning and continuing only for that case. Files: `internal/app/ingest/ingest.go`, possibly `internal/infra/fso/ffprobe.go`.

**Pros:**
- Keeps ffprobe as the authority on media detection.

**Cons:**
- Requires modifying `ffprobe.go` or parsing ffprobe error output, which is fragile and brittle across ffprobe versions.
- Violates non_goal "modifying the ffprobe binary or Prober implementation".
- Error messages from ffprobe are not guaranteed stable; parsing them is risky.

**Effort:** 1-2 hours, high risk.

## verdict: APPROVED

**Rationale:** Approach 1 is the clean, minimal fix. It directly addresses the problem statement: bypass ALL non-media files, log a WARN-level warning when skipping, and continue processing. It preserves subtitle sidecar handling, maintains domain purity, and requires no changes to the ffprobe binary or Prober. Approach 2 introduces brittle error-parsing and violates non_goals. The verdict is APPROVED with Approach 1 as the recommended implementation.

## open_questions:
- Non-goals confirm log level: the problem statement requires WARN-level logging for skipped non-media files, so the warning in Approach 1 uses `logging.Logger.Warn`. No further questions block implementation.
