---
id: "0010-dashboard-analytics-vision"
title: "Dashboard & Analytics Vision"
description: "Future vision for a TUI and/or web/SSE dashboard that surfaces pipeline processing history, including 'replacement' (episode processed before) detection, with analytics as its own domain separate from the pipeline's append-only JSONL store."
status: vision
date: "2026-11-08"
scope: future-feature
---

# Dashboard & Analytics Vision

## Context

The pipeline (`internal/app/pipeline/pipeline.go`) writes processing history to an append-only JSONL store (`internal/infra/persistence/jsonstore/jsonstore.go`) via `captureProcessingHistory`. Each line encodes a `.done` payload: `source_dir`, `unit_id`, `script_name`, `product_name`, `created_at`, `initial_files`, `created_files`. The store has no readers in production; consumers must parse the JSONL file externally (per ADR 0009).

## Future Vision

A future dashboard (TUI and/or web/SSE) to surface processing history, specifically to show when a product is a "replacement" (an episode processed before, i.e. the same `unit_id` appears in multiple JSONL lines).

## Analytics as Its Own Domain

Analytics/dashboard is its own domain, separate from the pipeline's append-only history store. The JSONL store remains the immutable audit trail. The dashboard needs a proper query surface.

## Data Source Options (to evaluate later)

- **Option A: Parse JSONL directly** — dashboard reads `storage.json`, groups lines by `unit_id`, counts occurrences. O(n) scan; fine for low volume; no real-time.
- **Option B: Derived analytics index** — a lightweight store (small JSON or SQLite) mapping `unit_id → {count, last_processed_at}`. O(1) lookup; no real-time.
- **Option C: Pipeline event channel** — SSE server wraps `Pipeline.events` channel; dashboard subscribes to `EventUnitDone`/`EventUnitFailed`. Real-time; zero JSONL parsing.

## Recommended Approach (provisional)

Keep JSONL as the immutable audit source. For the dashboard: use SSE over `Pipeline.events` for real-time state, and a lightweight unit-index (`unit_id → {count, last_processed_at}`) for "replacement" lookup (unit_id with count > 1 is a replacement; latest line's `created_at` is last processed time).

## Domain Boundaries

- **Pipeline domain:** writes JSONL history, emits events. No dashboard knowledge.
- **Analytics domain:** reads JSONL (audit) + index (query), computes replacement status.
- **Presentation domain (TUI/web):** consumes analytics domain via SSE or polling. No pipeline or JSONL coupling.

## Status

No implementation needed until a TUI layer is created. This is a backlog/vision note only.
