# script-generation Specification

## Purpose

Defines what a generated rendering script places in the render output directory and the metadata directory, and the naming those artifacts carry, so both the operator's handler and lazyam3 itself can rely on them.

## Requirements

### Requirement: Error log copy to the render output directory
A rendering script SHALL copy the non-fatal error log produced in the in-progress directory into the render output (target) directory, named after the primary render output's base name with the `.#err` suffix.

#### Scenario: Render produced an error log
- **WHEN** a rendering script finishes its render and an error log exists in the in-progress directory beside the source file
- **THEN** the script places a copy of that log in the target directory as `<OUTBASE>.#err`

#### Scenario: Render produced no error log
- **WHEN** a rendering script finishes its render and no error log exists in the in-progress directory
- **THEN** the script places no `.#err` file in the target directory
- **AND** the script continues to its completion steps without failing

### Requirement: Error log sidecar in metadata directory
A rendering script SHALL copy the non-fatal error log into the metadata directory as `<SCRIPT_NAME>.err`, containing the full text of the ffmpeg/fflite error log. This sidecar is the communication channel between the script and the pipeline.

#### Scenario: Render reported an error log
- **WHEN** a rendering script has copied an error log into the target directory
- **THEN** a file named `<SCRIPT_NAME>.err` exists in the metadata directory containing the full error log text

#### Scenario: Render reported nothing
- **WHEN** a rendering script completes without an error log
- **THEN** no `<SCRIPT_NAME>.err` file exists in the metadata directory

### Requirement: Ready marker copy to the render output directory
A rendering script SHALL place a ready marker named `<OUTBASE>.ready` in the render output (target) directory, after any error log copy has been performed.

#### Scenario: Render completes with ready marker
- **WHEN** a rendering script completes successfully
- **THEN** `<OUTBASE>.ready` appears in the target directory

#### Scenario: Ready marker follows error log
- **WHEN** a rendering script completes after copying an error log
- **THEN** the error log is already in the target directory before the ready marker is placed

### Requirement: Error log source is left in place
A rendering script SHALL NOT move or delete the error log it reads from the in-progress directory; the operator's own tooling owns that directory.

#### Scenario: Error log copied
- **WHEN** a rendering script copies an error log into the target directory
- **THEN** the file in the in-progress directory remains where it was

### Requirement: Operator-facing artifacts share the render base name
Every artifact a rendering script places in the render output directory SHALL be named from the same base name as the primary video output, so the operator's handler can correlate deliverables, completion markers, and error logs by name alone.

#### Scenario: Ready marker and error log for one render
- **WHEN** a render produces both a ready marker and an error log
- **THEN** both carry the identical base name, differing only in the `.ready` and `.#err` suffixes

### Requirement: No error log in the notification directory
A rendering script SHALL NOT place an error log in the notification directory. The operator's notifier already reports every log it finds in the in-progress directory, and a second copy would report the same render twice.

#### Scenario: Errored render
- **WHEN** a render produces an error log that the notifier also sees in the in-progress directory
- **THEN** the operator is notified once, under the name the notifier chose
- **AND** the generated script placed no `.#err` file in the notification directory

### Requirement: No ready marker in the notification directory
A rendering script SHALL NOT place a ready marker in the notification directory. The operator's notification handler manages `.ready` files in the in-progress directory on its own schedule.

#### Scenario: Clean render
- **WHEN** a rendering script completes without an error log
- **THEN** the generated script placed no `.ready` file in the notification directory
