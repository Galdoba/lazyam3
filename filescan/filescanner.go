// Package filescanner рекурсивно обходит дерево каталогов и строит
// структурированный "слепок". Каждый обычный файл проверяется на
// возможность записи (os.OpenFile с O_WRONLY без O_CREATE/O_TRUNC —
// файл не модифицируется). Это дешёвая best-effort проверка того,
// что файл доступен для записи (на Windows заодно означает, что его
// никто не удерживает эксклюзивно; на Unix — что есть права).
//
// Принцип обработки ошибок: возвращаются только фатальные ошибки —
// те, из-за которых нельзя построить слепок (корень не существует,
// не директория, не читается). Всё остальное (не удалось stat,
// не открылся на запись, нет прав на каталог, симлинк) фиксируется
// в полях Error у соответствующих узлов и НЕ прерывает обход.
package filescanner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// FileEntry — один файл в слепке.
type FileEntry struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
	Writable bool      `json:"writable"` // true — удалось открыть на запись
	// Error непустой, если что-то пошло не так (stat, симлинк,
	// не регулярный файл, ошибка открытия на запись). Writable в
	// этом случае равен false, но не стоит трактовать его как
	// "гарантированно не writable" — смотрите Error.
	Error string `json:"error,omitempty"`
}

// Directory — узел дерева.
type Directory struct {
	Name    string       `json:"name"`
	Path    string       `json:"path"`
	Files   []*FileEntry `json:"files"`
	Subdirs []*Directory `json:"subdirs"`
	// Error непустой, если не удалось прочитать содержимое директории.
	Error string `json:"error,omitempty"`
}

// Snapshot — результат сканирования.
type Snapshot struct {
	Root  *Directory            `json:"root"`
	Files map[string]*FileEntry `json:"-"` // path -> file
	Dirs  map[string]*Directory `json:"-"` // path -> dir
}

// Options управляет сканированием.
type Options struct {
	// CheckWritable — проверять ли файлы на возможность записи.
	CheckWritable bool
	// SkipDir — если возвращает true, директория пропускается целиком.
	SkipDir func(path string) bool
	// SkipFile — если возвращает true, файл не попадает в слепок.
	SkipFile func(path string) bool
}

// DefaultOptions — значения по умолчанию.
func DefaultOptions() *Options {
	return &Options{CheckWritable: true}
}

// Scan рекурсивно обходит root и возвращает Snapshot.
// Возвращает ошибку только если слепок построить невозможно
// (корень не существует / не директория / не читается).
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
	// Preflight: если корень не читается, дальше идти нет смысла —
	// получим пустой слепок без данных. Это единственный случай,
	// когда проблема с директорией считается фатальной.
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

		// Каталоги (симлинки НЕ раскрываем — обход без циклов).
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
		// Проверка отключена — считаем доступным; потребитель знает,
		// что CheckWritable=false и Writable не подтверждён.
		fe.Writable = true
	}
	return fe
}

// tryOpenForWrite — единственная проверка writability:
// O_WRONLY без O_CREATE/O_TRUNC не изменяет файл.
func tryOpenForWrite(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	return f.Close()
}

// ---------- методы Snapshot ----------

// File возвращает запись по полному пути.
func (s *Snapshot) File(path string) (*FileEntry, bool) {
	f, ok := s.Files[path]
	return f, ok
}

// Dir возвращает директорию по полному пути.
func (s *Snapshot) Dir(path string) (*Directory, bool) {
	d, ok := s.Dirs[path]
	return d, ok
}

// FilesIn — файлы конкретной директории (не рекурсивно).
func (s *Snapshot) FilesIn(path string) []*FileEntry {
	if d, ok := s.Dirs[path]; ok {
		return d.Files
	}
	return nil
}

// FilesRecursive — все файлы поддерева.
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

// Filter — все файлы слепка, удовлетворяющие предикату.
// Результат отсортирован по пути для детерминизма.
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

// NotWritable — все файлы, которые не удалось подтвердить как доступные
// на запись. Удобно для обнаружения незавершённых передач.
func (s *Snapshot) NotWritable() []*FileEntry {
	return s.Filter(func(f *FileEntry) bool { return !f.Writable })
}

// WithError — все файлы, у которых зафиксирована какая-либо ошибка
// (stat, симлинк, не регулярный, ошибка открытия).
func (s *Snapshot) WithError() []*FileEntry {
	return s.Filter(func(f *FileEntry) bool { return f.Error != "" })
}

// TopDirsAllFilesWritable — прямые поддиректории Root, в которых
// каждый файл успешно открылся на запись. Поддиректории не учитываются.
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