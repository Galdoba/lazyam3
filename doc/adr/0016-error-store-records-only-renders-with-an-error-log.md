---
id: "0016-error-store-records-only-renders-with-an-error-log"
title: "Error Store Records Only Renders That Produced An Error Log"
description: "The append-only error log store receives an entry only when the render left an error sidecar in the metadata directory. Clean renders write no entry, so every entry in the store is a real failure."
status: accepted
date: "2026-11-17"
scope: cross-package
---

# Error Store Records Only Renders That Produced An Error Log

## Context

lazyam3 gained a dedicated JSONL error log store alongside the processing history store. Each entry carries `script_name`, `source_dir`, `error_text`, and `timestamp`.

The first implementation appended an entry for every render whose `.done` marker was consumed, leaving `error_text` empty for a render that produced no error log. That made the store twice as large as the history store, filled it mostly with entries carrying no information, and forced any consumer to filter empty records to find the failures it cared about.

The signal that a render actually failed already exists: the `<SCRIPT_NAME>.err` sidecar the script writes into the metadata directory, and which only ever exists when the render left an error log.

## Decision

The pipeline reads `<SCRIPT_NAME>.err` from the metadata directory while committing the processing history entry for that render, and appends an error store entry only when the sidecar exists. A missing sidecar is treated as a clean render: no entry, no placeholder, no empty field.

Ordering and atomicity are unchanged: the history entry is written first, then the error store entry, inside the same pass over the consumed markers. If either write fails the whole transaction aborts, the `.done` marker and the sidecar are left in place for a later pass, and the source directory is not sanitized. When the commit succeeds, the marker and the sidecar are removed together in the same step.

## Rationale

The existence of the sidecar is authoritative and local: it is written by the process that observed the render, and it cannot exist for a clean render. Keying the store on that fact makes every entry actionable, which is what a separate error store is for. An empty `error_text` record asserts nothing beyond what the history store already asserts, so it costs disk and adds no signal.

## Consequences

- Store volume is proportional to failures rather than to throughput.
- The store no longer proves that a render was examined. A render whose sidecar was deleted before the pipeline read it is indistinguishable from a clean one, and that gap is not reported anywhere.
- An aborted transaction can leave a history entry for a render and no error entry; the retry on the next pass appends the history entry again, so a store failure can produce a duplicated history record.
- Sidecar and `.done` share one cleanup step, so a stale sidecar can never be read without its matching marker.
- lazyam3 only records the log text. No analysis, aggregation, or correlation is performed.

## Options

- **Append an entry for every processed render, empty `error_text` for clean ones.** Rejected: doubles store volume, and every consumer has to filter noise to find failures.
- **Derive the entry from the `<OUTBASE>.#err` copy in the target directory.** Rejected: the target directory is executor-side and unreadable from the generator process.
- **Write a placeholder entry when the sidecar is absent, marking the error as unknown.** Rejected: such an entry is indistinguishable in practice from a clean render that was mislabelled, and it reintroduces the noise the change removes.
