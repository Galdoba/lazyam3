# lazyam3

[![Go Version](https://img.shields.io/github/go-mod/go-version/Galdoba/lazyam3)](https://go.dev/) [![License](https://img.shields.io/github/license/Galdoba/lazyam3)](./LICENSE) [![Build Status](https://img.shields.io/github/actions/workflow/status/Galdoba/lazyam3/test.yml?branch=main)](https://github.com/Galdoba/lazyam3/actions) [![Coverage](https://img.shields.io/codecov/c/github/Galdoba/lazyam3)](https://codecov.io/gh/Galdoba/lazyam3) [![Go Report Card](https://goreportcard.com/badge/github.com/Galdoba/lazyam3)](https://goreportcard.com/report/github.com/Galdoba/lazyam3) [![Go Reference](https://pkg.go.dev/badge/github.com/Galdoba/lazyam3.svg)](https://pkg.go.dev/github.com/Galdoba/lazyam3)

lazyam3 is a Go CLI tool that scans an input directory for media files, resolves product metadata, and generates ffmpeg rendering scripts for hard-subtitle burn-in and multi-language audio selection.
DISCLAIMER: Fully `wibe-coded`, no human touched this code.

## Getting Started

```bash
go install github.com/Galdoba/lazyam3@latest
```

```bash
# Process a single pass using config file
lazyam3 process --config path/to/config.toml

# Or poll continuously (default when --once is not set)
lazyam3 process --config path/to/config.toml
```

### Configuration

Configuration is loaded via TOML (see `[script_generation.hardsub.tag]` for HSUB auto-detection). Required fields:

```toml
[paths.generator]
input_directory  = "/path/to/incoming"
output_directory = "/path/to/rendered"
metadata_path    = "/path/to/metadata.json"

[paths.executor]
buffer    = "/path/to/lazyam3/output"
in_progress = "/path/to/lazyam3/output"
done      = "/path/to/lazyam3/output"
notifications = "/path/to/lazyam3/output"
archive   = "/path/to/lazyam3/output/archive"
target    = "/path/to/lazyam3/output/target"

[script_generation.hardsub]
tag = "Forsirovannye"
```

Run `lazyam3 process --help` for all CLI flags.

## Features

- **Ingest**: scans directories, runs ffprobe for stream metadata, extracts product IDs from filenames, skips non-media files.
- **Metadata resolution**: looks up product info (series, season, episode, movie) from a metadata JSON catalog.
- **Script generation**: builds ffmpeg rendering scripts with audio filters, SRT/HSUB handling, and naming derivation.
- **HSUB auto-detection**: detects hard-subtitle sidecars via configurable filename tags.
- **Movie naming fix**: no `_se` suffix for movie units.
- **Config-driven paths**: TOML config controls generator and executor paths.
- **Empty-dir sanitation**: cleans empty directories after processing.
- **Cross-platform path resolution**: PathBuilder with canonical metadata markers.

## Contributing

Please read the [contributing guide](CONTRIBUTING.md) before submitting a PR.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
