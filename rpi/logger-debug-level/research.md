# Logger Debug Level - Where to Add/Change Logging Calls Research

## Problem Context

- Restated problem: The user configured `level = 'debug'` in `[logger]` TOML section, but the entire codebase has zero Debug-level log calls in production. All existing log calls are Info, Warn, or Error. This means setting debug level produces no additional output.
- Business/functional intent: When debug level is enabled, the user expects visibility into per-unit processing decisions, file operations, metadata lookups, script generation details, and storage operations - all the internal mechanics that are invisible at Info level.
- Current vs desired behavior: Currently, Info calls fire for lifecycle events (pipeline started/stopped) and one-per-unit events (processing, dirs found). Desired: Info calls that fire per-unit should be downgraded to Debug; new Debug calls should be added at key decision points; lifecycle events should remain at Info.
- Constraints: Must follow existing logging conventions (`logger.Info("msg", "key", val)`); must not change the logging infrastructure (it works correctly); must not break any existing tests.

## Affected Files

```
internal/app/pipeline/pipeline.go:158-164
internal/app/pipeline/pipeline.go:187
internal/app/pipeline/pipeline.go:246
internal/app/pipeline/pipeline.go:266
internal/app/pipeline/pipeline.go:196-212
internal/app/pipeline/pipeline.go:270-328
internal/app/pipeline/pipeline.go:335-349
internal/app/pipeline/pipeline.go:362-388
internal/app/pipeline/config.go:28-141
internal/app/ingest/ingest.go:31
internal/app/script/build.go:15,28
internal/app/app.go:59
internal/infra/commands/cmd_process.go:71,134
internal/infra/metadata/repository.go:34,61
internal/infra/persistence/jsonstore/jsonstore.go:122,135
internal/infra/fso/filescanner.go:73-133
internal/infra/fso/ffprobe.go:41-57
internal/domain/script/script.go:31-107
internal/domain/callname/names_output.go:36-121
internal/domain/callname/names_input.go:91-165
internal/app/pipeline/pipeline.go:220-242
```

Explanatory prose:

- **internal/app/pipeline/pipeline.go:158-164** - `scan` and `dirs found` are per-scan calls that fire every polling cycle. At Info level they flood the log during continuous polling. Should be Debug.
- **internal/app/pipeline/pipeline.go:187** - `sanitized empty dirs` fires per-scan, after sanitize. Should be Debug.
- **internal/app/pipeline/pipeline.go:246** - `processing` fires once per unit directory. Should be Debug.
- **internal/app/pipeline/pipeline.go:266** - `dry run: script build skipped` fires once per unit in dry-run mode. Should be Debug.
- **internal/app/pipeline/pipeline.go:196-212** - The main per-unit loop has no Debug calls. Need Debug for: loop entry per directory, `snap.FilesIn(d.Path)` count, `processUnit` result (success/failure).
- **internal/app/pipeline/pipeline.go:270-328** - The `processUnit` body has no Debug calls. Need Debug for: ingest result (stream count, file count), metadata lookup result (found/not-found), script config build result, file prepare (count of files moved), script build result, artifact write.
- **internal/app/pipeline/pipeline.go:335-349** - `prepareFiles` moves files but has no logging. Need Debug for: each file rename operation.
- **internal/app/pipeline/pipeline.go:362-388** - `findDoneFile` iterates .done files but has no Debug. Need Debug for: files scanned, matches found, payloads parsed.
- **internal/app/pipeline/pipeline.go:220-242** - `captureProcessingHistory` iterates matches and stores them. Need Debug for: keys stored, save result.
- **internal/app/pipeline/config.go:28-141** - `buildScriptConfig` is the heart of script generation. Needs Debug for: stream count, predictions, call names, subtitle classification, executor paths, script destination.
- **internal/app/ingest/ingest.go:31** - `ingesting directory` fires per-unit. Should be Debug.
- **internal/app/script/build.go:15,28** - `building rendering script` and `writing rendered script` fire per-unit. Should be Debug.
- **internal/app/app.go:59** - `modules initialized` fires once at startup. Keep as Info.
- **internal/infra/commands/cmd_process.go:71** - `create pipeline` fires once per command invocation. Keep as Info.
- **internal/infra/commands/cmd_process.go:134** - `unit done` fires per-unit. Should be Debug.
- **internal/infra/metadata/repository.go:34,61** - `metadata repository created` fires once. Keep as Info. `metadata reloaded` fires per polling cycle. Should be Debug.
- **internal/infra/persistence/jsonstore/jsonstore.go:122,135** - `store saved entries` and `store closed and saved` fire per-save/close. Should be Debug.
- **internal/infra/fso/filescanner.go:73-133** - Scan is the foundation of every pipeline pass. Needs Debug for: root path, files found, dirs found, writable checks.
- **internal/infra/fso/ffprobe.go:41-57** - Each file probe fires per file. Needs Debug for: file path, probe result, streams found.
- **internal/domain/script/script.go:31-107** - Script building has no logging. Needs Debug for: validation result, interlaced detection, audio tracks, HSUB detection, placeholder replacements.
- **internal/domain/callname/names_output.go:36-121** - Output prediction is called per-unit. Needs Debug for: predictions generated, HSUB detected.
- **internal/domain/callname/names_input.go:91-165** - Input naming is called per file. Needs Debug for: names generated.

