# Backlog

Items to revisit once the core pipeline is stable.

## Remove metadata pruning once `Product` is declared working

The `metadata` domain keeps a `Descendants` flag (`WithDescendants` /
`WithoutDescendants`) that prunes the tree returned by `Root.FindByID` into a
`Found` structure. This pruning only affects the shape of the pruned tree that
`scrips/find` serializes to JSON; the product pipeline never reads it —
`FoundToProduct` only extracts the target node's own fields.

Keep it until the `Product` flat record is confirmed working end-to-end, then
remove:

- the `Descendants` parameter on `Root.FindByID`
  (`internal/infra/metadata/types.go`);
- the `WithDescendants` / `WithoutDescendants` constants and all `*Shallow` /
  clone helpers (`types.go`);
- the `Found` type and the prune logic in `FindByID`
  (`internal/infra/metadata/types.go`);
- the intermediate `Found` return from the infra repository
  (`internal/infra/metadata/repository.go`), which should call
  `FoundToProduct` on a non-pruned result instead.

After removal the repository's `FindByID` becomes a direct node lookup.

## Language not translated in final output

The rendered scripts use the `Product.SeriesTitle` (transliterated) for naming, but the audio output filenames use language codes from stream metadata (`UND`, `RUS`, `ENG`) which are not transliterated. Synthetic test files may lack language data, so outputs fall back to `UND`. When real content has language info, the output suffixes like `_UN D20` should be transliterated to the target language (e.g., `_РУС20`).

## Ingest domain improvements

### Remove commented-out code in `unit.go`

Lines 184-198 of `internal/domain/ingest/unit.go` contain three commented-out items that serve no purpose: `ValidateMediaUnitWithFilepaths`, `MediaUnit.validate()` method, and `ValidationIssue.String()`. Either uncomment and wire them into the codebase or remove them to clean up the domain.

### Fix `parseDuration` precision loss

`internal/domain/ingest/parse.go:100-107` multiplies by 1000 and truncates to `int`, losing sub-millisecond precision. ffprobe duration strings like `"10.123456"` become `"10123/1000"` instead of preserving the original fraction. Consider storing as `Rational{Num: floor(v*denom), Denom: denom}` with a configurable denominator (e.g. 1000000) or store the raw string and parse on demand.

### `parseRational` silently swallows parse failures

`internal/domain/ingest/parse.go:122-131` returns `Rational{}` on any `Sscanf` failure (bad format, division by zero, etc.) with no way to distinguish a valid `0/0` from a parse error. Callers (frame rate, duration) treat a zero Rational as "unknown" but can't tell if it was intentional or a bug. Consider returning `(Rational, error)` or using a sentinel.

### `parseBitRate` and `parseInt` silently return 0 on failure

`internal/domain/ingest/parse.go:133-146` uses `fmt.Sscanf` with `//nolint:errcheck` — malformed bitrate or sample-rate strings silently become 0. Since 0 is a valid "unknown" value, callers can't distinguish "missing" from "corrupt". Consider returning `(int, error)` or `(int64, error)` so the caller can decide how to handle parse errors.

### `Layout` constants are too restrictive

`internal/domain/ingest/stream.go:26-31` hardcodes `LayoutMono`, `LayoutStereo`, `LayoutSurround51`, `LayoutSurround71` but ffprobe reports many more layouts (`4.0`, `5.0`, `stereo swap`, `dolby surround`, custom names). The `Layout` type is a string alias so it accepts any value, but the constants imply these are the only valid ones. Either document the full set or remove the constants and let infrastructure map codec-level layout names.

### Interlaced detection incomplete

`internal/domain/ingest/parse.go:148-153` only checks `field_order` tag for interlacing. ffprobe can also indicate interlaced content via `top_field_first`/`bottom_field_first` disposition flags, or the stream may report `field_order` as `tb`/`bt` (which are interlaced but not handled). Additionally, `Stream.Interlaced` is documented as "not used yet" in the domain docs — either wire it into the script generation (`scrips/video` filter) or remove the field.

### Duration consistency check reports multiple times

`internal/domain/ingest/unit.go:157-178` loops through all durations after detecting >10% variance, appending a new issue for each one. The `break` only exits the inner loop body but the issue is appended before the break, so if three streams are out of tolerance, three identical "duration_consistency" issues are added. Fix by appending once and breaking immediately after the first detection.

### `ValidationIssue.Field` comment is inaccurate

`internal/domain/ingest/unit.go:69` documents Field as `"e.g. 'codec', 'encoding', 'readiness'"` but actual usage sets it to `"resolution"`, `"bitrate"`, `"duration"`, `"coordinate"`, `"duration_consistency"`. Update the doc comment to match reality.

### `parseStreams` should handle format-level metadata

`internal/domain/ingest/parse.go:53` has a TODO noting it should "accept bytes from format and multiple streams". The current implementation only reads `streams` from the ffprobe JSON — the `format` section (which contains container-level metadata like `duration`, `bit_rate`, `nb_streams`) is ignored. Consider parsing format-level data to enrich streams or validate consistency (e.g., reported total bit-rate vs. sum of stream bit-rates).

## Metadata domain improvements

### Missing fields in `Product` from JSON catalog

`internal/domain/metadata/types.go` Product struct omits several fields present in the JSON catalog (`Studios`, `Persons`, `ReleaseYear`, `AgeRestrictions`). The `PersonsItem` struct and `Persons` JSON field in both `SeriesItem` and `MoviesItem` are parsed but never surfaced to `Product`. Consider adding `Studios`, `Persons`, `ReleaseYear`, and `AgeRestrictions` to `Product` if downstream consumers (script generation, naming) would benefit from them.

