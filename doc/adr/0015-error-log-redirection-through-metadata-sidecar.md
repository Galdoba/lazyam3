---
id: "0015-error-log-redirection-through-metadata-sidecar"
title: "Error Log Redirection Through The Metadata Sidecar"
description: "A generated render script copies the ffmpeg/fflite error log from the in-progress directory into the render output directory and into a metadata sidecar carrying the full log text. It copies, never moves, and never touches the notification directory."
status: accepted
date: "2026-11-17"
scope: cross-package
---

# Error Log Redirection Through The Metadata Sidecar

## Context

A render that succeeds can still produce a non-fatal error log: ffmpeg/fflite writes `<SOURCE>.#err` next to the media in the in-progress directory. The operator's notification handler scans in-progress every five seconds, announces each `.#err` it finds, and removes it from that directory. Nothing in a lazyam3-generated script ever put that log beside the finished render, so the log could not be correlated with the deliverables it describes, and the storage record gave no trace that a render had problems.

The constraints that shape the fix:

- The generator and the executor are different machines. Every path in `executorPaths` is embedded into the script as a string and is never resolved by the generator process, so the render output directory is not observable from the pipeline.
- The metadata directory is reachable from both hosts and already carries the `.done` marker, making it the established trust boundary between script and pipeline.
- The generated script runs under `set -o errexit` and `set -o nounset`, with post-render steps chained by `&&`. An error log only ever appears after a successful ffmpeg exit, so it can live inside that chain without needing a failure path.
- The operator's notification handler is fixed and serves renders beyond lazyam3's. It owns in-progress cleanup.

## Decision

The generated script gains one post-render block, appended to the render chain with `&&`:

```bash
&& OUTBASE="<OUTBASE>" && \
   if [ -f "${IN_PROGRESS}/${SOURCE}.#err" ]; then \
       cp -f "${IN_PROGRESS}/${SOURCE}.#err" "<TARGET_DIR>/${OUTBASE}.#err"; \
       cp -f "${IN_PROGRESS}/${SOURCE}.#err" "${METADATA_DIR}/${SCRIPT_NAME}.err"; \
   fi && \
touch "<TARGET_DIR>/${OUTBASE}.ready"
```

- The log is copied to the render output directory as `<OUTBASE>.#err`, beside the deliverables, and to the metadata directory as `<SCRIPT_NAME>.err` holding the complete log text. The sidecar is the script-to-pipeline channel for error reporting.
- `cp`, never `mv`, and never a delete: the in-progress directory belongs to the operator's tooling, which collects on its own schedule.
- A clean render finds no `.#err`, the `if` block exits zero, and the chain survives. The ready marker is always placed in the target directory.
- No error-archive lookup and no new `errors` executor path. The script only reads from in-progress.
- Generated scripts stop writing anything into the notification directory; the handler's own scanning and announcement behaviour is unchanged.

## Rationale

Copying into the target directory is the only way the log can reach the deliverables, because the generator process cannot resolve executor-side paths after the fact. The metadata directory is the only location both hosts can see, and the `.done` marker already establishes that channel, so a sidecar there reuses an existing contract instead of inventing a new one. Moving the log would race the notification handler that legitimately owns that directory. Carrying the full text rather than a reference keeps the eventual store record self-contained and independent of wherever the executor's copy ends up.

## Consequences

- Generated scripts on existing executor hosts change shape. Any host-side customization of the post-render section must be reviewed.
- If the notifier sweeps the log before the script reaches its copy step, the copy is missed and lazyam3 records nothing about that failure. The handler's archive copy on the executor remains available for manual inspection.
- The notification directory no longer receives markers from generated scripts, though it remains a required executor path in the configuration.
- Error log text is duplicated on the generator host (metadata sidecar, plus the target and archive copies).
- Hard render failures are unaffected: a non-zero ffmpeg exit still stops the chain and leaves `.processing` in place with no record at all.

## Options

- **Pipeline copies the log after the render.** Rejected: the render directory is executor-side and the generator host cannot resolve those paths.
- **Script moves the log instead of copying it.** Rejected: the operator's notification handler owns in-progress cleanup, and moving would remove the input the notifier scans for.
- **Have the script read the handler's error archive directory.** Rejected: it adds a new executor path and couples lazyam3 to the handler's naming and timing.
- **Sidecar records only a reference to the log.** Rejected: the reference is meaningless on the generator host once the executor's copy has been archived or deleted.
