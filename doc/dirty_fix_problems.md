# Dirty Fix Problems (commit cf2d9c5)

Commit `cf2d9c5` ("dirty fix: ingest, callname") solves two problems:

1. **ingest.go:45** — Skips `metadata.json` files during probing. Without this, `metadata.json` was passed to `Prober.ProbeFile()`, which fails on non-media files, aborting the entire directory ingest.

2. **callname/shared.go:9** — Added `time.Sleep(time.Second)` in `NewPRT()`. The PRT string uses format `PRT` + `YYMMDDHHMMSS`; without the sleep, concurrent calls created within the same second would generate identical PRT values, causing collisions.

Note: The sleep is a hacky workaround (as the commit message "dirty fix" implies) to force sequential uniqueness in PRT generation.
