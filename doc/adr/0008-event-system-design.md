---
id: "0008-event-system-design"
title: "Event System Design"
description: "The pipeline emits PipelineEvent signals (UnitDone, UnitFailed, ScanDone) to a channel, which are consumed by a background goroutine in the CLI command for logging and analytics, ensuring events are drained before the summary is printed."
status: accepted
date: "2026-11-07"
scope: cross-package
---

# Event System Design

## Context

The pipeline needs to communicate progress and results to the CLI layer without tightly coupling the pipeline to CLI-specific logic. Events are emitted for unit completion, unit failure, and scan completion. The CLI layer needs to consume these events for logging, analytics, and potentially TUI updates, while ensuring events are processed in order before the summary is printed.

## Decision

The pipeline emits `PipelineEvent` signals to a channel with capacity 100. Events include `EventUnitDone`, `EventUnitFailed`, and `EventScanDone`, each with a unit ID and optional error. The CLI command (`cmd_process.go`) starts a background goroutine that reads from the events channel, logs unit done and unit failed events, and signals completion via a `eventsDone` channel. The pipeline runs, then the events channel is closed and the consumer waits for `eventsDone` before printing the summary, ensuring pending events are drained in order.

## Rationale

The event system decouples the pipeline from the CLI layer, allowing the pipeline to be tested and reused independently of CLI output. The buffered channel with capacity 100 prevents blocking when events are emitted faster than they are consumed, while the background goroutine ensures events are processed in order. Draining events before printing the summary ensures that log output matches the actual processing order, avoiding misleading summaries.

## Consequences

- The pipeline emits events to a channel, decoupling it from CLI logic.
- Events are consumed by a background goroutine, processed in order.
- Pending events are drained before the summary is printed, ensuring accurate log order.
- The event system supports future consumers like analytics or TUI updates.

## Options

- **Call CLI logging directly from the pipeline.** Rejected: this would couple the pipeline to CLI logic, making it harder to test and reuse.
- **Use synchronous logging in the pipeline.** Rejected: synchronous logging blocks the pipeline and makes it harder to add asynchronous consumers like TUI.
- **Buffer events in memory without a channel.** Rejected: a channel allows the background goroutine to process events independently, enabling future consumers.
