---
name: doc-governance
description: Documentation governance conventions for lazyam3 doc/. Load before writing docs or modifying feature_list.json.
---

# Documentation Governance

## Naming Conventions

- Feature keys use `<prefix>-<number>` format (e.g., `A5`, `S1`, `S4a`, `E1`).
- Prefixes: `A` = architecture/config, `S` = script-generation, `D` = domain issues, `E` = pipeline errors.
- Doc files: `domain_<name>.md` for bounded contexts, `feature_list.json` for the machine-readable backlog, `feature_index.md` for the human-readable summary.

## Status Lifecycle

`proposed` → `in_progress` → `in_review` → `done` (or `deferred`/`cancelled`).

## Feature Format

Feature object schema (key fields): `title`, `domain`, `status`, `priority`, `created`, `updated`, `done`, `blocked_by`, `related`, `problem`, `proposal`, `outcome`, `tests`, `notes`. Notes is an array of strings. Tests is an array of test names (strings).

## Doc Creation Rules

- Every file in `doc/` MUST have an entry in `doc/INDEX.md`.
- After any doc change, update INDEX.md.
- After finishing a feature, update its `status` in `feature_list.json` and verify `feature_index.md` stays in sync.
- HANDOFF.md: rewrite completely each session.
- PROGRESS.md: append only.
