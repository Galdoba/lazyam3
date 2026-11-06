---
id: "0003-path-module-vs-filepath-module-in-domain"
title: "Path Module vs filepath Module in Domain"
description: "Use path.Join() and path.* functions for internal path construction in the domain layer instead of filepath.ToSlash(filepath.Join(...)), since paths are already forward-slash normalized internally."
status: accepted
date: "2026-11-07"
scope: architectural
---

# Path Module vs filepath Module in Domain

## Context

Domain functions (names_output.go, script.go) construct and manipulate paths that are already forward-slash normalized. The original code used `filepath.ToSlash(filepath.Join(...))` which is redundant when paths are normalized.

## Decision

Use `path.Join()`, `path.Dir`, `path.Base`, `path.Ext` for internal path construction in domain layer.

## Rationale

Paths are already forward-slash normalized internally. `path.*` functions operate on forward-slash strings and are the correct choice for constructing normalized path strings. `filepath.*` functions are OS-specific and only needed for actual OS calls (e.g. PathBuilder.native()).

## Consequences

- Domain code is cleaner and avoids redundant ToSlash calls
- Variables named `path` had to be renamed to avoid shadowing the `path` module

## Options

- Continue using `filepath.ToSlash(filepath.Join(...))` everywhere
- Use `path.Join()` and `path.*` functions for internal path construction
