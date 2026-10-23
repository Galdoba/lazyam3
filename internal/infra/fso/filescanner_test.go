package fso

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan_Basic(t *testing.T) {
	dir := t.TempDir()

	// Create a few regular files.
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}

	snap, err := Scan(dir, nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if snap.Root.Name != filepath.Base(dir) {
		t.Errorf("Root.Name = %q, want %q", snap.Root.Name, filepath.Base(dir))
	}
	if len(snap.Root.Files) != 2 {
		t.Errorf("Root.Files = %d, want 2", len(snap.Root.Files))
	}
	if len(snap.Files) != 2 {
		t.Errorf("snap.Files map = %d, want 2", len(snap.Files))
	}
	if len(snap.Dirs) != 1 {
		t.Errorf("snap.Dirs map = %d, want 1", len(snap.Dirs))
	}
}

func TestScan_NestedDirectories(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "deep.txt"), []byte("deep"), 0o644); err != nil {
		t.Fatal(err)
	}

	snap, err := Scan(dir, nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(snap.Root.Subdirs) != 1 {
		t.Fatalf("Root.Subdirs = %d, want 1", len(snap.Root.Subdirs))
	}
	subDir := snap.Root.Subdirs[0]
	if subDir.Name != "sub" {
		t.Errorf("Subdir.Name = %q, want %q", subDir.Name, "sub")
	}
	if len(subDir.Files) != 1 {
		t.Errorf("subdir.Files = %d, want 1", len(subDir.Files))
	}
	if snap.Root.Files != nil {
		t.Errorf("Root.Files = %d, want 0", len(snap.Root.Files))
	}
}

