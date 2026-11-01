---
problem_id: INGEST-001
title: Graceful handling of files with non-standard names
repository: github.com/Galdoba/lazyam3
baseline_commit: 18d36f3a05a2aca709ac037d587017b0d427909a
problem_type: improvement
status: APPROVED
---

## current_behavior: |
  The pipeline fails to process any unit whose files do not follow the naming convention of "leading digits + stem + extension". Two failure points exist:

  1. **Ingest phase** (`internal/domain/ingest/unit.go:51-54`): `NewMediaUnit()` calls `ExtractID()` on every filename. If the filename stem does not start with a digit, `ExtractID()` returns an error and the entire unit is rejected.

  2. **Callname phase** (`internal/domain/callname/names_input.go:101-102`): `BuildInputName()` also calls leading-digit extraction on the stem. If no digits found, it returns `filename "..." does not start with a digit`.

  3. **Pipeline level** (`internal/app/pipeline/pipeline.go:204-206`): Any error from `Ingest()` causes `EventUnitFailed` to be emitted and the unit is skipped entirely.

  Files with names like `MyShow_E01.mkv`, `s01e01.mkv`, `random_name.mp4`, or any name not starting with a numeric product ID are silently skipped by the pipeline. The user receives no output files for these units.

## desired_behavior: |
  The pipeline should process every directory that contains valid media files, even when filenames do not follow the expected "leading digits" convention. When a filename cannot be parsed for a product ID, the pipeline must:

  1. Assign a fixed synthetic product ID (9999) so the unit can proceed through the full pipeline.
  2. Generate call names and rendering scripts using whatever metadata is available (fallback metadata already exists when Product lookup fails).
  3. Log a warning indicating that a synthetic ID was used, so the user is aware.
  4. Emit `EventUnitDone` (not `EventUnitFailed`) so the unit is counted as processed.

  The pipeline must never skip a directory solely because its files have non-standard names, provided the files contain decodable media streams.

## rationale: |
  In real-world usage, source files arrive with arbitrary naming conventions from various providers (broadcast captures, web downloads, P2P releases). The pipeline currently discards these entirely, which means a significant portion of incoming media is silently ignored. Users expect all files in an `incoming/` directory to be processed -- the absence of output for some files is confusing and leads to manual rework.

  The metadata fallback mechanism (already present in `pipeline.go:212-216`) shows the codebase already anticipates imperfect inputs. Extending this resilience to the naming/ID extraction layer is a natural extension of the same philosophy.

## evidence:
  - type: code_review
    reference: internal/domain/ingest/unit.go:51-54
  - type: code_review
    reference: internal/domain/callname/names_input.go:101-102
  - type: code_review
    reference: internal/app/pipeline/pipeline.go:204-206
  - type: architecture
    reference: doc/domain_ingest.md (ingest expects digit-prefixed IDs)
  - type: architecture
    reference: doc/domain_callname.md (callname expects digit-prefixed IDs)

## acceptance_criteria:
  - criterion: A directory containing media files with names that do not start with digits (e.g., "MyShow_E01.mkv") is processed end-to-end and produces output files
    verification_method: integration test in pipeline_test.go that creates a temp directory with non-digit-named media files and asserts EventUnitDone + non-zero Processed count
  - criterion: The synthetic ID assignment is logged as a warning in the unit's processing log
    verification_method: capture log output and verify "synthetic ID" or equivalent warning string is present
  - criterion: Existing units with valid digit-prefixed names continue to work unchanged
    verification_method: existing tests in pipeline_test.go, ingest_test.go, and names_input_test.go all pass
  - criterion: Units with invalid names within the same directory are handled consistently (all files in a unit share the same synthetic ID)
    verification_method: test with a directory containing non-digit-named files, verify single unit is produced with consistent ID (example: /home/galdoba/go/src/github.com/Galdoba/lazyam3/testdata/synthetic/bad-name/)


## non_goals:
  - Changing the metadata resolution system (Product lookup via series/episode)
  - Adding interactive prompts or user input for ambiguous cases
  - Supporting completely arbitrary binary blobs (no decodable media streams)
  - Changing the output naming format itself (only the ID derivation changes)

## constraints:
  - Domain purity: ingest and callname domains must remain pure (no external deps)
  - Backward compatibility: existing behavior for digit-prefixed names must not change
  - Synthetic ID is a fixed value (9999), not derived per-directory. All synthetic units share the same ID.
  - The synthetic ID must still be an int64 compatible with the existing MediaUnit.ID field

## risks:
  - Synthetic IDs use a fixed value (9999), so no collision risk and no hashing logic needed.
  - Callname's `BuildInputNames` fails fast on any error (`names_input.go:160-162`), so synthetic ID must be set before callname is invoked.
  - Existing tests assume IDs are always extracted from filenames; some tests may need updating.

## open_questions:
  - When some files in a unit have valid IDs and others don't, should the unit use the valid ID or a synthetic one? (Recommendation: use the valid ID -- the unit's files are already linked by directory.)