### `ProductRepository` only supports `FindByID`

The port (`internal/domain/metadata/port.go`) defines only `FindByID(id int64)`. This is sufficient for the current pipeline (which resolves by the ID extracted from filenames), but limits flexibility. Consider adding `FindByTitle(title string)` or a `Search(query SearchQuery)` method if future features need title-based lookup, genre filtering, or catalog browsing.

### `WithoutDescendants` in `Root.FindByID` is unused in pipeline

The `Root.FindByID` method accepts a `Descendants` flag, but the infrastructure repository always calls it with `WithDescendants` (`internal/infra/metadata/repository.go:35`). The `scrips/find` CLI tool is the only consumer using `WithoutDescendants` via the `-shallow` flag. Document this or consider if the pruning logic is still needed (it is already listed in backlog item 1).

### `ProductRepository.FindByID` does not guard against nil `Product`

`FoundToProduct` can return `nil` when the pruned tree has empty collections (e.g., `KindSeries` with `[]SeriesItem{}`). The pipeline at `internal/app/pipeline/pipeline.go:123` checks the error but does not check if `prod` is nil before using it. If a corrupted or malformed JSON produces a node with empty children, the pipeline would panic on dereference. Add a nil check after `FindByID`.

### Inconsistent JSON field naming in catalog types

`internal/infra/metadata/types.go` has inconsistent `json` tags: `SeriesItem.ImdbId` uses snake_case (`"imdb_id"`) while `MoviesItem.ImdbId` uses camelCase (`"imdbId"`). Similarly, `KinopoiskId` is camelCase everywhere. This inconsistency is harmless for parsing but makes the types harder to maintain. Consider normalizing to a single convention (prefer camelCase for external API compatibility).

## Script domain improvements

### `BuildScript` uses O(n²) string replacements for template filling

`BuildScript` calls `replaceAll` ~20 times, each scanning the entire content string. For a large template this is O(n·k) where k is the number of placeholders. Replace with `strings.NewReplacer` (O(n)) or a single-pass template engine. The `findUnresolved` post-check should also use the same replacer to avoid a second regex pass.

### `translit` module panics on unknown characters (real-world crash)

`translit/translit.go:186-188` panics when a character is not in the `changeLiteralsMap`. The `panic.txt` file documents a real crash: `Декстер®` fails because `®` is not mapped. The `DEBUG_panicIfLiteralIsUNKNOWN` field is set to `true` by default in production. Either add `®` and other common symbols to `literals.go`, or change the default to `false` so unknown characters pass through unchanged.

### Hardcoded subtitle style in `buildHSUB`

`script.go:184` embeds a ~400-character inline SRT style string (font, size, colors, margins) directly in the `subtitles=` filter. This is impossible to customize per-project and makes the function hard to read. Extract the style to a configurable parameter or a template variable so users can adjust font size, position, or colors without modifying code.

### `atempo=25/(25/1)` in audio filter is confusing and possibly wrong

`audio.go:103` generates `[0:N]aresample=48000,atempo=25/(25/1)` for every audio track. Mathematically this equals `atempo=1.0` (no tempo change), but the expression is misleading. If the intention is "no change", use a simpler approach (omit atempo entirely, or use `atempo=1`). If there's a real tempo adjustment needed, document the rationale in a comment.

### No support for multiple input files in ffmpeg command

`BuildScript` uses only `Filenames[0]` as the ffmpeg input (`-i "${IN_PROGRESS}/${SOURCE}"`). Units can have multiple files (e.g., split parts, audio-only sidecars), but only the primary file is used. Consider supporting multi-input ffmpeg commands (e.g., `-i file1 -i file2`) or at least logging a warning if more than one file exists.

### No unit tests for `deriveNameTag` and `deriveSeasonDir`

`naming.go` has `TestDeriveOutBase` but no direct tests for `deriveNameTag` (the core S/E suffix logic) or `deriveSeasonDir`. Add table-driven tests covering all four branches: season+episode, season-only, episode-only, and neither.

### `StreamSelection.Kind` uses bare strings instead of typed constants

`StreamSelection.Kind` is a `string` with magic values `"v"`, `"a"`, `"s"`. Define typed constants (e.g., `const KindVideo = "v"`) to prevent typos and make the API self-documenting.

### `buildOutputFiles` hardcodes codec parameters

The video output line hardcodes `-c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile:v high -g 0` and audio uses `alac`. These are impossible to change without modifying code. Consider adding a `CodecConfig` or `OutputProfile` struct that can be passed through `ScriptConfig` or `Environment` for future customization.

### Template has no versioning mechanism

`AmediaGeneric` is the only template, hard-coded as a package-level variable. If different rendering approaches are needed (e.g., different container formats, different filter chains), there is no extension point. Consider making the template pluggable via a `Template` interface or a template selector in `ScriptConfig`.

### No validation of environment path separators

`Environment` fields (Buffer, InProgress, etc.) are used directly in bash variable assignments. If paths contain spaces, special characters, or backslashes (e.g., from Windows-mounted drives), the generated script would break. Add validation or quoting for path values.

## Source files should be renamed/moved before script rendering

Scripts assume source files arrive in `BUFFER` with their final names (`12254.mp4`, `12254-rus_Russkii.srt`, etc.). The feeder/ingest step that places files into buffer does not rename or restructure them, so the script's `mv BUFFER→IN_PROGRESS` works only if the source files already carry the correct names. A pre-render hook (or feeder convention) should normalize filenames before placing them in buffer, so scripts are resilient to arbitrary input naming.
