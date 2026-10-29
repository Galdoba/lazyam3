# Naming Problem — Cross-Platform Path Resolution

## Problem

When the generator runs on Windows and the script executor runs on Linux, not all paths are properly resolved, causing path mismatches between the source file names and the output file names.

## Root Cause Analysis

The naming script uses **Unix-style backslash notation** (`\buffer\IN\metadata\`) mixed with **forward slashes** for system paths, creating platform inconsistencies:

### Example from `doc/naming.md`

```bash
# Inconsistent path notation
BUFFER="/home/pemaltynov/IN/"                          # Forward slash (Unix)
IN_PROGRESS="/home/pemaltynov/IN/_IN_PROGRESS/"       # Forward slash (Unix)
NOTIFICATIONS="/home/pemaltynov/IN/notifications/"     # Forward slash (Unix)

# But this line uses backslashes with escaping:
[[ -f "\\192.168.31.4\buffer\IN\metadata/..." ]]      # Backslashes (Windows-style)
```

### The Issue

1. **Windows path format**: `192.168.31.4\buffer\IN\metadata\filename.processing`
2. **Linux path format**: `/home/pemaltynov/IN/_IN_PROGRESS/filename`
3. **Mixed notation**: The script uses both forward and backward slashes inconsistently

### Consequences

- **Path resolution failures**: Files generated on Windows may not be found on Linux
- **File movement failures**: `mv` commands may fail or move files to wrong locations
- **Metadata mismatch**: `.processing` and `.done` markers may not match actual file paths
- **Cross-platform incompatibility**: Pipeline breaks when generator and executor run on different OSes

## Solution Requirements

1. **Standardize path format**: Use consistent forward slashes (`/`) for all paths
2. **Remove Windows-specific syntax**: Eliminate backslash notation and IP-based host references
3. **Platform abstraction**: Paths should be OS-independent, using the filesystem's native separator
4. **Configurable paths**: Define paths in a config file (TOML) for flexibility
5. **Path validation**: Ensure paths exist before file operations

## Expected Behavior

- Paths should use forward slashes regardless of underlying OS
- All file operations (move, copy, detect) should work across platforms
- Metadata files (`.processing`, `.done`) should use consistent naming
- Script executor should detect files using the same path format as generator

## Files Affected

- Script executor (`.processing` → `.done` marker logic)
- File mover logic (generating script names for output files)
- Path construction in generator and executor
- Any hardcoded paths in the naming script
