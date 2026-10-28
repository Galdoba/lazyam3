// Package filescanner recursively traverses a directory tree and builds
// a structured "snapshot". Each regular file is checked for
// writability (os.OpenFile with O_WRONLY, without O_CREATE/O_TRUNC —
// the file is not modified). This is a lightweight best-effort check that
// the file is writable (on Windows this also means no other process has
// it exclusively locked; on Unix — that permissions allow it).
//
// Error handling principle: only fatal errors are returned —
// those that prevent building the snapshot (root does not exist,
// not a directory, not readable). Everything else (stat failed,
// unable to open for writing, no permissions on directory, symlink) is recorded
// in the Error fields of the corresponding nodes and DOES NOT stop the traversal.
package fso

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// FileEntry represents a single file in the snapshot.
type FileEntry struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
	Writable bool      `json:"writable"` // true if the file could be opened for writing
	// Error is non-empty if something went wrong (stat, symlink,
	// not a regular file, write open error). Writable is
	// false in this case, but don't interpret it as
	// "guaranteed not writable" — check Error instead.
	Error string `json:"error,omitempty"`
}

// Directory is a node in the tree.
type Directory struct {
	Name    string       `json:"name"`
	Path    string       `json:"path"`
	Files   []*FileEntry `json:"files"`
	Subdirs []*Directory `json:"subdirs"`
	// Error is non-empty if the directory contents could not be read.
	Error string `json:"error,omitempty"`
}

// Snapshot is the result of a scan.
type Snapshot struct {
	Root  *Directory            `json:"root"`
	Files map[string]*FileEntry `json:"-"` // path -> file
	Dirs  map[string]*Directory `json:"-"` // path -> dir
}

// Options controls scanning behavior.
type Options struct {
	// CheckWritable determines whether to check files for writability.
	CheckWritable bool
	// SkipDir, if it returns true, skips the entire directory.
	SkipDir func(path string) bool
	// SkipFile, if it returns true, the file is not included in the snapshot.
	SkipFile func(path string) bool
}

// DefaultOptions returns the default values.
func DefaultOptions() *Options {
	return &Options{CheckWritable: true}
}

// Scan recursively traverses root and returns a Snapshot.
// It returns an error only if the snapshot cannot be built
// (root does not exist / is not a directory / is not readable).
func Scan(root string, opts *Options) (*Snapshot, error) {
	if opts == nil {
		opts = DefaultOptions()
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	info, err := os.Stat(rootAbs)
	if err != nil {
		return nil, fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root %q is not a directory", rootAbs)
	}
	// Preflight: if the root is not readable, there's no point going further —
	// we'd get an empty snapshot with no data. This is the only case
	// where a directory problem is considered fatal.
	if _, err := os.ReadDir(rootAbs); err != nil {
		return nil, fmt.Errorf("readdir root: %w", err)
	}

	snap := &Snapshot{
		Root:  &Directory{Name: filepath.Base(rootAbs), Path: rootAbs},
		Files: make(map[string]*FileEntry),
		Dirs:  make(map[string]*Directory),
	}
	snap.Dirs[rootAbs] = snap.Root
	scanDir(rootAbs, snap.Root, snap, opts)
	return snap, nil
}

func scanDir(path string, dir *Directory, snap *Snapshot, opts *Options) {
	entries, err := os.ReadDir(path)
	if err != nil {
		dir.Error = "readdir: " + err.Error()
		return
	}
	for _, e := range entries {
		full := filepath.Join(path, e.Name())

		// Directories (do NOT follow symlinks — traversal without cycles).
		if e.IsDir() {
			if opts.SkipDir != nil && opts.SkipDir(full) {
				continue
			}
			sub := &Directory{Name: e.Name(), Path: full}
			dir.Subdirs = append(dir.Subdirs, sub)
			snap.Dirs[full] = sub
			scanDir(full, sub, snap, opts)
			continue
		}

		if opts.SkipFile != nil && opts.SkipFile(full) {
			continue
		}

		fe := buildFileEntry(e, full, opts)
		dir.Files = append(dir.Files, fe)
		snap.Files[full] = fe
	}
}

func buildFileEntry(e os.DirEntry, full string, opts *Options) *FileEntry {
	fe := &FileEntry{Name: e.Name(), Path: full}

	fi, err := e.Info()
	if err != nil {
		fe.Error = "stat: " + err.Error()
		return fe
	}
	fe.Size = fi.Size()
	fe.ModTime = fi.ModTime()

	switch {
	case fi.Mode()&os.ModeSymlink != 0:
		fe.Error = "symlink"
	case !fi.Mode().IsRegular():
		fe.Error = "not a regular file: " + fi.Mode().String()
	case opts.CheckWritable:
		if err := tryOpenForWrite(full); err != nil {
			fe.Error = "write: " + err.Error()
		} else {
			fe.Writable = true
		}
	default:
		// Check is disabled — assume accessible; the consumer knows
		// that CheckWritable=false and Writable is not confirmed.
		fe.Writable = true
	}
	return fe
}

// tryOpenForWrite is the only writability check:
// O_WRONLY without O_CREATE/O_TRUNC does not modify the file.
func tryOpenForWrite(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	return f.Close()
}

// ---------- Snapshot methods ----------

// File returns the file entry by full path.
func (s *Snapshot) File(path string) (*FileEntry, bool) {
	f, ok := s.Files[path]
	return f, ok
}

// Dir returns the directory by full path.
func (s *Snapshot) Dir(path string) (*Directory, bool) {
	d, ok := s.Dirs[path]
	return d, ok
}

// FilesIn returns files in a specific directory (not recursive).
func (s *Snapshot) FilesIn(path string) []*FileEntry {
	if d, ok := s.Dirs[path]; ok {
		return d.Files
	}
	return nil
}

// FilesRecursive returns all files in the subtree.
func (s *Snapshot) FilesRecursive(path string) []*FileEntry {
	d, ok := s.Dirs[path]
	if !ok {
		return nil
	}
	var out []*FileEntry
	var walk func(*Directory)
	walk = func(dd *Directory) {
		out = append(out, dd.Files...)
		for _, sub := range dd.Subdirs {
			walk(sub)
		}
	}
	walk(d)
	return out
}

// Filter returns all snapshot files matching the predicate.
// Results are sorted by path for determinism.
func (s *Snapshot) Filter(pred func(*FileEntry) bool) []*FileEntry {
	out := make([]*FileEntry, 0, len(s.Files))
	for _, f := range s.Files {
		if pred(f) {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// NotWritable returns all files that could not be confirmed as writable.
// Useful for detecting incomplete transfers.
func (s *Snapshot) NotWritable() []*FileEntry {
	return s.Filter(func(f *FileEntry) bool { return !f.Writable })
}

// WithError returns all files that have a recorded error
// (stat, symlink, not regular, open error).
func (s *Snapshot) WithError() []*FileEntry {
	return s.Filter(func(f *FileEntry) bool { return f.Error != "" })
}

// TopDirsAllFilesWritable returns direct subdirectories of Root where
// every file successfully opened for writing. Subdirectories are not considered.
func (s *Snapshot) TopDirsAllFilesWritable() []*Directory {
	out := make([]*Directory, 0, len(s.Root.Subdirs))
	for _, d := range s.Root.Subdirs {
		if allFilesWritable(d.Files) {
			out = append(out, d)
		}
	}
	return out
}

func allFilesWritable(files []*FileEntry) bool {
	for _, f := range files {
		if !f.Writable || f.Error != "" {
			return false
		}
	}
	return true
}
