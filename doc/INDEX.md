# Documentation Index

All docs live in `doc/`. Every file in `doc/` MUST have an entry here.

| File | Type | Purpose |
|---|---|---|
| `INDEX.md` | Master index | This file — catalog of all docs |
| `SKILL.md` | Skill | Documentation governance conventions |
| `architecture.md` | Architecture | Layer diagrams, dependency rules, dev guidelines |
| `lazyam3.md` | Overview | Workflow, domain modules, repository tree |
| `domain_ingest.md` | Bounded Context | Ingest domain specification |
| `domain_metadata.md` | Bounded Context | Metadata domain specification |
| `domain_script.md` | Bounded Context | Script domain specification |
| `domain_callname.md` | Bounded Context | CallName domain specification |
| `feature_list.json` | Backlog | Machine-readable feature tracking |
| `feature_index.md` | Summary | Human-readable feature table (synced from JSON) |
| `HANDOFF.md` | Handoff | Session knowledge transfer |
| `PROGRESS.md` | Progress | Session progress log |
| `BACKLOG.md` | Backlog | Known issues and footguns |
| `NOTES.md` | Notes | Unexpected discoveries |
| `sanitation_problem_statement.md` | Problem Statement | Sanitation problem — artifact-based solution proposal for deleting empty incoming dirs |
| `sanitation_problem.md` | Verification Results | Sanitation problem — verification results and implementation plan for artifact-based solution |
| `naming_problem_statement.md` | Problem Statement | Naming problem — cross-platform path resolution between Windows generator and Linux executor |
| `naming_context.md` | Context | Naming problem — generator/executor path examples |
| `naming_decidion_record.md` | Decision Record | Naming problem — selected OPT-3 config-driven paths with platform abstraction |
| `naming_soluution_options.md` | Solution Options | Naming problem — OPT-1/2/3 options analysis |
| `naming_implementation_plan.md` | Implementation Plan | Naming problem — executable plan for OPT-3 cross-platform path resolution |
| `text_logger_format_refactoring.md` | Spec | Logger format template specification |
| `pipeline_chart.html` | Visualization | Mermaid graph visualizing pipeline operations, data flow, and module coverage |

## Feature Backlog

See `feature_list.json` for the machine-readable backlog. See `feature_index.md` for the human-readable summary.

## Conventions

Naming, status lifecycle, feature format, and doc creation rules are in `SKILL.md`.
