---
name: doc-governance
description: Use when creating, updating, or deleting documentation files in doc/. Ensures consistent naming, feature tracking, and INDEX.md synchronization. Load before writing any doc or modifying feature_list.json.
---

# Documentation Governance

<!-- doc-governance -->

Use this skill whenever you write, update, or delete documentation.

1. Read `doc/INDEX.md` first — it lists every valid doc, naming conventions, and the status lifecycle.
2. Read `doc/feature_list.json` before adding or modifying a feature. Feature keys use `<prefix>-<number>` format (e.g., `infra-005`, `script-003`).
3. After any doc change, update INDEX.md — every file in `doc/` must have an entry.
4. After finishing a feature, update its `status` in `feature_list.json` and verify `feature_index.md` stays in sync.

Feature object schema (key fields): `title`, `domain`, `status`, `priority`, `created`, `updated`, `done`, `blocked_by`, `related`, `problem`, `proposal`, `outcome`, `tests`, `notes`. Notes is an array of strings. Tests is an object with `total`, `passed`, `skipped`, and `list`.

Status lifecycle: `proposed` → `in_progress` → `in_review` → `done` (or `deferred`/`cancelled`).

HANDOFF.md: rewrite completely each session. PROGRESS.md: append only.
