---
id: "0012-remove-id-from-output-filenames"
title: "Remove ID from Output Filenames, Use PRT as Unique Identifier"
description: "Output and script filenames embed a PRT timestamp that is guaranteed unique per unit within a process run. The product id (_idN) is removed from filenames; tracking relies on PRT in script_name and the .done JSON markers keyed by source_dir."
status: accepted
date: "2026-11-09"
scope: cross-package
---

# Remove ID from Output Filenames, Use PRT as Unique Identifier

## Context

Output and script filenames currently embed the product id as `_idN`:

- `names_output.go` builds `OUTBASE = baseName + nameTag + "_PRT" + timestamp + "_idN"` (lines 66-74).
- `names_input.go` builds `ScriptInput.String()` with `s += fmt.Sprintf("_id%d", si.ID)` (line 42).

The id serves as the filename-level signal that ties a rendered output back to its source product. The pipeline already guarantees PRT uniqueness per unit within a process run via a mutex-guarded monotonic counter (`pipeline/config.go`, ADR 0010). The pipeline also persists a JSON payload (`.processing`/`.done` markers) containing `source_dir`, `unit_id`, `script_name`, `initial_files`, and `created_files`, and stores it keyed by `script_name` in `captureProcessingHistory`.

The proposed change removes `_idN` from filenames and relies on PRT as the unique identifier, keeping PRT in the output base name and script name.

## Decision

Remove the id component from output and script filenames:

1. In `names_output.go`, drop `if prodID > 0 { outBase += fmt.Sprintf("_id%d", prodID) }` and update the doc comment. The output base becomes `baseName + nameTag + "_PRT" + timestamp`.
2. In `names_input.go`, drop `s += fmt.Sprintf("_id%d", si.ID)` and update the doc comment.
3. Update tests in `names_output_test.go` and `names_input_test.go` that assert `_idN` presence.

PRT remains embedded in:

- Output paths via `PredictOutputs` (called from `pipeline/config.go` line 63 with `prod.ID`, but the id is no longer appended to the name).
- Script destination via `PredictScriptName` (line 86).
- The `.done` payload `script_name` and the store key `script_<script_name>`.

Tracking of "what was made from what" continues via:

- The `.done` JSON marker per unit, keyed by `source_dir` in `findDoneFile`.
- The store entry keyed by `script_name`, which embeds PRT.

## Rationale

PRT uniqueness is process-scoped but format-encodes `YYMMDDHHMMSS`, so cross-run collisions are unlikely. The tracking payload retains `unit_id` and `product_name`, so the filename no longer needs the id to preserve traceability. Synthetic products (filename has no leading digits, `SyntheticID = 9999`) are handled correctly because tracking uses `source_dir` and `script_name`, not filename-extracted ids.

Removing the id simplifies filenames and removes a filename-parsing dependency, while the monotonic PRT generator and `.done` markers provide the authoritative source of truth for source->output mapping.

## Consequences

- Filenames no longer contain the explicit product id; traceability relies on PRT and `.done` markers.
- PRT uniqueness is guaranteed within a single process run; a fresh process restarts from the wall clock. A restart could, in pathological cases (clock reset, same-second reprocessing), generate a PRT that collides with a previous run's `script_name` store key. The pipeline mitigates duplicate processing via `source_dir` matching in `findDoneFile`.
- Test suite changes: `names_output_test.go` and `names_input_test.go` must be updated to drop `_idN` assertions.

## Options Considered

- Keep `_idN` in filenames: preserves explicit product-id signal in the filename, but adds filename-parsing coupling and redundancy with PRT-based tracking.
- Remove id from video/audio only, keep in script/marker name: partial cleanup, leaves script-name id handling, requires metadata tracking.
- Remove id entirely, rely on PRT + `.done` markers (chosen): simplifies filenames, uses existing monotonic PRT guarantee and existing `.done`/store tracking.
