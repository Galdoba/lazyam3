---
id: "0004-pipeline-error-handling-strategy"
title: "Pipeline Error Handling Strategy"
description: "Unit failures are counted and emitted as events without halting the pipeline, scan failures are logged but not returned, and context cancellation is handled specially to preserve aggregated totals across interrupted passes."
status: accepted
date: "2026-11-07"
scope: cross-package
---

# Pipeline Error Handling Strategy

## Context

The pipeline orchestrates three domain layers: ingest, metadata, and script rendering. Each unit processing involves multiple steps that can fail independently: file scanning, ffprobe probing, metadata lookup, and script generation. The pipeline needs a consistent strategy for handling failures across these layers without losing progress or halting the entire run for transient errors.

## Decision

Per-unit failures are counted and emitted as events, not returned as errors that halt the pipeline. The `RunOnce` method returns scan failures and context cancellation as errors, but unit-level failures (ingest, metadata lookup, script build, file preparation) increment a failure counter and emit `EventUnitFailed` or `EventUnitDone` events. Context cancellation (`context.Canceled` or `context.DeadlineExceeded`) returns the aggregated totals alongside the error, preserving units processed in the interrupted pass.

## Rationale

This approach aligns with the pipeline's role as an inter-process worker: individual units may fail due to missing metadata, invalid media, or transient probe errors, but the pipeline should continue processing remaining units rather than aborting entirely. The pipeline is designed to be resilient to per-unit failures, with failures surfaced through the event system for downstream handling (logging, analytics, TUI). Returning only scan failures and context cancellation ensures that structural errors (scan failure, cancellation) are propagated while unit failures are handled gracefully.

## Consequences

- Unit failures do not stop the pipeline; remaining units are processed normally.
- Failure counts are tracked in `RunOutput.Failed` and surfaced via `EventUnitFailed`.
- Scan failures and context cancellation are the only errors returned from `RunOnce` and `Run`.
- Context cancellation preserves the totals from the interrupted pass, avoiding loss of progress.
- Fallback naming is applied when metadata lookup fails, allowing processing to continue for units without metadata.

## Options

- **Return unit failures as errors that halt the pipeline.** Rejected: would stop processing on the first unit failure, losing progress on large input sets.
- **Ignore unit failures silently.** Rejected: failures must be surfaced for visibility and analytics; the event system provides this without halting the pipeline.
- **Return scan failures as warnings.** Rejected: scan failures are structural errors that indicate the pipeline cannot proceed, so they are returned as errors.
