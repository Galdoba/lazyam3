---
id: "0001-fso-logger-injection"
title: "Logger Injection Through FSO Package"
description: "Add explicit logger injection through fso.Options and fso.Prober to enable debug-level logging in file scanning and media probing, with type-assertion-based conversion from pipeline.Logger interface to concrete *logging.Logger."
status: accepted
date: "2026-01-11"
scope: cross-package
---

# Logger Injection Through FSO Package

## Context

The `fso` package (filescanner and ffprobe) had no logging infrastructure. When debug-level logging was added across the pipeline, the `fso` package needed the ability to emit debug messages at scan completion, probe start, and probe result status points. The package had no logger parameter, no global logger, and no existing pattern for logging.

## Decision

Add a `Logger *logging.Logger` field to `fso.Options` and a `logger *logging.Logger` field to the `Prober` struct. Callers in `pipeline.go` and `ingest.go` pass the logger when constructing `fso.Options` and calling `NewProber`. The `pipeline.Logger` interface (abstract) is converted to the concrete `*logging.Logger` via type assertion in `pipeline.go` before being passed to `fso.Scan`. When the assertion fails, fso logging is silently skipped.

## Rationale

This approach keeps logging explicit and testable without introducing global state, matching the existing dependency-injection pattern used in the app and infra layers. The type-assertion conversion avoids changing the `pipeline.Logger` interface, preserving the abstraction while allowing fso debug logging when a concrete logger is available.

## Consequences

- Callers of `fso.Scan` and `NewProber` must pass the logger through `fso.Options`.
- `pipeline.go` and `ingest.go` were updated to thread the logger through.
- `ffprobe_test.go` calls to `NewProber` were updated to pass `nil`.
- If `pipeline.Logger` is not a `*logging.Logger` (e.g. a custom implementation), fso logging is silently skipped, which is a known limitation.
- ffprobe probe debug logging captures probe start and result status but not stream counts, since `ProbeFile` returns raw JSON without parsed stream information.

## Options

- **Add a logger parameter to `Scan()` and `NewProber()` signatures directly.** Rejected - would require changing function signatures across all callers and tests; using `Options` struct is more extensible.
- **Use a global logger in the fso package.** Rejected - introduces global state, reduces testability, and contradicts the project's dependency-injection pattern.
- **Skip logging in fso entirely.** Rejected - loses visibility into file scanning and media probing decisions, which are important for debug-level diagnostics.
