---
id: "0005-dry-run-behavior-design"
title: "Dry Run Behavior Design"
description: "The dryRun flag skips script build and file writes entirely, allowing preview of processing decisions without executing the pipeline, while still emitting events and counting processed units."
status: accepted
date: "2026-11-07"
scope: cross-package
---

# Dry Run Behavior Design

## Context

The pipeline supports a dry run mode where the actual file operations and script generation are skipped. This mode is used for testing, previewing processing decisions, and validating configuration without side effects. The dry run flag affects the entire pipeline flow, from metadata lookup to script building and file preparation.

## Decision

When `dryRun` is true, the pipeline skips script build and file preparation entirely. After metadata lookup and script config assembly, the pipeline emits `EventUnitDone` and returns without calling `appscript.Build` or moving source files to the buffer directory. The unit is still counted as processed, and events are still emitted, so the dry run produces the same event output as a live run without the side effects.

## Rationale

Dry run mode is intended for previewing and testing: it should show what the pipeline would do (which units, which metadata, which script names) without executing the actual work. Skipping script build and file writes ensures no side effects, while still emitting events and counting units preserves the observable behavior of the pipeline for tools that consume events. This makes dry runs useful for validation without risking corruption of input files or generation of artifacts.

## Consequences

- Dry runs do not write scripts or move source files, ensuring no side effects.
- Events are still emitted for each unit, allowing event consumers to observe dry run behavior.
- Units processed in dry run mode are counted as processed in `RunOutput.Processed`.
- Metadata fallback naming is still applied, so dry runs show the same naming decisions as live runs.

## Options

- **Execute script build but skip file writes in dry run.** Rejected: this would still generate scripts and risk side effects, defeating the purpose of dry run.
- **Skip metadata lookup in dry run.** Rejected: metadata lookup is needed to preview naming decisions, so it must still run.
- **Not support dry run at all.** Rejected: dry run is a valuable feature for testing and previewing pipeline behavior without side effects.
