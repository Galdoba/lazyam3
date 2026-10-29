---
decision_id: DR-2026-1029-naming
problem_id: P-2026-1029-cross-platform-path-resolution
selected_option_id: OPT-3

evaluation_criteria:
  correctness: OPT-3 fully resolves cross-platform path issues by guaranteeing identical format via shared config; OPT-1 risks buffer-root mismatches; OPT-2 normalizes but lacks config-driven guarantees.
  completeness: OPT-3 satisfies all requirements: forward slashes, removes Windows syntax, platform abstraction, TOML-configurable paths, and path validation.
  maintainability: OPT-3 centralizes path definitions in TOML and uses a PathBuilder service, making the system easier to understand and modify.
  risk: OPT-3 has a larger change surface (config schema, app wiring, domain ports) but is the most robust; config drift must be prevented via validation.
  testability: OPT-3 supports config schema tests, PathBuilder unit tests, and live tests with explicit path-validation failures.
  reversibility: OPT-3 has high reversibility via config rollback and clear domain ports.
  compatibility: OPT-3 extends TOML config schema; generator and executor share/resolvable config ensures compatibility.
  cost: OPT-3 requires high implementation and review effort due to config schema, wiring, and domain port changes.
  security: No new attack surface introduced; path validation guards against path traversal.
  performance: No degradation to critical paths; PathBuilder is lightweight.

decision_rationale: |
  The problem requires cross-platform path resolution between a Windows generator and Linux executor with five explicit requirements: standardized forward-slash paths, removal of Windows-specific syntax and IP references, platform abstraction, TOML-configurable paths, and path validation. OPT-3 (config-driven paths with platform abstraction) is the only option that satisfies all requirements, particularly the config-driven paths and path validation. OPT-1 is too narrow (string-only fix that fails if buffer roots differ) and OPT-2 normalizes paths but lacks config-driven guarantees and validation. OPT-3 centralizes path definitions in TOML via a PathBuilder service, guaranteeing identical format on both sides and enabling path existence validation before operations. The higher cost and change surface are justified by the architectural requirement for configurable, validated paths.

rejected_options:
  - option_id: OPT-1
    reason: Minimal string-only patch that does not address config-driven paths or path validation requirements; risks buffer-root mismatches if generator and executor use different roots.
  - option_id: OPT-2
    reason: Path normalization layer lacks config-driven path definitions and explicit path validation; medium complexity but incomplete against requirements.

required_prototype: false
