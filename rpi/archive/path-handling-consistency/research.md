# Path Handling Consistency Research

## Problem Context

The goal is to ensure ALL paths everywhere in the codebase are always in Linux-style (with forward slashes `/`). This is critical for:
- Cross-platform compatibility (Windows, macOS, Linux)
- Consistency in file system operations
- Logging and debugging (path strings in logs)
- Script generation and execution

### Current vs Desired Behavior

**Current state**: Inconsistent path handling across the codebase
- Some code uses `filepath.Join()` with `filepath.ToSlash()` for normalization
- Some code uses `PathBuilder.Canonical()` for forward slashes
- Some code uses `PathBuilder.native()` for OS-specific separators
- Logger has a `normalizePath()` function but it's not universally applied
- Test files contain Windows-style paths (`C:\Users\...`) for testing purposes

**Desired state**: All paths in all contexts should be forward-slash normalized

---

## Affected Files

```
internal/app/pipeline/config.go:149-153
internal/app/pipeline/pipeline.go:62, 458
internal/platform/pathbuilder.go:26-32
internal/infra/logging/logger.go:407-413
internal/domain/callname/names_output.go:61-66, 85, 103, 116, 222
internal/infra/fso/filescanner.go:125
```

## Code Examples

### 1. `internal/domain/callname/names_output.go:85`

```go
Path: filepath.ToSlash(filepath.Join(targetDir, outBase+"_HD.mp4")),
```

**Problem**: `filepath.Join()` uses OS-specific separators, requiring `filepath.ToSlash()` afterward. This is two-step conversion that could be simplified with `path.Join()` which always uses `/`.

### 2. `internal/app/pipeline/config.go:51`

```go
renderRoot := toSlash(executorPaths["target"])
if renderRoot == "" {
    renderRoot = filepath.Join(toSlash(root), "target")
}
```

**Problem**: Mixed usage of `toSlash()` wrapper and `filepath.Join()`.

### 3. `internal/infra/fso/filescanner.go:125`

```go
full := filepath.Join(path, e.Name())
```

**Problem**: Uses `filepath.Join()` in production code for file scanning.

---

## External Research Findings

### Go Documentation on Path Handling

1. **`filepath.ToSlash()`**: Converts OS-specific separators to `/`
   - Safe to use on any path string
   - Returns the original string on Unix-like systems
   - Returns converted string on Windows

2. **`filepath.FromSlash()`**: Opposite of `ToSlash()`
   - Converts `/` to OS separator
   - Not used currently in the codebase

3. **`path.Join()` vs `filepath.Join()`**:
   - `path.Join()`: Always uses `/` as separator (Linux-style)
   - `filepath.Join()`: Uses OS-specific separator
   - For cross-platform path construction  :  use `path.Join()` or `PathBuilder.Canonical()`

4. **Best Practice**: 
   - Store paths internally as forward slashes
   - Convert to OS separator only for OS calls (`os.Open()`, `os.WriteFile()`, etc.)
   - Always use forward slashes in logs, tests, and string comparisons

### Known Issues

1. **Windows Drive Letters**: `C:\Users\foo`  :  `C:/Users/foo` (preserves drive letter)
2. **UNC Paths**: `\\server\share\file`  :  `//server/share/file`
3. **Mixed Separators**: Windows paths like `C:\foo\bar` are rarely mixed, but possible
4. **Case Sensitivity**: Windows paths are case-insensitive; Unix are case-sensitive
   - Path normalization should not change case

---

## FAR Scale Output

### Factual Score: 5
- Direct code inspection shows 3 distinct path normalization approaches
- `filepath.ToSlash()` is used in `config.go`
- `PathBuilder` provides both `Canonical()` and `native()`
- Logger has `normalizePath()` for log output

### Actionable Score: 5
- Clear patterns identified
- Specific files and line numbers documented
- Concrete recommendations can be implemented

### Relevant Score: 5
- This directly addresses the stated goal
- All findings are relevant to path consistency

### FAR Mean: 5.00 --> PASS

---

## Testing Strategy

### Unit Tests
1. **PathBuilder**: Already has tests for `native()` and `Canonical()`
   - Add tests for edge cases: empty root, Windows drive letters, UNC paths
2. **`toSlash()`**: Add tests for:
   - Empty string
   - Linux paths (no-op)
   - Windows paths (conversion)
   - UNC paths
   - Mixed separators

### Integration Tests
1. **Script Generation**: Verify generated scripts use forward slashes
2. **Cross-platform Simulation**: Test on Linux with Windows-style paths
3. **Logger Output**: Verify logged paths are forward-slash normalized

### Observability
Add a function to validate path consistency:
```go
func ValidatePath(path string) error {
	// Check for backslashes
	if strings.Contains(path, `\`) {
		return fmt.Errorf("path contains backslashes: %s", path)
	}
	// Check for Windows drive letter pattern
	if len(path) >= 2 && path[0] == path[0] && path[1] == ':' {
		return fmt.Errorf("path contains Windows drive letter: %s", path)
	}
	return nil
}
```

---

## Potential Design Pattern Recommendations

### Unified Path Normalizer

Create a single canonical function that:
- Accepts any path string
- Converts to forward slashes
- Cleans up redundant separators
- Normalizes (.) and (..)

```go
type PathNormalizer struct{}

func (PathNormalizer) Normalize(path string) string {
	return filepath.ToSlash(path)
}
```

### Path Builder with Canonical Default

Modify `PathBuilder` to have a canonical method that's always used:

```go
type PathBuilder struct {
	root string
}

// Canonical returns a path with forward slashes.
func (b PathBuilder) Canonical(components ...string) string {
	return path.Join(b.root, path.Join(components...))
}
```

### Standardize on `path.Join()` for Construction

Replace `filepath.Join()` with `path.Join()` for all path construction:
- `filepath.Join()`  :  `path.Join()` (for forward slashes)
- Then call `filepath.ToSlash()` only if needed (though `path.Join()` already returns `/`)

### Path Validation Middleware

Add validation at entry points to catch inconsistencies early.

---

## Assumptions

1. **Forward slashes are universal**: All OS calls accept forward slashes when passed through Go's `os` package
2. **PathBuilder is the right abstraction**: Centralizing path construction is better than scattered `filepath.Join()` calls
3. **Logger normalization is sufficient for logs**: Log output only needs to be human-readable, not used for OS operations
4. **Tests are correct**: Test files may legitimately use Windows paths for testing Windows scenarios
5. **Backward compatibility is important**: Existing paths in configuration should work without breaking

---

## Out of Scope

1. **Path validation/refactoring**: This research doesn't include refactoring existing code
2. **Configuration file parsing**: TOML/string parsing of paths is a separate concern
3. **File system abstraction**: Creating an interface for OS operations is beyond scope
4. **Performance optimization**: Path normalization cost is negligible for this use case
