---
id: "0006-script-generation-design"
title: "Script Generation Design"
description: "buildScriptConfig assembles the domain ScriptConfig by sorting streams, computing predictions against the executor's target path, and deriving call names from predictions, with the script destination driven by compact naming format."
status: accepted
date: "2026-11-07"
scope: cross-package
---

# Script Generation Design

## Context

The pipeline generates rendering scripts for each media unit. The script generation involves assembling a `ScriptConfig` from the ingested media unit, metadata product, and configuration. The configuration must account for the executor's filesystem layout, where buffer, in_progress, done, notifications, archive, and target paths differ from the generator's output directory. Predictions drive all downstream logic, including call names and script naming.

## Decision

`buildScriptConfig` sorts streams by `(FileIndex, Index)`, computes predictions against the executor's target path (so ffmpeg output paths are accessible to the executor), derives call names from unit filenames, and resolves SRT and HSUB files to their call names. The script destination is named using a compact naming format (e.g. `Show_s01e06_PRT261023120000.sh`) extracted from predictions. Environment paths are derived from `executorPaths` with sensible defaults under the destination root.

## Rationale

Script generation must produce paths that the executor can use, which means predictions and destinations must be computed against the executor's filesystem layout rather than the generator's. Sorting streams ensures deterministic output, and predictions driving all downstream logic ensures consistency between the script content and the actual files the executor will process. The compact naming format provides predictable, readable script filenames that encode the product and timestamp.

## Consequences

- Predictions are computed against the executor's target path, ensuring ffmpeg output paths are accessible.
- Streams are sorted deterministically, ensuring consistent script generation across runs.
- Call names are derived from predictions, ensuring the script references the correct files.
- The `newPRTTimestamp` function currently uses `time.Sleep(time.Second)` as a placeholder, which needs to be made more idiomatic.

## Options

- **Compute predictions against the generator's output directory.** Rejected: the executor needs paths relative to its own filesystem layout, not the generator's.
- **Sort streams by stream index only.** Rejected: FileIndex must be considered first to group streams from the same file together.
- **Use full descriptive names for scripts.** Rejected: compact naming format encodes product and timestamp in a predictable, space-efficient way.
