---
id: "0002-storage-timing-done-detection"
title: "Storage Persistence on .done Detection"
description: "Persist storage.json when .done files are detected and the source directory is empty, rather than at script-build time, to avoid false-positive completion signals in the inter-process pipeline."
status: accepted
date: "2026-11-05"
scope: cross-package
---

# Storage Persistence on .done Detection

## Context

Work in the pipeline has two stages: building the script (cheap, ~1s) and processing media via an external executor (minutes to hours). From the stakeholder's perspective, work is not complete until the script finishes processing. Storing storage.json at build time would create a false-positive where storage.json marks work "done" while the external executor is still processing.

## Decision

Rely on `captureProcessingHistory` to persist processing history when `.done` files are detected and the source directory is empty, rather than storing at script-build time.

## Rationale

Storing at `.done` detection avoids false-positives for the inter-process pipeline: `storage.json` only records work that has actually completed. A missed signal is less dangerous than a false positive for the larger pipeline of which lazyam3 is an element.

## Consequences

- `storage.json` is updated at the start of the cycle after `.done` appears (~1s poll interval).
- Build-time data is not recorded, avoiding false positives.

## Options

- Store in `processUnit` at build time (false-positive risk).
- Store in `captureProcessingHistory` when `.done` files are found (execution complete).
