# Backlog

## Known Issues / Footguns

### Empty input directories.
When script is rendered, programm moves files out of input directory. Next cycle detects empty directory and signals minor error - so logs are getting messy.
We need a way to handle this issue.
- directory MUST not be deleted untill script execuuted successfuly.
- no way to track when script is executed or not.
- solid marker that script executed succesfuly SHOULD be an artifact created by script
  - how artifact should be ingested?
- program does not know where scripts are archived (more options MUST be explored)
  - Option 1: store a local path for scripts to be stored?
  - Option 2: infuse a signal from script to program via marker file (report?) in `Output Directory` - the one programm knows about and renders scripts to?

## Feature Queue

### Move ExtractID from ingest to callname subdomain
**Status**: planned
**Problem**: `callname` currently duplicates `ingest.ExtractID` for domain purity (no callname → ingest dependency). ID extraction logic belongs in the callname subdomain, not ingest.
**Plan**: Migrate `ExtractID` logic to `callname.ExtractID`, migrate `ingest` callers to use it, then deprecate/remove `ingest.ExtractID`.

### FFProbe Media Detection Proposition
**Status**: proposal, future consideration
**Problem**: Non-media file detection relies on ffprobe probe failure. Alternative: ffprobe could classify files explicitly (media vs non-media) via a detection flag or exit code.
**Proposal**: Explore ffprobe options or a wrapper that returns a media/non-media classification, reducing brittle error-parsing in ingest logic.
