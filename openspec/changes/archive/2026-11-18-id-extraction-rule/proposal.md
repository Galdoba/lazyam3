# Proposal

## Why

The product ID is parsed out of incoming filenames in three places with two different stem rules and two different definitions of a valid name, and a unit whose name carries no ID is disguised as `9999`. "We could not tell what this product is" is therefore indistinguishable from a real catalog key, and names that do not follow the accepted shape are silently accepted, given a claimed ID, and renamed into artifacts instead of degrading to fallback.

## What Changes

- State one input-name grammar for incoming files: `<digits> [ '-' tag [ '_' tag ]* ] [ '.' ext ]`. The digit run must end at `-`, `.` or the end of the stem. `63597-rus_Russkii_Forsirovannye.srt` is the most complex valid form.
- ID extraction becomes a claim decision, never an error: a file either claims a product ID or claims nothing. A claim is a positive number; `0` is not claimable.
- Extraction takes the names that already took part in the unit: `extractID(candidateNames ...string) int64`. It decides who votes nowhere — the candidate names are filtered before the call, so the filter can change later without touching the rule.
- **BREAKING** A unit with no claim is unidentified and carries `ID = 0`. `ingest.SyntheticID` (`9999`) is removed, so no downstream component can mistake a fabricated number for a catalog key. `MediaUnit.Synthetic` becomes `ID == 0`.
- **BREAKING** Identity requires unanimity over the candidate names. If they do not all claim the same ID, the unit has no identity: `12345.mp4` plus `12346.mp4` gives up both claims, and `63603.mp4` plus `no_id_in_video.mp4` is unidentified rather than `63603`.
- **BREAKING** The "inconsistent IDs" error is removed. A unit whose names do not agree is no longer rejected: it continues as unidentified and reaches fallback, where it is reported.
- **BREAKING** `callname` stops deriving product IDs: `callname.ExtractID`, the duplicate digit scan inside `BuildInputName`, and the `ScriptInput.ID` field they existed to fill are removed, and `ScriptInput.validate()` no longer rejects a zero ID. Without that last removal, every unidentified unit would fail name building. The field is deleted rather than refilled, because the ID has been absent from output file names since ADR 0012 and nothing reads it.
- Malformed names reach fallback: `0_show.mp4`, `1080p_Show.mp4`, `2017_Show.mp4`, `id_is_03_video.mp4`, `no_id_in_video.mp4`, `something-unrelated.md`. Because identity is unanimous, a malformed name among the candidates drags the whole directory there.
- Test expectations that flip by design: `63604_PRT251205.mp4` and `63604_1080.mp4` no longer claim `63604`, the inconsistent-IDs case becomes an unidentified unit rather than an error, and the fallback pipeline test expects `UnitDone(0)` instead of `UnitDone(9999)`.

## Capabilities

### New Capabilities

- `product-identity`: how an incoming directory is mapped to a catalog product from its file names — the accepted name shape, what counts as a claim, which names take part in the decision, the unanimity rule, and what an unidentified unit means for everything downstream.

### Modified Capabilities

- None. `execution-signals` and `script-generation` cover rendering artifacts and script output, not identity. The `callname` removals are captured as a `product-identity` requirement that no component other than ingest derives a product ID from a file name.

## Impact

- `internal/domain/ingest`: `extract.go` (one private variadic extractor, no error return), `unit.go` (`SyntheticID` constant, `resolveID` collapsing into a single extractor call, `Synthetic` semantics).
- `internal/domain/callname`: `names_input.go` (`ExtractID`, the inline scan in `BuildInputName`, the `ID == 0` gate, `ScriptInput.ID`).
- `internal/app/pipeline`: fallback and synthetic warn text (`pipeline.go`), name-building error text (`config.go`).
- Tests: `internal/domain/ingest/unit_test.go`, `internal/domain/callname/names_input_test.go`, `internal/app/pipeline/pipeline_test.go`.
- Docs: `doc/domain_ingest.md` (vocabulary, Scenario 7, acceptance criterion), `doc/domain_callname.md`, `doc/architecture.md`, a new ADR, and `doc/adr/INDEX.md`.
- Verification: `go test -race ./...`, `golangci-lint run ./...`, `./scripts/live_test.sh` against the `bad-name` and `bad-name-2` fixtures.
- Out of scope: moving ID extraction out of ingest, changing which names are candidates for the decision (today's filter is kept as it is), re-picking the synthetic sentinel, parsing tags once for all consumers, and wiring `MediaUnit.Validation` into the pipeline.
