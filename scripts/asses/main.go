package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/Galdoba/lazyam3/internal/domain/metadata" // замените на ваш путь
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "check":
		cmdCheck(os.Args[2:])
	case "check-required":
		cmdCheckRequired(os.Args[2:])
	case "check-null":
		cmdCheckNull(os.Args[2:])
	case "field-values":
		cmdFieldValues(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `usage: %s <command> [flags]

commands:
  check            декодировать JSON в metadata.Root и показать OK / ошибку схемы
  check-required   найти отсутствующие обязательные поля
  check-null       найти все null-значения
  field-values     напечатать все строки файла, содержащие указанное поле

run '%s <command> -h' для флагов команды
`, os.Args[0], os.Args[0])
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

// ---------------------------------------------------------------------------
// check
// ---------------------------------------------------------------------------

func cmdCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	filePath := fs.String("file", "metadata.json", "путь к JSON")
	fs.Parse(args)

	data, err := os.ReadFile(*filePath)
	if err != nil {
		die(err)
	}

	var root metadata.Root
	if err := DecodeJSONBytes(data, &root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("OK: series=%d movies=%d\n", len(root.Series), len(root.Movies))
}

// ---------------------------------------------------------------------------
// field-values
// ---------------------------------------------------------------------------

func cmdFieldValues(args []string) {
	fs := flag.NewFlagSet("field-values", flag.ExitOnError)
	filePath := fs.String("file", "metadata.json", "путь к JSON")
	field := fs.String("field", "", "имя поля (обязательно)")
	noLines := fs.Bool("no-lines", false, "не печатать строки, только счётчики")
	fs.Parse(args)

	if *field == "" {
		die(errors.New("-field обязателен"))
	}

	data, err := os.ReadFile(*filePath)
	if err != nil {
		die(err)
	}

	if !*noLines {
		printFieldLines(data, *field)
	}

	printFieldValueCounts(data, *field)
}

func printFieldLines(data []byte, field string) {
	needle := []byte(`"` + field + `"`)

	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 1<<20), 1<<24)

	lineNo, printed := 0, 0
	for sc.Scan() {
		lineNo++
		if bytes.Contains(sc.Bytes(), needle) {
			printed++
			fmt.Printf("%6d: %s\n", lineNo, strings.TrimRight(string(sc.Bytes()), "\r"))
		}
	}
	if err := sc.Err(); err != nil {
		die(err)
	}
	if printed == 0 {
		fmt.Fprintf(os.Stderr, "поле %q не найдено в строках\n", field)
	} else {
		fmt.Fprintf(os.Stderr, "matched lines: %d\n", printed)
	}
}

func printFieldValueCounts(data []byte, field string) {
	root := gjson.ParseBytes(data)

	counts := map[string]int{}
	var order []string // порядок первого появления

	walkGJSON(root, "@this", func(path string, v gjson.Result) {
		if lastPathSegment(path) != field {
			return
		}
		repr := valueRepr(v)
		if _, ok := counts[repr]; !ok {
			order = append(order, repr)
		}
		counts[repr]++
	})

	type entry struct {
		repr  string
		count int
	}
	entries := make([]entry, 0, len(order))
	total := 0
	for _, r := range order {
		entries = append(entries, entry{r, counts[r]})
		total += counts[r]
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count != entries[j].count {
			return entries[i].count > entries[j].count
		}
		return entries[i].repr < entries[j].repr
	})

	fmt.Fprintf(os.Stderr, "\nvalue counts for %q (total %d, unique %d):\n",
		field, total, len(entries))
	for _, e := range entries {
		fmt.Printf("%6d  %s\n", e.count, e.repr)
	}
}

func lastPathSegment(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[i+1:]
	}
	return path
}

// valueRepr даёт каноническое представление значения для группировки:
// строки — в кавычках (чтобы "0" и 0 не смешались), числа — как есть,
// null/true/false — словами.
func valueRepr(v gjson.Result) string {
	switch v.Type {
	case gjson.Null:
		return "null"
	case gjson.String:
		return strconv.Quote(v.String())
	case gjson.True:
		return "true"
	case gjson.False:
		return "false"
	default:
		// Number, JSON (объект/массив)
		return v.Raw
	}
}

// ---------------------------------------------------------------------------
// check-required (на gjson)
// ---------------------------------------------------------------------------

// Пути в нотации gjson. '#' — любой индекс массива.
var requiredByPath = map[string][]string{
	"@this":                         {"series", "movies"},
	"series.#":                      {"id", "title", "ageRestrictions"},
	"series.#.persons.#":            {"type", "name"},
	"series.#.seasons.#":            {"id", "seasonNumber", "episodes"},
	"series.#.seasons.#.persons.#":  {"type", "name"},
	"series.#.seasons.#.episodes.#": {"id", "number"},
	"movies.#":                      {"id", "title", "premierYear"},
	"movies.#.persons.#":            {"type", "name"},
	"movies.#.episodes.#":           {"id", "number"},
}

func cmdCheckRequired(args []string) {
	fs := flag.NewFlagSet("check-required", flag.ExitOnError)
	filePath := fs.String("file", "metadata.json", "путь к JSON")
	fs.Parse(args)

	data, err := os.ReadFile(*filePath)
	if err != nil {
		die(err)
	}

	li := newLineIndex(data)

	var issues []requiredIssue // ← было: []issue

	root := gjson.ParseBytes(data)
	root.ForEach(func(key, value gjson.Result) bool {
		checkRequiredInValue(key.String(), value, li, &issues)
		return true
	})
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].line != issues[j].line {
			return issues[i].line < issues[j].line
		}
		return issues[i].col < issues[j].col
	})

	for _, iss := range issues {
		fmt.Printf("%d:%d  %s: missing required field %q\n",
			iss.line, iss.col, iss.path, iss.field)
	}
}

