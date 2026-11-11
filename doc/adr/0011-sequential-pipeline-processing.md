---
id: "0011-sequential-pipeline-processing"
title: "Sequential Pipeline Processing"
description: "Pipeline units are processed sequentially rather than in parallel, because script preparation is fast enough that parallelism adds unnecessary complexity without meaningful throughput improvement."
status: accepted
date: "2026-11-09"
scope: cross-package
---

# Sequential Pipeline Processing

## Context

The pipeline orchestrates a multi-stage processing chain per media unit: ingest (file scanning + ffprobe), metadata resolution, script configuration, file preparation, and script rendering. A natural optimization for throughput is to process multiple units in parallel using worker pools or fan-out/fan-in patterns.

## Decision

Units are processed sequentially within each scan pass. The `RunOnce` method iterates over discovered unit directories with a single for-loop, processing each unit completely before starting the next. Parallelism is not introduced at the pipeline level.

## Rationale

Script preparation (ingest, metadata lookup, config build, render) is fast enough that parallelism does not meaningfully improve throughput:

- **Script preparation is fast**: The pipeline's processing steps (ffprobe, metadata lookup, template rendering) complete quickly per unit. The bottleneck is not the preparation phase.
- **External processing is slower**: The actual media processing (rendering scripts executed by external tools like ffmpeg) takes significantly longer than preparation. Preparing scripts faster for multiple units in parallel does not reduce total wall-clock time.
- **Source upload is slower**: Source files are uploaded to the input directory, and the upload process is the dominant time consumer. Parallel preparation cannot outpace the upload rate.
- **Complexity cost**: Introducing parallelism adds goroutine lifecycle management, error aggregation, ordering non-determinism, and concurrency bugs (data races, channel leaks). The complexity cost outweighs the negligible throughput gain.

## Consequences

- **Lower complexity**: No goroutine lifecycle management, no channel ownership concerns, no need for `errgroup` or `sync.WaitGroup` in the pipeline loop.
- **Deterministic ordering**: Units are always processed in filesystem enumeration order, making debugging and reproducibility straightforward.
- **No throughput gain from parallelism**: Total processing time is dominated by external tool execution and source upload, not preparation. Parallel preparation would not reduce wall-clock time.
- **Simpler error handling**: Failures are straightforward to track and report without aggregating results from concurrent workers.
- **Predictable resource usage**: No risk of resource exhaustion from too many concurrent ffprobe processes or file operations.

## Options

- **Process units in parallel with a worker pool.** Rejected: adds goroutine lifecycle complexity, non-deterministic ordering, and error aggregation overhead for negligible throughput gain. The bottleneck is external processing and upload, not preparation.
- **Use a channel-based fan-out/fan-in pipeline.** Rejected: same reasons as worker pool, plus additional channel ownership and buffering complexity.
- **Add parallelism only when processing very large unit sets (>1000).** Rejected: the bottleneck is external processing and upload, not preparation. Even with many units, parallel preparation does not reduce wall-clock time.
