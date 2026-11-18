---
id: "0020-ingest-carries-stream-facts-it-does-not-validate"
title: "Ingest Carries Stream Facts It Does Not Validate"
description: "The ingest package does not validate media quality. The descriptive Stream fields are model vocabulary rather than dead code, and any future validation returns as its own change wired through the pipeline's error handling."
status: accepted
date: "2026-11-18"
scope: domain
---

# Ingest Carries Stream Facts It Does Not Validate

## Context

The `internal/domain/ingest` package parses ffprobe output into a `Stream` model carrying `Codec`, `Width`, `Height`, `FrameRate`, `PixelFormat`, `SampleRate`, `SampleFormat`, `Forced`, `Duration`, and `BitRate`. A media-quality validation subsystem sat alongside it — `ValidateMediaUnit`, `validateVideo`, `validateAudio`, `validateDurations`, `rationalMeanFloat`, and the `ValidationIssue` / `ValidationIssues` types — but it was never written into `MediaUnit` and never read by anything outside the package. The subsystem was dormant, not dead by accident: wiring it into the pipeline was listed as out of scope in the identity-extraction change, and nothing since picked it up.

## Decision

**Ingest does not validate media quality.** The validation subsystem was removed entirely, along with the `MediaUnit.Validation` field it would have populated, the `TagMap` type with zero references, and the orphan comment left behind by the identity-extraction refactor. `buildStreams` now returns the coordinate map only.

**The descriptive `Stream` fields are model vocabulary, not dead code.** They are the facts a media product has, and each has an obvious future consumer: validation, selection, or reporting. A field that nothing reads is indistinguishable from dead code unless someone wrote it down, so this decision is recorded here. Consumers that already reach into the model (`callname`, `script`) keep reading the fields they use; trimming the model at that boundary was rejected.

**Validation, if it ever returns, returns as its own change.** Any media-quality validation must wire unit issues through the pipeline's error handling (see ADR 0004 and ADR 0016), not resurrect dormant logic.

## Rationale

Keeping the `Stream` fields preserves the domain's vocabulary for a media product and avoids narrowing the model at the boundary where `callname` and `script` already reach into it. The alternative — trimming `Stream` to the six fields consumers read — would have forced those consumers to reconstruct facts they do not need and pushed parsing logic out of ingest. The price is real: an unread field looks like dead code. The ADR makes the choice honest rather than accidental.

## Consequences

- Coverage of the ingest package reflects only code the pipeline uses; the removed subsystem and its three test suites are gone together, so coverage of what remains is unchanged.
- A future reader sees `Stream` fields with no current consumer and checks this ADR before treating them as candidates for removal.
- If media-quality validation is wanted later, it is a feature with a decision about where a suspect unit goes, not a cleanup.

## Options

- **Trim `Stream` to the fields consumers read.** Rejected: it narrows the model at the boundary where `callname` and `script` already reach in, and it would strip facts a media product has no obvious reason to lose.
- **Keep the validation subsystem dormant with a comment.** Rejected: a dormant subsystem with a comment is worse than absent logic, because a reader concludes the pipeline checks media quality when it does not.
- **Record the decision as a spec requirement.** Rejected: specs describe behavior the system performs; "the model carries facts no consumer is required to read" and "ingest does not validate media quality" are decisions with no observable behavior attached.
