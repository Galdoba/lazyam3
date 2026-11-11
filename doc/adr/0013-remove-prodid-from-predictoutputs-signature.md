---
id: "0013-remove-prodid-from-predictoutputs-signature"
title: "Remove prodID Parameter from PredictOutputs Signature"
description: "After removing _idN from filenames, the prodID argument to PredictOutputs became dead. The parameter was removed from the signature and all 30+ callers updated to enforce the no filename-extracted id constraint at the type level."
status: accepted
date: "2026-11-11"
scope: cross-package
---

# Remove prodID Parameter from PredictOutputs Signature

## Context

`PredictOutputs` in `internal/domain/callname/names_output.go` previously accepted a `prodID int64` argument that was used only to append `_idN` to output base names. With the `_idN` removal documented in ADR 0012, the parameter no longer served any purpose in the function body. Leaving it in the signature would create a dead parameter passed by 30+ callers across `script.go`, `config.go`, `build_test.go`, `script_test.go`, and `names_output_test.go`.

## Decision

Remove the `prodID int64` parameter from the `PredictOutputs` signature and update all callers to drop the trailing argument. The signature becomes:

`PredictOutputs(product, unit, targetRoot, hardsubTag string) []PredictedOutput`

## Rationale

This enforces the "no filename-extracted id" constraint at the type level. Go permits unused function parameters, so leaving `prodID` would compile, but callers passing `prod.ID` or literals would become dead arguments, causing confusion and violating Go idioms. Removing it at the signature level prevents future callers from reintroducing the id into output names.

## Consequences

- All 30+ call sites (script.go, config.go, build_test.go, script_test.go, names_output_test.go) must drop the trailing argument.
- The `PredictOutputs` signature is now cleaner and reflects that output naming depends only on product metadata, unit streams, target root, and hardsub tag.

## Options

- Leave `prodID` in the signature: compiles, non-idiomatic, dead parameter.
- Remove `prodID` from the signature and update all callers (chosen).
