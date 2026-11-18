---
id: "0021-metadata-carries-product-facts-it-does-not-classify"
title: "Metadata Carries Product Facts It Does Not Classify"
description: "The metadata domain publishes a flat product record and a Kind enum but performs no classification. The six Kind predicates were removed rather than left dormant, and the Product fields infra assigns and nothing reads are model vocabulary rather than dead code."
status: accepted
date: "2026-11-18"
scope: domain
---

# Metadata Carries Product Facts It Does Not Classify

## Context

`internal/domain/metadata` is three files: a flat `Product` record of thirteen fields, a `Kind` enum with five values, and the `ProductRepository` port. Alongside the enum sat six predicate methods — `IsMovie`, `IsSeries`, `IsSeason`, `IsEpisode`, `IsMovieEpisode`, `IsSeriesEpisode` — that no production code called. Each had exactly three gopls references: its own declaration and two lines of the package's only test file, which existed solely to assert against them. The package reported 100% coverage because of that file, which hid the fact that the pipeline never consulted a predicate.

Classification the pipeline really performs is structural. `callname` builds the season/episode tag from whether `SeasonNumber` and `EpisodeNumber` are nil (`s01e06`, `s01`, `e06`, or `""`), and its own comment states that products without those numbers must not get `s`/`e` fragments. The pipeline's fallback for a unit with no resolved product marks it as `KindMovie` directly. Meanwhile `Kind` travels from infrastructure into the record and stops there: `FoundToProduct` switches on `Found.Kind` to decide which shape to build, and nothing in `internal/app` or `internal/domain` reads `Product.Kind`.

## Decision

**The metadata domain classifies nothing.** The six predicates were deleted together with `types_test.go`, the file that tested only them. Nothing outside the package was edited, because nothing outside the package referenced them.

**The unread `Product` fields are model vocabulary, not dead code.** `Kind`, `ParentID`, `SeasonEp`, `Description`, `Genres`, `Countries`, `KinopoiskID`, `IMDBID` are assigned by `internal/infra/metadata` and read by no consumer; `ID`, `Title`, `SeriesTitle`, `SeasonNumber`, `EpisodeNumber` are read by the pipeline, `callname`, and `script`. All thirteen stay, as do the `json` tags, even though nothing marshals a `Product` today. A field that nothing reads is indistinguishable from dead code unless someone writes it down, so this decision is recorded here.

**Kind-based branching returns as its own change.** If a consumer ever needs to ask a product what kind it is, that arrives as a feature with the consumer named and its behavior specified, not as resurrected helpers.

## Rationale

The predicates were dormant behavior rather than carried data, which is the opposite half of what ADR 0020 protected. Keeping them would have left two ways to answer the same question — the kind enum and the season/episode numbers — and the numbers are the one `callname` actually trusts. Dormant helpers invite a future reader to branch on `Kind` where the existing derivation already decides, and a reader skimming the package would conclude the pipeline separates movies from series by kind, which it does not.

The fields are the other half, and the reasoning is the same as ADR 0020's: they are the facts a catalog product has, trimming them would narrow the record at the boundary infrastructure already fills, and unlike the predicates it could not be done inside the domain package alone.

## Consequences

- The package keeps 35 lines across `port.go` and `types.go`, all of it declarations. Its six predicate bodies were its only executable statements, so `go test -cover` now reports a package with nothing to cover. Coverage no longer overstates what the pipeline exercises.
- A future reader sees eight fields with no reader-side code and checks this record before treating them as removal candidates.
- A future reader sees `Kind` on the record with no consumer and should not assume it is unused infrastructure plumbing: it is how `FoundToProduct` chooses the record's shape.
- If classification is wanted, it is a change with a named consumer and specified behavior, not a cleanup.

## Options

- **Keep the predicates as ready-made classification vocabulary.** Rejected: dormant behavior is worse than absent behavior, because the package advertised a capability the pipeline does not use, and it competed with the number-based derivation `callname` already relies on.
- **Trim `Product` to the five fields consumers read.** Rejected: it narrows the record at the boundary infrastructure fills, forces edits in `internal/infra/metadata` and its tests for no behavior change, and strips facts a catalog product has no reason to lose.
- **Record the outcome as a spec requirement.** Rejected: specs describe behavior the system performs; "the record carries facts no consumer is required to read" and "the domain classifies nothing" carry no observable behavior.
