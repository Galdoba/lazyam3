---
id: "0018-mandatory-writable-paths-fail-fast-at-startup"
title: "Mandatory Writable Paths, Fail Fast At Startup"
description: "The log, history store, error store, and status file all point at mandatory configuration fields that are validated at load and opened before processing starts, so an unwritable path aborts startup instead of surfacing mid-run."
status: accepted
date: "2026-11-17"
scope: architectural
---

# Mandatory Writable Paths, Fail Fast At Startup

## Context

lazyam3 writes four files: the log, the processing history store, the error log store, and the pipeline status file. Before this change the status path was derived from the history store path, the error store path was optional (an unset path simply meant no error records), and an unwritable path was only discovered at the moment the pipeline tried to write, possibly in the middle of committing a transaction.

The new error store made the question explicit: should a missing or unusable path degrade the run quietly, or stop it?

## Decision

Every writable path is a mandatory configuration field: `logger.path`, `storage.path`, `storage.error_store_path`, and `storage.status_path`. All four are listed in `requiredConfigFields` (so the default-configuration error names them) and rejected by `Config.Validate` when empty, which runs as part of config load.

Each is opened before processing begins:

- the log inside `AppModules.Init`, immediately after the config loads;
- both stores through `AppModules.Store` and `AppModules.ErrorStore`, called eagerly by the process command, with the failure cached by the `sync.Once` so repeated calls report the same error;
- the status file through `AppModules.EnsureStatusFile`, which opens and closes it purely to probe writability.

The pipeline takes the status path from configuration instead of deriving it next to the history store. Any of these failures returns an error from the command action before the pipeline is constructed; there is no optional-store nil-skip and no directory creation on the user's behalf.

## Rationale

A run that dies halfway through committing a transaction is strictly worse than a run that refuses to start: the operator has to work out which markers survived. These files are prerequisites of the pipeline rather than incidental outputs, so opening them early turns a mid-run surprise into a startup error that names the field. Making the paths mandatory rather than optional also keeps the error store honest: an absent path can no longer silently disable error recording.

## Consequences

- Existing configuration files must gain `storage.error_store_path` and `storage.status_path`; a config equal to the built-in default is rejected with a message listing the required fields.
- The status file is no longer colocated with the history store, so deployments that relied on that have to set it explicitly.
- Store paths do not get their parent directories created. A path inside a non-existent directory fails at startup, which is intentional.
- Startup does file I/O that was previously lazy, which is negligible at this scale but changes where the error surfaces.

## Options

- **Keep the error store optional and skip writes when unconfigured.** Rejected: error records would be silently lost, and an operator could not tell a quiet run from a misconfigured one.
- **Derive the status path from the history store directory.** Rejected: implicit coupling with no way to point the status file somewhere else, and the derived location was never validated before use.
- **Create missing parent directories on startup.** Rejected: it hides path mistakes, which is exactly the class of failure this decision is meant to surface.