// checkRequiredInValue рекурсивно обходит значение и для объектов,
// путь которых есть в requiredByPath, проверяет обязательные ключи.
func checkRequiredInValue(path string, v gjson.Result, li *lineIndex, issues *[]requiredIssue) {
	if v.IsArray() {
		v.ForEach(func(_, item gjson.Result) bool {
			checkRequiredInValue(path, item, li, issues)
			return true
		})
		return
	}
	if !v.IsObject() {
		return
	}

	// Проверяем обязательные ключи для этого пути.
	if req, ok := requiredByPath[path]; ok {
		for _, f := range req {
			if !v.Get(f).Exists() {
				line, col := li.locate(int64(v.Index))
				*issues = append(*issues, requiredIssue{line, col, path, f})
			}
		}
	}
	// Рекурсивно спускаемся по вложенным объектам/массивам,
	// расширяя путь.
	v.ForEach(func(k, sub gjson.Result) bool {
		childPath := path
		if childPath == "@this" {
			childPath = k.String()
		} else {
			childPath = childPath + "." + k.String()
		}
		checkRequiredInValue(childPath, sub, li, issues)
		return true
	})
}

// ---------------------------------------------------------------------------
// check-null (на gjson)
// ---------------------------------------------------------------------------

func cmdCheckNull(args []string) {
	fs := flag.NewFlagSet("check-null", flag.ExitOnError)
	filePath := fs.String("file", "metadata.json", "путь к JSON")
	prefix := fs.String("prefix", "", "показывать только пути с этим префиксом")
	fs.Parse(args)

	data, err := os.ReadFile(*filePath)
	if err != nil {
		die(err)
	}

	li := newLineIndex(data)

	type nullHit struct {
		line, col int
		path      string
	}
	var hits []nullHit

	root := gjson.ParseBytes(data)
	walkGJSON(root, "@this", func(path string, v gjson.Result) {
		if v.Type != gjson.Null {
			return
		}
		if *prefix != "" && !strings.HasPrefix(path, *prefix) {
			return
		}
		line, col := li.locate(int64(v.Index))
		hits = append(hits, nullHit{line, col, path})
	})

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].line != hits[j].line {
			return hits[i].line < hits[j].line
		}
		return hits[i].col < hits[j].col
	})

	for _, h := range hits {
		fmt.Printf("%d:%d  %s = null\n", h.line, h.col, h.path)
	}
	fmt.Fprintf(os.Stderr, "\nnull values: %d\n", len(hits))
}

// walkGJSON рекурсивно обходит Result, вызывая fn для каждого узла.
// path начинается с "@this" для корня.
func walkGJSON(v gjson.Result, path string, fn func(path string, v gjson.Result)) {
	fn(path, v)
	switch {
	case v.IsObject():
		v.ForEach(func(k, sub gjson.Result) bool {
			child := path
			if child == "@this" {
				child = k.String()
			} else {
				child = child + "." + k.String()
			}
			walkGJSON(sub, child, fn)
			return true
		})
	case v.IsArray():
		v.ForEach(func(_, sub gjson.Result) bool {
			walkGJSON(sub, path+".#", fn)
			return true
		})
	}
}

// ---------------------------------------------------------------------------
// индексация строк
// ---------------------------------------------------------------------------

type lineIndex struct{ starts []int64 }

func newLineIndex(data []byte) *lineIndex {
	starts := []int64{0}
	for i, b := range data {
		if b == '\n' {
			starts = append(starts, int64(i+1))
		}
	}
	return &lineIndex{starts: starts}
}

// locate возвращает 1-based line и col для байтового offset.
func (li *lineIndex) locate(offset int64) (line, col int) {
	if offset < 0 {
		return 1, 1
	}
	lo, hi := 0, len(li.starts)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if li.starts[mid] <= offset {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo + 1, int(offset-li.starts[lo]) + 1
}

// ---------------------------------------------------------------------------
// декодер с указанием строки/колонки (для subcommand "check")
// ---------------------------------------------------------------------------

type JSONDecodeError struct {
	Err    error
	Offset int64
	Line   int
	Column int
}

func (e *JSONDecodeError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf(
			"JSON schema mismatch at line %d, column %d (offset %d): %v",
			e.Line, e.Column, e.Offset, e.Err,
		)
	}
	return e.Err.Error()
}

func (e *JSONDecodeError) Unwrap() error { return e.Err }

func DecodeJSONBytes(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		return wrapJSONError(data, err, dec.InputOffset())
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return &JSONDecodeError{
				Err:    errors.New("unexpected extra JSON value"),
				Offset: dec.InputOffset(),
			}
		}
		return wrapJSONError(data, err, dec.InputOffset())
	}
	return nil
}

func wrapJSONError(data []byte, err error, fallbackOffset int64) error {
	offset := fallbackOffset

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		offset = syntaxErr.Offset
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		offset = typeErr.Offset
	}

	line, col := newLineIndex(data).locate(offset)
	return &JSONDecodeError{Err: err, Offset: offset, Line: line, Column: col}
}

type requiredIssue struct {
	line, col int
	path      string
	field     string
}
