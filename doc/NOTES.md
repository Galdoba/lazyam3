# NOTES

## Unexpected discoveries

### D2 — Builder trailing separator bug

- `internal/domain/script/script.go` — `buildOutputFilesPredictions` appends the shell continuation `` \" `` to every generated line, including the last one.
- **Discovery:** trailing continuation can produce invalid shell when joined/rendered; candidate fix is conditional separator handling.

### D3 — Fallback predictions helper is test-only

- `internal/domain/script/script.go` — `derivePredictionsFromStreams` is documented as ``backward compat for tests``.
- **Discovery:** helper is only used by tests / fallback paths; candidate is removal once all callers provide predictions.

### D4 — `BuildScript` mutates input

- `BuildScript` assigns `cfg.Predictions = predictions` — mutating domain config.
- **Discovery:** pure-function expectation violated; candidate is derive effective config locally.
