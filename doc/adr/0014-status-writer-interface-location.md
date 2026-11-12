---
id: "0014-status-writer-interface-location"
title: "StatusWriter Interface Location"
description: "The StatusWriter port interface is defined in the pipeline dto.go alongside Logger, with the concrete StdStatusWriter implementation in platform/status.go."
status: accepted
date: "2026-11-12"
scope: cross-package
---

# StatusWriter Interface Location

## Context
The pipeline needs a status bar port (StatusWriter) in addition to the existing structured logging port (Logger). The decision concerns where to define the StatusWriter interface and where to place its concrete implementation.

## Decision
The StatusWriter interface is defined in `internal/app/pipeline/dto.go` alongside the Logger interface. The concrete `StdStatusWriter` implementation lives in `internal/platform/status.go`.

## Rationale
This is consistent with the existing Logger port placement: the pipeline layer owns its ports (interfaces), while the platform layer provides stateless implementations. The platform implementation satisfies the pipeline.StatusWriter interface via Go structural typing.

## Consequences
- The pipeline imports platform for the StdStatusWriter concrete type.
- The platform implementation satisfies pipeline.StatusWriter interface via structural typing.

## Options
- Define interface in `internal/platform/status.go`
- Define interface in `internal/app/pipeline/dto.go` alongside Logger
