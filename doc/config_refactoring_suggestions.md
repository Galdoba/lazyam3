# Default Configuration Refactoring — Suggestions

## Overview

Feature: `D1` — Default configuration refactoring.

The goal is to give the user a default configuration that lets them start working immediately, while making sure they know exactly which fields they should fill in.

## Problem

- The built-in default configuration in `internal/infra/config/config.go` (`defaultCFG()`) does not let the user start working immediately.
- More importantly, the user does not know which fields in the configuration are available at all.
- Default values look like usable paths (`/tmp/lazyam3/input`) or dummy placeholders (`/path/to/file.log`, `/path/to/file.json`).
- A fresh setup silently runs the pipeline on such defaults: media scanning points to non-existent directories, the metadata path is wrong, the log file path is invalid.

## Proposal

1. Refactor the default configuration so it is self-documenting: every field carries an obvious placeholder value that the user must replace (for example `/path/to/lazyam3/input`).
2. Detect that the effective configuration equals the built-in default (fresh install with no config file, or a config file that only contains default values).
3. Terminate startup with a clear message that points the user to the config file that has to be edited and lists the required fields to fill in.

## Proposed New Default Configuration

| Config field | Proposed default value |
|---|---|
| `paths.generator.input_directory` | `/path/to/lazyam3/input` |
| `paths.generator.output_directory` | `/path/to/lazyam3/output` |
| `paths.generator.metadata_path` | `/path/to/lazyam3/metadata.json` |
| `paths.generator.ffprobe_binary` | `ffprobe` |
| `paths.executor.<key>` | placeholders mirroring the generator output directory |
| `logger.level` | `info` |
| `logger.path` | placeholder log file path |
| `logger.terminal` / `logger.format` / `logger.color` | `true` / `text` / `true` |
| `storage.path` | `/path/to/lazyam3/storage.json` |
| `script_generation.hardsub.tag` | `Forsirovannye` |
| `script_generation.ffmpeg_binary` | `ffmpeg` |

The default values must be obviously placeholders so that opening the generated config file immediately tells the user what to edit.

## Default Detection and Fail-Fast Behavior

When the effective configuration equals the built-in default, the program:

1. Returns a default-configuration error.
2. Prints a message with the location of the config file the user has to edit.
3. Terminates instead of running the pipeline.

Draft error message:

```text
default configuration in use: edit the config file "<config file path>" and fill in the required fields (paths.generator.input_directory, paths.generator.output_directory, paths.generator.metadata_path) to enable processing
```

The config file path is resolved from the `--config` flag when provided, otherwise from the standard location (for example `~/.config/lazyam3/config.toml`).

## Implementation Plan

1. Refactor `defaultCFG()` in `internal/infra/config/config.go` to self-documenting placeholders.
2. Add default-configuration detection (`IsDefault`) on the effective config.
3. Return a typed/sentinel error (`ErrDefaultConfig`) with the config file path on detection.
4. Fail fast in the app startup path before the pipeline is created.

## Tests

- Default config contains self-documenting placeholder values.
- Effective config equal to the default is detected.
- Fresh install (no user config) triggers the default-config error.
- User-edited config does not trigger the default-config error.

## Trade-offs

- The default configuration will no longer be directly usable — the user must always create or edit a config file.
- Equality-based detection also flags user configs that intentionally mirror the defaults.
