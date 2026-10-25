# HANDOFF

## Exploration: `internal/domain/script/`

### Refactoring Options

#### S4 — Split god `script` builder module

- **Problem:** `internal/domain/script/script.go` (~364 lines) fuses domain types, section builders, validation, placeholder engine and fallback prediction logic.
- **Proposal:**
  - `script.go` — public types (`Script`, `StreamSelection`) + `BuildScript` orchestration.
  - `subtitles.go` / SRT section builders.
  - `hsub.go` / HSUB section builders.
  - `audio.go` / audio builders & track selection.
  - `validate.go` / validation + domain errors.
  - `placeholders.go` / template placeholder engine (`replaceAll`, `findUnresolved`, regex/placeholder format).
- **Effort:** 2-4 hours.
- **Risk:** low (public API stays unchanged).

#### S5 — Make `BuildScript` pure and use single replacement pass

- Stop mutating `cfg.Predictions = predictions`; derive an effective predictions value locally.
- Replace the chained `replaceAll` calls with a single pass over a `map[string, string]` of placeholder values.

#### S6 — Decompose `ScriptConfig` god struct

- Split `Filenames`, `Streams`, `Srt`, `HsubFile`, `Product`, predictions-related naming fields, `Env`, `FfmpegBinary`, `Harsub`, `Destination` into cohesive sub-configs (naming/predictions, env, ffmpeg, harsub) or typed sub structs.

#### S7 — Typed script domain errors

- Replace `fmt.Errorf("script: ...")` + printf-style placeholder strings with dedicated domain error types (`ScriptError`, `ValidationError`).

### Dead code / cleanup candidates

- Deprecated audio facts in `internal/domain/script/audio.go` (`selectAudioTracks`, `buildOutputFiles`) → candidates for removal after callname migration.

### Test organization candidates

- `internal/domain/script/script_test.go` (1824 lines) mixes validation, placeholder, section builder and integration tests.

### What to Watch For

- Next refactoring candidates surfaced while the script domain is explored.
