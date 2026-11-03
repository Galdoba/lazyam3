# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- INGEST-002: bypass non-media files during ingest (logs WARN and skips files ffprobe cannot probe; preserves subtitle sidecar handling and metadata.json skip). Test: `TestIngest_NonMediaFilesSkipped`.

### Changed

- Refactored ingest domain: split `NewMediaUnit` into `collectMediaFiles`/`buildStreams`/`resolveID` helpers; extracted `ExtractID`, `ValidateMediaUnit` helpers, and stream field helpers. Ingest coverage: 97.9%.

### Fixed

- Movie naming: removed `_se` suffix for movie units.
- Marker metadata path bug: corrected metadata path resolution.

### Other

- Task 1-10 complete: pipeline fully wired end-to-end with HSUB auto-detection, TOML config, config-driven paths, empty-dir sanitation, and cross-platform path resolution (PathBuilder, canonical metadata markers).

## [1.0.0] - 2026-02-11

### Added

- Initial release
