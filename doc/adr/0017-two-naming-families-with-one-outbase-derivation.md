---
id: "0017-two-naming-families-with-one-outbase-derivation"
title: "Two Naming Families With One OUTBASE Derivation"
description: "Artifacts lazyam3 reads back are named after the script; artifacts lazyam3 places for the operator are named after the render output base name. The base name is derived in exactly one helper shared by the ready marker and the copied error log."
status: accepted
date: "2026-11-17"
scope: cross-package
---

# Two Naming Families With One OUTBASE Derivation

## Context

The change introduced a second operator-facing file in the render output directory (the copied error log) alongside the existing ready marker, and a pipeline-read sidecar in the metadata directory. Three names for one render raised the question of which name belongs to which artifact, and the ready-marker name was previously derived inline inside the ready-file builder, so a second copy of that logic would have had to be kept in sync by hand.

## Decision

Two naming families, made explicit:

- **Script-name family**, for anything lazyam3 reads back: `<SCRIPT_NAME>.sh`, `<SCRIPT_NAME>.processing`, `<SCRIPT_NAME>.done`, and the error sidecar `<SCRIPT_NAME>.err` in the metadata directory.
- **Render-outbase family**, for anything lazyam3 places for the operator in the target directory: `<OUTBASE>.ready` and `<OUTBASE>.#err`.

The outbase is derived once by `renderOutBase(predictions)`, which strips the extension from the first video prediction. The generated script exports it as the `OUTBASE` shell variable, and both operator-facing names in the post-render block are built from that variable, so their parity is structural rather than something tests have to police.

The one documented exception: the announcement name of an error log the operator's notifier collected itself keeps whatever name the handler gave it. lazyam3 does not rename it.

## Rationale

Correlation inside lazyam3 runs through the PRT-bearing script name: it is the store key, the `.done` payload field, and the sidecar key. Correlation for the operator runs through the deliverable: files sitting in the render output directory have to match the media they describe. Forcing one name on both roles would either make the handler learn lazyam3's internal naming or break the link between a record and its script when the same product is rendered more than once.

## Consequences

- One render has two distinct artifact names in two directories; joining a `<OUTBASE>.#err` in the target directory to its error store entry goes through `script_name`.
- The ready-marker touch now lives inside the post-render copy block instead of a standalone placeholder in the template, so the standalone ready-file builder was deleted rather than left in place unused. Its extension-stripping coverage moved to the derivation helper.
- The error store entry takes part in neither derivation: it is keyed by the script name.

## Options

- **One uniform name everywhere.** Rejected: keying operator-facing files on the PRT-bearing script name forces the notification handler to learn lazyam3's internal naming; keying markers on the outbase breaks the record-to-script correlation when the same product is rendered more than once.
- **Derive the outbase independently in each builder.** Rejected: the two operator-facing names would drift as soon as one derivation changed, and only a test would notice.