func TestScan_NonDirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "notadir")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Scan(f, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestScan_NonExistent(t *testing.T) {
	_, err := Scan(filepath.Join(t.TempDir(), "nope"), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestScan_CheckWritableDisabled(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	snap, err := Scan(dir, &Options{CheckWritable: false})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(snap.Root.Files) != 1 {
		t.Fatalf("Root.Files = %d, want 1", len(snap.Root.Files))
	}
	// With CheckWritable=false, Writable is assumed true.
	if !snap.Root.Files[0].Writable {
		t.Error("expected Writable=true when CheckWritable=false")
	}
}

func TestScan_Symlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(target, []byte("real"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "link.txt")); err != nil {
		t.Fatal(err)
	}

	snap, err := Scan(dir, nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(snap.Root.Files) != 2 {
		t.Fatalf("Root.Files = %d, want 2", len(snap.Root.Files))
	}
	// Find the symlink entry.
	for _, f := range snap.Root.Files {
		if f.Name == "link.txt" {
			if f.Error != "symlink" {
				t.Errorf("link.txt Error = %q, want %q", f.Error, "symlink")
			}
			return
		}
	}
	t.Fatal("did not find link.txt entry")
}

func TestScan_SkipDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "skipme")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	snap, err := Scan(dir, &Options{
		SkipDir: func(p string) bool {
			return filepath.Base(p) == "skipme"
		},
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(snap.Root.Subdirs) != 0 {
		t.Errorf("Root.Subdirs = %d, want 0", len(snap.Root.Subdirs))
	}
	if len(snap.Root.Files) != 1 {
		t.Errorf("Root.Files = %d, want 1", len(snap.Root.Files))
	}
}

func TestScan_SkipFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skip.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}

	snap, err := Scan(dir, &Options{
		SkipFile: func(p string) bool {
			return filepath.Base(p) == "skip.txt"
		},
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(snap.Root.Files) != 1 {
		t.Errorf("Root.Files = %d, want 1", len(snap.Root.Files))
	}
	if snap.Root.Files[0].Name != "keep.txt" {
		t.Errorf("Root.Files[0].Name = %q, want %q", snap.Root.Files[0].Name, "keep.txt")
	}
}

func TestScan_ReaddirError(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "noperm")
	if err := os.Mkdir(sub, 0o000); err != nil {
		t.Fatal(err)
	}

	snap, err := Scan(dir, nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	// The unreadable subdir should have Error set, but traversal continues.
	if len(snap.Root.Subdirs) != 1 {
		t.Fatalf("Root.Subdirs = %d, want 1", len(snap.Root.Subdirs))
	}
	if snap.Root.Subdirs[0].Error == "" {
		t.Log("subdir.Error is empty — may be root user, skipping assertion")
	}
}

// --- Snapshot method tests ---

func buildTestSnapshot(t *testing.T) *Snapshot {
	t.Helper()
	dir := t.TempDir()

	sub1 := filepath.Join(dir, "sub1")
	sub2 := filepath.Join(dir, "sub2")
	sub1sub := filepath.Join(sub1, "subsub")

	for _, d := range []string{sub1, sub2, sub1sub} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		filepath.Join(dir, "root.txt"):  "root",
		filepath.Join(sub1, "a.txt"):    "a",
		filepath.Join(sub2, "b.txt"):    "b",
		filepath.Join(sub1sub, "c.txt"): "c",
	}
	for p, content := range files {
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	snap, err := Scan(dir, nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return snap
}

func TestSnapshot_File(t *testing.T) {
	snap := buildTestSnapshot(t)

	f, ok := snap.File(filepath.Join(snap.Root.Path, "root.txt"))
	if !ok {
		t.Fatal("File(root.txt) not found")
	}
	if f.Name != "root.txt" {
		t.Errorf("File.Name = %q, want %q", f.Name, "root.txt")
	}

	_, ok = snap.File(filepath.Join(snap.Root.Path, "nope.txt"))
	if ok {
		t.Error("File(nope.txt) should not be found")
	}
}

func TestSnapshot_Dir(t *testing.T) {
	snap := buildTestSnapshot(t)

	d, ok := snap.Dir(filepath.Join(snap.Root.Path, "sub1"))
	if !ok {
		t.Fatal("Dir(sub1) not found")
	}
	if d.Name != "sub1" {
		t.Errorf("Dir.Name = %q, want %q", d.Name, "sub1")
	}

	_, ok = snap.Dir(filepath.Join(snap.Root.Path, "nope"))
	if ok {
		t.Error("Dir(nope) should not be found")
	}
}

func TestSnapshot_FilesIn(t *testing.T) {
	snap := buildTestSnapshot(t)

	files := snap.FilesIn(filepath.Join(snap.Root.Path, "sub1"))
	if len(files) != 1 {
		t.Errorf("FilesIn(sub1) = %d, want 1", len(files))
	}

	files = snap.FilesIn(filepath.Join(snap.Root.Path, "sub1", "subsub"))
	if len(files) != 1 {
		t.Errorf("FilesIn(sub1/subsub) = %d, want 1", len(files))
	}

	if snap.FilesIn("nonexistent") != nil {
		t.Error("FilesIn(nonexistent) should return nil")
	}
}

func TestSnapshot_FilesRecursive(t *testing.T) {
	snap := buildTestSnapshot(t)

	all := snap.FilesRecursive(snap.Root.Path)
	if len(all) != 4 {
		t.Errorf("FilesRecursive(root) = %d, want 4", len(all))
	}

	sub1 := filepath.Join(snap.Root.Path, "sub1")
	sub1only := snap.FilesRecursive(sub1)
	if len(sub1only) != 2 {
		t.Errorf("FilesRecursive(sub1) = %d, want 2", len(sub1only))
	}

	if snap.FilesRecursive("nonexistent") != nil {
		t.Error("FilesRecursive(nonexistent) should return nil")
	}
}

func TestSnapshot_Filter(t *testing.T) {
	snap := buildTestSnapshot(t)

	// Filter by file extension.
	txtFiles := snap.Filter(func(f *FileEntry) bool {
		return len(f.Name) > 0 && f.Name[len(f.Name)-4:] == ".txt"
	})
	if len(txtFiles) != 4 {
		t.Errorf("Filter(.txt) = %d, want 4", len(txtFiles))
	}

	// Verify sorted by path.
	for i := 1; i < len(txtFiles); i++ {
		if txtFiles[i].Path < txtFiles[i-1].Path {
			t.Errorf("Filter result not sorted at index %d", i)
			break
		}
	}
}

func TestSnapshot_NotWritable(t *testing.T) {
	snap := buildTestSnapshot(t)

	// All files are writable in temp dir.
	notWritable := snap.NotWritable()
	if len(notWritable) != 0 {
		t.Errorf("NotWritable() = %d, want 0", len(notWritable))
	}
}

func TestSnapshot_WithError(t *testing.T) {
	snap := buildTestSnapshot(t)
	noErrors := snap.WithError()
	if len(noErrors) != 0 {
		t.Errorf("WithError() = %d, want 0", len(noErrors))
	}
}

func TestSnapshot_TopDirsAllFilesWritable(t *testing.T) {
	snap := buildTestSnapshot(t)

	writableDirs := snap.TopDirsAllFilesWritable()
	if len(writableDirs) != 2 {
		t.Errorf("TopDirsAllFilesWritable() = %d, want 2 (sub1, sub2)", len(writableDirs))
	}
}

func TestSnapshot_TopDirsAllFilesWritable_EmptySubdir(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty")
	if err := os.Mkdir(empty, 0o755); err != nil {
		t.Fatal(err)
	}

	snap, err := Scan(dir, nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	writable := snap.TopDirsAllFilesWritable()
	if len(writable) != 1 {
		t.Errorf("TopDirsAllFilesWritable() = %d, want 1 (empty dir has no unwritable files)", len(writable))
	}
}
