# Design

## Context

See `proposal.md` — Why, for motivation. The shape that constrains the approach:

```
internal/domain/ingest                     Reached by production code?
+---------------------+--------------------------------------------------+
| port.go   Organizer | yes   internal/app/pipeline/{dto,pipeline}.go   |
| unit.go   MediaUnit | yes   ID, Filenames, Streams, PRTTimestamp,     |
|           Synthetic |       Synthetic  (pipeline.go, config.go)        |
|           Validation| NO    never written, never read                  |
|           Validate* | NO    no caller outside the package              |
| extract.go extractID| yes   called by NewMediaUnit                     |
| order.go sortUnit…  | yes   called by collectMediaFiles                |
| parse.go parseSt…   | yes   called by NewMediaUnit                     |
| stream.go  Stream   | yes   Kind, FileIndex, Index, Language,         |
|                     |       ChannelLayout, Interlaced are read by     |
|                     |       pipeline, callname, script                |
+---------------------+--------------------------------------------------+
```

Constraints:

- The project gate cannot see this dead code. `unused` does not report exported symbols in a non-`main` package, so `golangci-lint run ./...` reports 0 issues while 138 lines of unreachable logic sit at 100% coverage.
- The previous change made this explicit rather than accidental: wiring `MediaUnit.Validation` into the pipeline was listed as out of scope in `openspec/changes/archive/2026-11-18-id-extraction-rule/proposal.md:37`, and nothing since picked it up.
- Project rules bound the work to one package: no consumer of `MediaUnit` is edited, domains stay pure, and minimal edits only.

## Goals / Non-Goals

**Goals:**

- Every remaining declaration in `internal/domain/ingest` is reachable from production code or is a deliberate, documented model fact.
- Tests left in the package cover behavior the pipeline depends on, including the defensive parsing branches.
- The decision to keep unread descriptive fields is written down where a future reader will find it.

**Non-Goals:**

- Giving ingest a validation capability, or any opinion about media quality.
- Changing the shape of `Stream` or `MediaUnit`, or which ffprobe facts get parsed.
- Deduplicating `abbreviateKind` (`stream.go:74`) against `abbrevKind` (`internal/domain/script/script.go:379`) — that crosses the package boundary.
- Changing ffprobe probing in `internal/infra/fso` or anything in `internal/app/ingest`.

## Decisions

**1. Delete the validation subsystem rather than wire it or leave it dormant.**

Dormant logic with a passing suite is worse than absent logic: a reader sees `Validation` on `MediaUnit` and concludes the pipeline checks media quality, which it does not. Wiring it is a feature — it needs a decision about where a suspect unit goes (pipeline warn, the error store under ADR 0016, or a rejection under ADR 0004), and that decision is not part of a cleanup. Alternative considered and rejected: keep it "for later", with a comment; rejected because the comment is exactly the kind of code-comment noise the project rules avoid, and it preserves 138 lines nobody can reach.

**2. Keep the descriptive `Stream` fields, and record why.**

Two coherent endings exist: keep `Codec`, `Width`, `Height`, `FrameRate`, `PixelFormat`, `SampleRate`, `SampleFormat`, `Forced`, `Duration`, `BitRate` as the domain's vocabulary for a media product, or trim `Stream` to the six fields consumers actually read and let `Rational`, `parseDuration`, `parseRational`, `parseInt`, `parseBitRate` and `Disp` go with them. Vocabulary was chosen: this is a media tool, these are the facts a media product has, and each has an obvious future consumer (validation, selection, reporting). Trimming would also narrow the model at the exact boundary where `callname` and `script` already reach into it. The price is real — a field nobody reads is indistinguishable from dead code unless someone wrote it down — so the decision is recorded in `doc/adr/` as part of this change, which is the third option that makes the choice honest rather than accidental.

**3. Delete only the tests of deleted code.**

`TestParseDuration`, `TestParseRational`, `TestParseBitRate`, `TestStreamKind` and `TestDetectInterlaced` stay: their subjects stay in production, and they are the only coverage of the `N/A`, `0/0`, empty-string and malformed branches. Alternative considered: fold them into `TestParseStreams`; rejected as churn that would weaken exactly the branches they exist for.

**4. `buildStreams` returns the coordinate map only.**

Its second result is `len(filenames)`, which any call site can compute, and the only call site discards it into `_`. Alternative considered: keep it as a placeholder for a future file count; rejected — an always-discarded result invites a reader to assume the pair carries meaning.

**5. Fix the `"tag"` fixtures instead of dropping the assertion.**

`unit_test.go:85` and `:265` write `"tag"` where ffprobe emits `"tags"`, so `field_order` never reaches `detectInterlaced` and the interlaced assertion at `unit_test.go:130` cannot fail. Alternative considered: delete the assertion; rejected, because the rest of the fixture is faithful ffprobe output and the assertion is worth having as a real one.

**6. Record the outcome as an ADR, not a spec requirement.**

Specs in this project describe behavior the system performs. "The model carries facts no consumer is required to read" and "ingest does not validate media quality" are decisions with no observable behavior attached, which is what `doc/adr/` exists for. Alternative considered: a new `media-unit-model` capability spec; rejected as inventing a requirement to satisfy tooling.

## Risks / Trade-offs

- [Someone later wants ingest media-quality validation] → the ADR names the deleted capability and the return path: its own change that wires unit issues through the pipeline's error handling, not restored dormant code.
- [Unread fields creep back in unnoticed] → nothing in the gate catches this (`unused` cannot see exported symbols here); it is caught by review at verify/archive time, and the ADR is the thing a reviewer checks against.
- [Coverage percentage looks different after the change] → code and its tests are removed together, so coverage of what remains is unchanged; the number should not be treated as a signal that anything regressed.
- [A deletion reaches something the pipeline does use] → each symbol is checked with gopls symbol references before removal, and the gate for this change is `./scripts/full_test.sh`, whose two live runs exercise ingest end to end.

## Migration Plan

No stored data, configuration, or output shape changes: nothing outside the package referenced the removed symbols, so there is nothing to migrate. Rollback is a revert of a single commit.
