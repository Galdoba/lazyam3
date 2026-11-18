# Spec Delta

## Purpose

Establishes which catalog product an incoming directory refers to from the names of its files, and defines what an unidentified unit means for everything downstream of that decision.

## ADDED Requirements

### Requirement: Accepted incoming name shape
An incoming file name SHALL match `<digits> [ '-' tag [ '_' tag ]* ] [ '.' extension ]`, and the leading digit run MUST be followed by `-`, `.`, or the end of the name. A name that does not match SHALL carry no product identity claim.

#### Scenario: Most complex accepted form
- **WHEN** `63597-rus_Russkii_Forsirovannye.srt` is examined
- **THEN** it is a valid name and its ID claim is `63597`

#### Scenario: Bare ID with no extension
- **WHEN** `63574` is examined
- **THEN** it is a valid name and its ID claim is `63574`

#### Scenario: Letters directly after digits
- **WHEN** `12254rus.srt` is examined
- **THEN** it carries no ID claim

#### Scenario: Underscore directly after digits
- **WHEN** `0_show.mp4` or `63604_1080.mp4` is examined
- **THEN** neither carries an ID claim

#### Scenario: No leading digits
- **WHEN** `no_id_in_video.mp4` or `something-unrelated.md` is examined
- **THEN** neither carries an ID claim

### Requirement: The names that take part are chosen before extraction
The set of candidate names that take part in an identity decision SHALL be selected before extraction, and the extraction step MUST NOT admit or exclude names from that set on its own.

#### Scenario: A name outside the candidate set cannot void an identity
- **WHEN** a directory holds `123.mp4`, `123-rus.srt` and `notes.txt`, and the candidates selected for the unit are its media files
- **THEN** the unit identity is `123`

#### Scenario: A candidate set may change without changing the rule
- **WHEN** a later change selects a different set of candidates for the same directory
- **THEN** the identity rule applied to those names is the same rule, applied to the names it was given

### Requirement: Identity requires unanimity
Every candidate name of a unit SHALL claim the same product ID. If any candidate carries no claim, or the candidates claim different IDs, the unit SHALL have no product identity, and no candidate's claim SHALL be adopted on its own.

#### Scenario: Two different claims void each other
- **WHEN** the candidates of one unit are `12345.mp4` and `12346.mp4`
- **THEN** the unit identity is `0` and neither `12345` nor `12346` is adopted

#### Scenario: A unit is never identified against one of its own names
- **WHEN** `63603.mp4` and `12254-rus.srt` both take part in the decision
- **THEN** the unit identity is `0` and neither `63603` nor `12254` is adopted

#### Scenario: A candidate with no claim voids the identity
- **WHEN** the candidates of one unit are `63603.mp4` and `no_id_in_video.mp4`
- **THEN** the unit identity is `0` and `63603` is not adopted

#### Scenario: Candidates that agree establish the identity
- **WHEN** the candidates of one unit are `63603.mp4` and `63603.mkv`
- **THEN** the unit identity is `63603`

#### Scenario: No candidate claims
- **WHEN** every candidate of a unit carries no claim
- **THEN** the unit identity is `0`

### Requirement: An unresolved identity is not a failure
The system SHALL NOT abort a unit, report an error, or substitute a placeholder ID because its names did not resolve to one identity. An unresolved unit SHALL continue through the pipeline as unidentified.

#### Scenario: A single unrecognisable name
- **WHEN** a directory contains `id_is_03_video.mp4` only
- **THEN** the unit is built and reported as unidentified

#### Scenario: Disagreeing names are no longer a rejection
- **WHEN** the candidates of a unit carry different claims
- **THEN** the unit continues as unidentified instead of being rejected with an inconsistent-identities error

#### Scenario: No placeholder ID is produced
- **WHEN** a unit's identity is unresolved
- **THEN** no component downstream receives a fabricated ID in the shape of a catalog key

### Requirement: Zero is reserved for "no identity"
The value `0` SHALL be reserved for "this unit has no established product identity" and MUST NOT be produced as an ID claim read from a name.

#### Scenario: Literal zero stem
- **WHEN** `0.mp4` is examined
- **THEN** it carries no ID claim

#### Scenario: A short claim is still a claim
- **WHEN** `1.mp4` is examined as the only candidate
- **THEN** the unit identity is `1` and the unit is identified

### Requirement: An unidentified unit carries no fabricated identity
An unidentified unit SHALL carry `0` as its identity, and the pipeline SHALL report it as unidentified rather than as a product lookup for a placeholder number.

#### Scenario: Unidentified unit falls back to the directory name
- **WHEN** a unit is unidentified
- **THEN** no product is found for it and the unit proceeds with a fallback product named after the source directory, carrying identity `0`

#### Scenario: Unidentified unit is recorded
- **WHEN** an unidentified unit finishes processing
- **THEN** its processing record and its completion event report identity `0`

### Requirement: Product identity is derived from names in exactly one place
Only the ingest stage SHALL read a product identity out of incoming file names. Naming and rendering SHALL use the identity the pipeline resolved and MUST NOT re-derive it from a name.

#### Scenario: Naming an unidentified unit
- **WHEN** call names are built for a unit whose identity is `0`
- **THEN** the call names are produced and the unit is not failed for a missing identity

#### Scenario: A unit with no identity still names its files with their tags
- **WHEN** a unit's names are `63603.mp4`, `no_id_in_video.mp4` and `63603-rus.srt`
- **THEN** the unit identity is `0`, and the call name for the subtitle keeps the `rus` tag