## Code Examples

### Before (Info-level, too verbose for per-unit):

```go
// internal/app/pipeline/pipeline.go:158
p.logger.Info("scan")
// internal/app/pipeline/pipeline.go:164
p.logger.Info("dirs found", "count", len(dirs))
// internal/app/pipeline/pipeline.go:246
p.logger.Info("processing", "dir", dir)
// internal/app/ingest/ingest.go:31
ing.logger.Info("ingesting directory", "dir", dir)
```

### After (downgraded to Debug):

```go
p.logger.Debug("scan")
p.logger.Debug("dirs found", "count", len(dirs))
p.logger.Debug("processing", "dir", dir)
ing.logger.Debug("ingesting directory", "dir", dir)
```

### New Debug calls at decision points:

- **processUnit ingest and metadata decisions** - after ingest and after metadata lookup

```go
// internal/app/pipeline/pipeline.go:247 - after ingest
unit, err := p.in.Ingest(dir)
if err != nil {
    // ... error handling
} else {
    p.logger.Debug("ingest complete", "unit_id", unit.ID, "streams", len(unit.Streams))
}

// internal/app/pipeline/pipeline.go:255 - after metadata lookup
prod, err := p.md.FindByID(unit.ID)
if err != nil {
    // ... warn + fallback
} else {
    p.logger.Debug("metadata found", "id", unit.ID, "title", prod.Title)
}
```

- **Script config build, file operations, predictions** - after config build, during file moves, after predictions, after interlaced detection

```go
// internal/app/pipeline/pipeline.go:270 - after script config build
cfg, err := buildScriptConfig(...)
if err != nil {
    // ... error
} else {
    p.logger.Debug("script config built", "unit_id", unit.ID, "predictions", len(cfg.Predictions))
}

// internal/app/pipeline/pipeline.go:335 - in prepareFiles
for orig, call := range callNames {
    src := filepath.Join(sourceDir, orig)
    dst := filepath.Join(bufferDir, call)
    if err := os.Rename(src, dst); err != nil {
        // ... error
    }
    p.logger.Debug("file renamed", "source", src, "destination", dst)
}

// internal/app/pipeline/config.go:62 - after predictions
predictions := callname.PredictOutputs(prod, unit, renderRoot, hardsubTag, prod.ID)
p.logger.Debug("output predictions", "unit_id", unit.ID, "count", len(predictions))

// internal/domain/script/script.go:45 - after interlaced detection
interlaced := getInterlaced(cfg.Streams)
p.logger.Debug("interlaced detection", "streams", len(cfg.Streams), "interlaced", interlaced)
```

### Lifecycle events (keep as Info):

