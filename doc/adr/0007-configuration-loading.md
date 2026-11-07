---
id: "0007-configuration-loading"
title: "Configuration Loading"
description: "Configuration is loaded via appcontext configmanager from file, environment, and defaults, with file taking precedence over environment and environment over defaults, and shared as a container through AppModules."
status: accepted
date: "2026-11-07"
scope: cross-package
---

# Configuration Loading

## Context

The pipeline requires configuration for paths, ffprobe binary, metadata path, hardsub tag, ffmpeg binary, and dormancy period. Configuration can come from a file, environment variables, or defaults. The CLI accepts flags that override file configuration. The configuration must be shared across the pipeline, ingest, metadata, and script rendering components without circular dependencies.

## Decision

Configuration is loaded via the `appcontext` configmanager package, which reads from a TOML file, environment variables, and built-in defaults. File configuration takes precedence over environment, and environment takes precedence over defaults. The CLI flags override file configuration (e.g. `--source-root`, `--target-root`, `--ffprobe-path`, `--metadata-path`). The loaded configuration is stored in `AppModules.Config` and accessed by commands to assemble the pipeline options. Path resolution uses helper methods like `Paths.GeneratorInput()`, `Paths.GeneratorOutput()`, and `Paths.Executor` to derive the appropriate paths for the generator and executor filesystems.

## Rationale

Using the configmanager package provides a consistent, tested mechanism for loading configuration from multiple sources with well-defined precedence. Sharing configuration through `AppModules` avoids circular dependencies between the CLI, pipeline, and infrastructure layers. The precedence (file > environment > default) ensures that explicit configuration in a file or via flags takes precedence over defaults, while still allowing sensible defaults for testing and development.

## Consequences

- Configuration is loaded from file, environment, and defaults with file taking precedence.
- CLI flags override file configuration for paths and binaries.
- Shared configuration is accessed through `AppModules.Config`, avoiding circular dependencies.
- Path resolution helpers ensure generator and executor paths are correctly derived.

## Options

- **Use a custom configuration loader.** Rejected: the configmanager package provides tested precedence and TOML parsing.
- **Load configuration only from environment variables.** Rejected: file configuration is needed for reproducible, version-controlled setup.
- **Pass configuration as flat parameters to pipeline constructors.** Rejected: `AppModules` provides a shared container that avoids parameter proliferation and circular dependencies.
