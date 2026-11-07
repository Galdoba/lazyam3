---
id: "0009-jsonl-append-semantics"
title: "JSONL Append-Only Storage"
description: "The jsonstore.Store was rewritten from an in-memory key-value store with full-file overwrite semantics to an append-only JSONL writer where each Set call appends a single JSON line."
status: accepted
date: "2026-11-08"
scope: cross-package
---

# JSONL Append-Only Storage

## Context

The jsonstore.Store previously held an in-memory map and flushed to a JSON file on Save/Close. The pipeline only needs to record processing history per unit, not read or update existing entries. A new storage approach was needed that avoids full-file rewrites and guarantees atomic per-line writes for small payloads.

## Decision

Rewrote jsonstore.Store to be an append-only JSONL store. Each Set call appends a single JSON line via json.Encoder.Encode to a file opened in O_CREATE|O_WRONLY|O_APPEND mode with a sync.Mutex to serialize writes. The Get, Delete, Keys, Len, and Save methods were removed. Close now syncs the file before closing it and is idempotent. The key parameter in Set is retained for signature compatibility but not used for indexing.

## Rationale

Per-unit history writes are low-volume and the pipeline never reads back from the store during execution. JSONL append semantics avoid full-file rewrites and guarantee atomic per-line writes at the OS level for small payloads via O_APPEND. The sync.Mutex around the Encode call ensures thread-safe encoding and well-formed JSONL lines under concurrent Set calls.

## Consequences

- The store no longer supports reading back entries; consumers must parse the JSONL file externally.
- The Get/Delete/Keys/Len/Save methods were removed from the Store API.
- Reading is handled externally by consumers.
- Close must sync the file before closing to avoid data loss on process crash.

## Options

- **Keep in-memory map with full-file overwrite.** Rejected: full-file rewrites are unnecessary for low-volume history writes and do not provide atomic per-line guarantees.
- **Use a separate log file per unit.** Rejected: a single JSONL store is simpler and sufficient for the low-volume history use case.