```go
// internal/app/pipeline/pipeline.go:96 - fires once
p.logger.Info("pipeline loop started", "poll_interval", p.pollInterval)
// internal/app/pipeline/pipeline.go:100 - fires once
p.logger.Info("pipeline stopped", "error", ctx.Err())
// internal/infra/commands/cmd_process.go:71 - fires once per command
logger.Info("create pipeline")
// internal/infra/commands/cmd_process.go:152 - fires once
logger.Info("pipeline finished", "processed", out.Processed, "failed", out.Failed)
// internal/app/app.go:59 - fires once
logger.Info("modules initialized")
```

## External Research Findings

- Go's `log/slog` handler filtering: A handler at `LevelInfo` will skip any record at `LevelDebug` regardless of the Logger's level. The `multiHandler.Enabled()` method uses OR logic - if any handler is enabled for a level, the record is dispatched. This means both file and terminal handlers must be at the right level for debug to appear everywhere.
- The `configmanager` library (used for TOML parsing) merges user config into defaults by overwriting present fields. Setting `level = 'debug'` in TOML correctly overwrites the default `"info"`. This was verified in the prior research.
- Best practice for structured logging: per-unit operations should use Debug level; lifecycle events should use Info; warnings/errors for exceptional conditions should use Warn/Error. This follows the principle that Info should be useful in production without being noisy.

## FAR Scale Output

### Factual Score: 5
All findings are directly evidenced in the code:
- Zero Debug calls in production code (verified by grep)
- Info calls at internal/app/pipeline/pipeline.go:158, internal/app/ingest/ingest.go:31, internal/app/script/build.go:15 fire per-unit (verified by reading the call sites in loops)
- Lifecycle Info calls at internal/app/pipeline/pipeline.go:96, internal/app/app.go:59 fire once (verified by reading the call sites outside loops)
- No logging at key decision points (verified by reading processUnit, buildScriptConfig, prepareFiles, findDoneFile)

### Actionable Score: 5
The changes are clear and well-scoped:
- Downgrade N Info calls to Debug (per-unit operations)
- Add M new Debug calls at decision points
- Keep K Info calls as-is (lifecycle events)
- No architectural changes needed

### Relevant Score: 5
This directly addresses the user's problem: enabling debug level currently produces zero additional output because no Debug calls exist. Adding them at the right places gives the user visibility into exactly what they need.

### Mean: 5.00  --> PASS

## Testing Strategy

1. **Unit tests**: Add tests in `logger_test.go` that verify Debug messages are written when level is "debug" and filtered when level is "info".
2. **Integration**: Run `scripts/live_test.sh` with `level = 'debug'` and verify Debug messages appear in both terminal and file output.
3. **Filtering**: Run with `level = 'info'` and verify Debug messages are NOT present.
4. **Regression**: Run `go test -race -cover ./...` to ensure no existing tests break.

## Potential Design Pattern Recommendations

- **Per-unit Debug calls**: Use `p.logger.Debug("event", "unit_id", unit.ID, "detail", value)` pattern consistently.
- **Grouped Debug calls**: For multi-step operations (like `processUnit`), use `p.logger.With("unit_id", unit.ID).Debug("step", step_name)` to group related messages.
- **Config-driven verbosity**: The existing `level = 'debug'` TOML setting already provides the switch. No additional infrastructure needed.
- **Avoid over-logging**: Don't add Debug calls inside tight loops (e.g., per-file rename in prepareFiles) unless the count is small. Use aggregate Debug calls (e.g., "moved 3 files") instead of per-item Debug calls in high-volume loops.

## Assumptions

1. The user expects Debug output during pipeline execution, not just at startup.
2. The user will run with `level = 'debug'` during development/troubleshooting, not in production.
3. Per-unit Debug calls should include enough context (unit_id, file paths) to correlate messages across the pipeline.
4. The existing `logging.Logger.Debug()` method signature (`Debug(msg string, args ...any)`) is the correct pattern to follow.

## Out of Scope

- Changes to the logging infrastructure (logger.go, handler creation).
- Changes to the configmanager library.
- Adding a `Debugf` method (the variadic args pattern is sufficient).
- Changes to test files (the goal is runtime visibility, not test coverage).
- Changes to the ffprobe binary path or ffprobe execution logic.
- Changes to the metadata.json format or schema.
