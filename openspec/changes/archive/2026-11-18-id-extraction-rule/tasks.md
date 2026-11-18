# Tasks

## 1. One extractor: unanimous claim or nothing

- [x] 1.1 In `internal/domain/ingest/extract.go`, replace the exported `ExtractID(filename string) (int64, error)` with a private `extractID(candidateNames ...string) int64` that reads each name with the accepted grammar (the digit run ends at `-`, `.`, or the end of the stem, and `0` is not a claim) and returns that claim only when every name given to it carries the same one, returning `0` when any name claims nothing or the claims differ, never an error, and update `TestExtractID` in `internal/domain/ingest/unit_test.go:7-45` in the same step so `go test ./internal/domain/ingest/` passes with `63604_PRT251205.mp4` and `63604_1080.mp4` expecting `0`
- [x] 1.2 Extend that table with the accepted and rejected real shapes and verify each expectation: `63597-rus_Russkii_Forsirovannye.srt` → 63597, `12254-rus_Russkii.srt` → 12254, `65003-eng.srt` → 65003, `63603` → 63603, `1.mp4` → 1, `12254rus.srt` → 0, `0.mp4` → 0, `1080p_Show.mp4` → 0, `2017_Show.mp4` → 0, `no_id_in_video.mp4` → 0, `something-unrelated.md` → 0, `""` → 0
- [x] 1.3 Add the unanimity cases to the same table and verify each: `extractID()` → 0, `("12345.mp4", "12346.mp4")` → 0, `("63603.mp4", "no_id_in_video.mp4")` → 0, `("63603.mp4", "63603.mkv")` → 63603, `("63603.mp4", "12254-rus.srt")` → 0
- [x] 1.4 Document the grammar and the unanimity rule in the `extract.go` header comment (accepted form, digit-run terminator, `0` is not a claim, the candidate names are filtered before the call) and verify `golangci-lint run ./internal/domain/ingest/` is clean

## 2. Naming stops deriving product IDs

- [x] 2.1 Delete `ExtractID` from `internal/domain/callname/names_input.go:119-152` and its `TestExtractID` table at `internal/domain/callname/names_input_test.go:344-375`, and verify `go build ./...` succeeds
- [x] 2.2 Delete the inline digit scan at `internal/domain/callname/names_input.go:93-106` and the `ScriptInput.ID` field at `names_input.go:26`, so `BuildInputName` produces no product ID at all, then verify `TestBuildInputName` at `names_input_test.go:217-304` still yields the same call name string for `12345.mp4` and for `no_digits.mp4`
- [x] 2.3 Remove the `si.ID == 0` rejection from `validate()` at `names_input.go:74-76` and verify `TestBuildInputName` builds call names successfully for a product whose ID is `0`, so `internal/app/pipeline/config.go:66-69` no longer fails unidentified units
- [x] 2.4 Drop the assertions on a scanned ID in `internal/domain/callname/names_input_test.go` (`:242`, `:254`, `:274`) and verify `go test ./internal/domain/callname/` passes

## 3. Identity resolution and the removed sentinel

- [x] 3.1 Remove `SyntheticID` from `internal/domain/ingest/unit.go:8-10` and make `MediaUnit.Synthetic` mean `ID == 0` (`unit.go:21`, `unit.go:52`), and verify `grep -rn "SyntheticID" --include=*.go .` returns nothing
- [x] 3.2 Collapse `resolveID` (`internal/domain/ingest/unit.go:92-111`) into the single `extractID(candidateNames...)` call in `NewMediaUnit`, so the unit ID is the unanimous claim or `0` and no identity path returns an error, keeping the candidate names as `collectMediaFiles` produces them today; verify `go test ./internal/domain/ingest/` passes
- [x] 3.3 Flip the `NewMediaUnit` table in `internal/domain/ingest/unit_test.go` (`wantID: SyntheticID` at `:348`, `:359`, the synthetic assertions at `:381-384`, and the conflicting case at `:327`) to the `0` expectation and verify every one of them now expects an unidentified unit that builds without an error
- [x] 3.4 Add unit cases and verify each outcome: `63603.mp4` + `no_id_in_video.mp4` resolves to `0` and is unidentified; `63603.mp4` + `99999.mp4` resolves to `0` and is unidentified rather than an error; `63603.mp4` + `63603.mkv` resolves to `63603`; `MyShow_E01.mp4` + `MyShow_E01-rus.srt` resolves to `0`

## 4. Pipeline reporting of an unidentified unit

- [x] 4.1 Update the synthetic warn at `internal/app/pipeline/pipeline.go:325-327` to state that the file names did not agree on one product identity, and verify a run over a directory whose candidates carry no agreed claim logs that line
- [x] 4.2 Update `TestRunOnce_SyntheticID` at `internal/app/pipeline/pipeline_test.go:514-551` to expect `UnitDone(0)` and verify `go test ./internal/app/pipeline/` passes
- [x] 4.3 Verify the fallback path end to end in a test: an unidentified unit produces a fallback product whose ID is `0` (`pipeline.go:485-491`) and a processing record whose `unit_id` is `0` (`pipeline.go:387`)

## 5. Documentation of the rule

- [x] 5.1 Update `doc/domain_ingest.md`: replace the **SyntheticID** vocabulary row at `:47` with the unidentified identity (`0`), restate Scenario 7 around unanimity (`:92`), and rewrite the acceptance criterion at `:128` for the accepted grammar and the unanimous rule; verify no `9999` identity wording remains in the file
- [x] 5.2 Update `doc/domain_callname.md:31`, `:126`, `:157` to drop `ExtractID` and state that the ID comes from the resolved product; verify the file no longer describes name-based ID parsing
- [x] 5.3 Update `doc/architecture.md:89` to the private extractor signature owned by ingest; verify the documented signature matches `extract.go`
- [x] 5.4 Write `doc/adr/0019-input-name-grammar-owned-by-ingest.md` covering the grammar, `0` as unidentified, unanimity-or-nothing, and callname no longer deriving IDs, and add its row to `doc/adr/INDEX.md`; verify the index lists 0019 next to 0012 and 0013

## 6. Integration checks

- [x] 6.1 Run `go build ./...`, `go vet ./...`, `go test -race -cover ./...`, `golangci-lint run ./...` and verify all exit clean
- [x] 6.2 Run `./scripts/live_test.sh` and verify it reports a live test pass, with `bad-name` and `bad-name-2` reaching fallback products and every identified fixture keeping its catalog ID — no fixture has candidates that disagree, so every fixture is expected to land where it lands today
- [x] 6.3 Run `./scripts/full_test.sh` and verify it passes, or record the failure against this change before archiving

`scripts/full_test.sh` did not exist when this task was reached, so it was added: it runs build, vet, `go test -race -cover ./...`, `golangci-lint run ./...`, then `live_test.sh` and `live_test_2.sh`, and reports PASS or FAIL per step with a log kept per step. Its first run passed every step.

## Workflow follow-up

- Run `openspec verify --change id-extraction-rule` against the delta specs before archive.
- Archive the change so `product-identity` lands in `openspec/specs/`.
