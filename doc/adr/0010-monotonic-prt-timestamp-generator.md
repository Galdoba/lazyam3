---
id: "0010-monotonic-prt-timestamp-generator"
title: "Monotonic PRT Timestamp Generator"
description: "Generated script filenames embed a PRT timestamp, and two units of the same episode built in the same wall-clock second must not collide. The pipeline now guarantees PRT uniqueness with a mutex-guarded monotonic counter instead of the forbidden time.Sleep."
status: accepted
date: "2026-11-08"
scope: cross-package
---

# Monotonic PRT Timestamp Generator

## Context

The pipeline builds a rendering script for each media unit and names it with a PRT timestamp embedded in the file name (for example `Show_s01e06_PRT261023120000.sh`). The timestamp was produced by `newPRTTimestamp` in the pipeline package and, in a separate callname helper, by a second `NewPRT` function. Both used `time.Sleep(time.Second)`, which the forbidigo linter forbids in production code, and both derived uniqueness from the wall clock: two units processed within the same second received the same timestamp and therefore the same script file name. A collision would overwrite one unit's script and break per-unit processing state tracked in the buffer and storage.

## Decision

Replace both sleeps with a single monotonic generator in the pipeline package. A package-level `sync.Mutex` guards a package-level `lastPRTSec int64` counter. Each call reads `time.Now().Unix()`, and when that value is not strictly greater than the stored second it is replaced with `lastPRTSec + 1`. The value is stored back and formatted as a 12-digit local-time string with no PRT prefix. Callers keep the same function and signature, so no interface changes are required.

## Rationale

`sync.Mutex` has a usable zero value, so no lazy initialization or clock abstraction is needed. The critical section holds only integer comparisons and one formatting call, with no I/O, so contention is negligible. The strictly-increasing guard guarantees that every timestamp in a run is unique even when many units are built within the same wall-clock second, while preserving the existing 12-digit local-time format and no-prefix behavior that the domain script-naming layer and the rendered file names depend on. An injectable clock interface was rejected as unnecessary indirection for a single-use correctness guarantee.

## Consequences
- PRT file names stay unique across same-second builds of the same episode, eliminating script overwrites.
- A small amount of package-level mutable state is introduced; it is confined to the pipeline package and only read while the mutex is held.
- The pipeline no longer self-paces with a 1-second delay per unit, which is a deliberate speed-up verified by the end-to-end live test.
- Uniqueness is guaranteed within a single process run; a fresh process restarts from the wall clock.

## Options
- Keep `time.Sleep(time.Second)`: simple but slow and forbidden by forbidigo in production code.
- Use an injectable clock interface: adds indirection the blocker does not need.
- Add a sub-second component to the timestamp: would change the 12-digit format contract used by the naming layer.
