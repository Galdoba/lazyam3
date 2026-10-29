# Naming Solution Options — Cross-Platform Path Resolution

## options:

- option_id: OPT-1
  title: Minimal Patch — Normalize path separators in naming script
  summary: >
    Replace backslash notation in the generated naming script with forward slashes
    and resolve the metadata path via a variable that maps to the platform-native
    location. No Go code changes; only the script-executor path construction is
    adjusted.
  approach: >
    - In the naming script generator, emit `metadata="/home/pemaltynov/IN/metadata/"`
      instead of `\\192.168.31.4\buffer\IN\metadata\`.
    - Replace the marker check `[[ -f "\\192.168.31.4\buffer\IN\metadata/..." ]]`
      with a variable-based forward-slash check.
    - Keep IP/host references out of the emitted script; resolve host at runtime
      via config or environment.
  affected_components:
    - naming script generator (script executor marker logic)
    - file mover logic that assembles output script names
    - hardcoded path strings in the naming script
  assumptions:
    - The metadata directory layout is identical on generator and executor host
      (same relative structure under the buffer root).
    - The host IP (`192.168.31.4`) is not required for path resolution and can be
      dropped or resolved via env/config.
  risks:
    - If generator and executor use different buffer roots, a pure string fix
      will not suffice; a variable-based resolution is required.
    - Backward-compatibility with existing `.processing` markers on Windows hosts
      may be broken if markers were written with backslashes.
  test_strategy:
    - Generate a naming script on a Linux host and verify all path strings use
      forward slashes.
    - Simulate the marker `mv` with forward-slash paths and assert success.
    - Run the live test (`scripts/live_test.sh`) and verify `.processing` → `.done`
      transitions.
  estimated_complexity: low
  reversibility: low
  open_questions:
    - Is the buffer root (`/home/pemaltynov/IN/`) fixed per host or configurable?
    - Do existing `.processing` markers on Windows hosts need to be migrated?

- option_id: OPT-2
  title: Local Refactor — Path normalization layer in Go
  summary: >
    Introduce a path-normalization function that converts all incoming paths to a
    canonical forward-slash form before emitting the naming script or constructing
    metadata markers, then maps to the platform-native separator at execution time.
  approach: >
    - Add a `NormalizePath(path string) string` helper that replaces `\` with `/`
      and collapses redundant separators.
    - Apply normalization to source file names, metadata markers, and output paths
      in the domain layer (script generation, metadata, ingest).
    - Use Go's `filepath` module only for platform-native construction at the
      infrastructure boundary, keeping the domain path strings forward-slash.
  affected_components:
    - domain/script.go (script name construction)
    - domain/metadata.go (`.processing`/`.done` marker paths)
    - domain/ingest.go (source file path handling)
    - infrastructure implementations that emit the naming script
  assumptions:
    - The domain layer can own canonical forward-slash path strings and the
      infrastructure layer converts to native separators on emit.
    - Metadata markers are written and read using the same canonical form.
  risks:
    - Changing canonical path representation may require updating existing tests
      that assert on backslash-form paths.
    - If a Windows host writes `.processing` markers with backslashes, the Go
      normalization must tolerate both forms on read.
  test_strategy:
    - Unit tests for `NormalizePath` covering `\`, `/`, mixed, and redundant
      separators.
    - Integration test: generate script on Linux, run marker logic, assert
      forward-slash paths and successful `.done` transition.
  estimated_complexity: medium
  reversibility: medium
  open_questions:
    - Should normalization happen in the domain layer or at the infrastructure
      boundary?
    - Must the Go code also read legacy backslash-form markers for migration
      periods?

- option_id: OPT-3
  title: Architectural Change — Config-driven paths with platform abstraction
  summary: >
    Move all path definitions (buffer, in-progress, done, archive, notifications,
    metadata) into the TOML config and construct paths via a platform-agnostic path
    builder. The generator and executor both resolve paths from the same config,
    guaranteeing identical format regardless of OS.
  approach: >
    - Extend the TOML config with a `paths` section (buffer_root, metadata_dir,
      in_progress_dir, done_dir, archive_dir, notifications_dir).
    - Introduce a `PathBuilder` service that uses `filepath` for the host OS but
      exposes canonical forward-slash strings for metadata markers.
    - Generator writes metadata markers using the canonical form; executor reads
      using the same canonical form from config.
    - Add path existence validation before any file operation (move, detect).
  affected_components:
    - config/TOML schema (paths section)
    - app module wiring for path builder
    - domain/script.go, domain/metadata.go, domain/ingest.go (consume path builder)
    - infrastructure file mover logic
  assumptions:
    - The TOML config can be shared or generated consistently across generator
      and executor hosts.
    - Buffer root and metadata layout are configurable per host but must be
      resolvable to the same canonical form by both sides.
  risks:
    - Config drift between generator and executor could reintroduce mismatches;
      validation is required.
    - Larger change surface: touches config schema, app wiring, and domain ports.
  test_strategy:
    - Config schema test: validate paths section loads and normalizes correctly.
    - PathBuilder unit tests: native vs canonical output per OS.
    - Live test (`scripts/live_test.sh`) with config-driven paths and explicit
      path-validation failures (missing source dir).
  estimated_complexity: high
  reversibility: high
  open_questions:
    - Will generator and executor share the same config file, or generate
      compatible configs independently?
    - Should path validation fail fast (per requirements) or degrade gracefully?
