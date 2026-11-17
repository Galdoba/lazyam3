# Spec Delta

## Purpose

Defines the error log store that lazyam3 maintains alongside the main processing history, the sidecar marker channel between a generated script and the pipeline, and what the pipeline commits to storage from those signals.

## ADDED Requirements

### Requirement: Error log reported through the sidecar marker
A rendering script SHALL report the presence and content of an error log by writing a sidecar marker named after the rendering script in the metadata directory, containing the full text of the ffmpeg/fflite error log.

#### Scenario: Render reported an error log
- **WHEN** a rendering script has copied an error log into the target directory
- **THEN** a marker named `<SCRIPT_NAME>.err` exists in the metadata directory containing the full error log text

#### Scenario: Render reported nothing
- **WHEN** a rendering script completes without an error log
- **THEN** no `<SCRIPT_NAME>.err` marker exists in the metadata directory

### Requirement: Error store captures full error text
When a completed render has an error log sidecar, the pipeline SHALL record an error log store entry whose `error_text` field holds the full text read from the `<SCRIPT_NAME>.err` sidecar.

#### Scenario: Completed render with an error log
- **WHEN** the pipeline reads a `<SCRIPT_NAME>.err` sidecar for a completed render
- **THEN** the error store entry's `error_text` field contains the full text from the sidecar

#### Scenario: Clean completed render
- **WHEN** the pipeline finds no `<SCRIPT_NAME>.err` sidecar for a completed render
- **THEN** the pipeline writes no entry to the error log store

### Requirement: Error log store records only renders with an error log
The pipeline SHALL append to the error log store only for renders that actually produced an error log. Clean renders SHALL be recorded in the processing history store and SHALL NOT add any entry to the error log store, so that every entry in it describes a real failure.

#### Scenario: Error store stays quiet on a clean run
- **WHEN** every render consumed in a pass completed without an error log
- **THEN** the error log store stays empty while the processing history store holds one entry per render

#### Scenario: Mixed pass
- **WHEN** a pass consumes several completed renders and only some of them have an error-log sidecar
- **THEN** the error log store holds exactly one entry per render that had a sidecar

### Requirement: Error store entry keyed by script name and source directory
Each error log store entry SHALL include the script name and the source directory of the render, so downstream consumers can correlate an error with its render and source.

#### Scenario: Correlating an error with its render
- **WHEN** a downstream consumer reads the error log store
- **THEN** each entry contains both the script name and the source directory, allowing correlation with the main processing history

### Requirement: Error store write follows the main history entry
The pipeline SHALL write an entry to the error log store only after the corresponding entry in the main processing history store has been committed, and only when an error-log sidecar is present.

#### Scenario: Successful commit to both stores
- **WHEN** the pipeline consumes a `.done` marker, writes to the main processing history store, and finds an error-log sidecar for that render
- **THEN** the pipeline also writes the corresponding entry to the error log store

#### Scenario: Main store write fails
- **WHEN** the pipeline fails to write to the main processing history store
- **THEN** the pipeline does not write to the error log store (atomicity: both or neither)

### Requirement: Writable state files are mandatory configuration
The configuration SHALL specify paths for the log file, the processing history store, the error log store, and the JSON status file. The process command opens each of them for writing at startup; if a path is empty or cannot be opened for writing, startup fails immediately as an infrastructure error before any processing.

#### Scenario: All writable paths configured
- **WHEN** the configuration provides non-empty paths for the log file, both JSON stores, and the status file, and each can be opened for writing
- **THEN** startup succeeds and runtime writes go to those files

#### Scenario: Writable path set to empty
- **WHEN** the configuration sets any of the writable paths to an empty string
- **THEN** startup fails with a configuration error naming the missing field, and nothing is processed

#### Scenario: Writable path cannot be opened
- **WHEN** a configured writable path cannot be opened for writing (missing parent directory, permissions)
- **THEN** the process command fails at startup with an infrastructure error, before scanning any input

### Requirement: Two naming families for execution artifacts
Artifacts the pipeline reads back SHALL be named after the rendering script; artifacts the operator's handler reads SHALL be named after the primary render output's base name.

#### Scenario: Correlating a storage record with its markers
- **WHEN** the pipeline looks up the markers belonging to a rendering script
- **THEN** it finds the processing, done, and error-log markers under that script's name

#### Scenario: Correlating operator-facing artifacts
- **WHEN** the operator's handler inspects the render output or in-progress directory
- **THEN** the deliverables, ready marker, and error log share one base name

### Requirement: Consumed signals are cleaned up together
When the pipeline has committed a completed render and cleared its working paths, it SHALL remove the done artifact and the error-log sidecar for that render together; when the commit fails, both SHALL remain for a later pass.

#### Scenario: Successful commit
- **WHEN** the pipeline commits a record and clears the render's working paths
- **THEN** both the done artifact and the error-log sidecar are removed

#### Scenario: Failed commit
- **WHEN** the processing history write fails for a completed render
- **THEN** both the done artifact and the error-log sidecar remain
