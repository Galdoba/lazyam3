#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C

usage() {
  echo "Usage: $0 <path-to-repository>" >&2
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

TARGET="$1"

if [[ ! -d "$TARGET" ]]; then
  echo "Error: '$TARGET' is not a directory." >&2
  exit 1
fi

REPO_ABS="$(cd "$TARGET" && pwd -P)"

# ---------------------------------------------------------------------------
# Проверка, что путь относится к Git-репозиторию.
# Если git недоступен, допускаем наличие .git в каталоге.
# ---------------------------------------------------------------------------
if command -v git >/dev/null 2>&1; then
  if ! git -C "$REPO_ABS" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    if [[ ! -e "$REPO_ABS/.git" ]]; then
      echo "Error: '$REPO_ABS' is not inside a Git repository." >&2
      exit 1
    fi
  fi
else
  if [[ ! -e "$REPO_ABS/.git" ]]; then
    echo "Error: git command is not available and '$REPO_ABS'/.git was not found." >&2
    exit 1
  fi
fi

for tool in awk find sort sed; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Error: required tool '$tool' is not available." >&2
    exit 1
  fi
done

TMP_WORK="$(mktemp -d "${TMPDIR:-/tmp}/go-repo-scan.XXXXXX")"
trap 'rm -rf "$TMP_WORK"' EXIT

EDGES_FILE="$TMP_WORK/edges.tsv"
: > "$EDGES_FILE"

shopt -s nullglob
shopt -u dotglob

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

rel_path() {
  local p="$1"
  local rel="${p#"$REPO_ABS"}"
  rel="${rel#/}"
  if [[ -z "$rel" ]]; then
    rel="."
  fi
  printf '%s' "$rel"
}

abs_from_rel() {
  if [[ "$1" == "." ]]; then
    printf '%s' "$REPO_ABS"
  else
    printf '%s/%s' "$REPO_ABS" "$1"
  fi
}

# Печатает дерево каталога без скрытых файлов и каталогов.
print_tree() {
  local dir="$1"
  local prefix="$2"
  local entries=()
  local p

  # Сначала каталоги, затем файлы. Скрытые объекты не попадают в glob.
  for p in "$dir"/*/; do
    p="${p%/}"
    # Не рекурсируем в каталоги-символические ссылки, чтобы избежать циклов.
    if [[ -d "$p" && ! -L "$p" ]]; then
      entries+=("d|$p")
    fi
  done

  for p in "$dir"/*; do
    if [[ -f "$p" || ( -L "$p" && ! -d "$p" ) ]]; then
      entries+=("f|$p")
    fi
  done

  local count="${#entries[@]}"
  local i=0
  local entry type path name connector child_prefix

  for entry in "${entries[@]}"; do
    type="${entry%%|*}"
    path="${entry#*|}"
    name="${path##*/}"

    i=$((i + 1))

    if (( i == count )); then
      connector='└── '
      child_prefix="$prefix    "
    else
      connector='├── '
      child_prefix="$prefix│   "
    fi

    printf '%s%s%s\n' "$prefix" "$connector" "$name"

    if [[ "$type" == "d" ]]; then
      print_tree "$path" "$child_prefix"
    fi
  done
}

# Загружает Go-файлы каталога в глобальный массив GO_FILES.
# mode=api: предпочитает файлы, не являющиеся *_test.go.
# Если таких нет, использует все *.go.
load_go_files() {
  local dir="$1"
  local mode="${2:-api}"
  GO_FILES=()

  if [[ "$mode" == "api" ]]; then
    mapfile -d '' GO_FILES < <(
      find "$dir" -mindepth 1 -maxdepth 1 -type f ! -name '.*' -name '*.go' ! -name '*_test.go' -print0 2>/dev/null | LC_ALL=C sort -z
    )

    if (( ${#GO_FILES[@]} == 0 )); then
      mapfile -d '' GO_FILES < <(
        find "$dir" -mindepth 1 -maxdepth 1 -type f ! -name '.*' -name '*.go' -print0 2>/dev/null | LC_ALL=C sort -z
      )
    fi
  else
    mapfile -d '' GO_FILES < <(
      find "$dir" -mindepth 1 -maxdepth 1 -type f ! -name '.*' -name '*.go' -print0 2>/dev/null | LC_ALL=C sort -z
    )
  fi
}

get_package_name() {
  local f out
  for f in "$@"; do
    out="$(awk '/^[[:space:]]*package[[:space:]]+/ { print $2; exit }' "$f" 2>/dev/null || true)"
    if [[ -n "$out" ]]; then
      printf '%s' "$out"
      return 0
    fi
  done
  return 0
}

get_package_comment() {
  local f out
  for f in "$@"; do
    out="$(
      awk '
        function trim(s) {
          sub(/^[[:space:]]+/, "", s)
          sub(/[[:space:]]+$/, "", s)
          return s
        }

        BEGIN {
          inblock = 0
          buf = ""
        }

        {
          line = $0

          if (inblock) {
            c = line
            if (c ~ /\*\//) {
              sub(/\*\/.*/, "", c)
              inblock = 0
            }
            sub(/^[[:space:]]*\*+[[:space:]]?/, "", c)
            c = trim(c)
            if (c != "") buf = buf c "\n"
            next
          }

          if (line ~ /^[[:space:]]*\/\//) {
            c = line
            sub(/^[[:space:]]*\/\//[[:space:]]?/, "", c)
            c = trim(c)
            if (c != "") buf = buf c "\n"
            next
          }

          if (line ~ /^[[:space:]]*\/\*/) {
            c = line
            sub(/^[[:space:]]*\/\*/, "", c)
            if (c ~ /\*\//) {
              sub(/\*\/.*/, "", c)
              inblock = 0
            } else {
              inblock = 1
            }
            sub(/^[[:space:]]*\*+[[:space:]]?/, "", c)
            c = trim(c)
            if (c != "") buf = buf c "\n"
            next
          }

          if (line ~ /^[[:space:]]*package[[:space:]]+/) {
            if (buf != "") printf "%s", buf
            exit
          }

          if (line ~ /^[[:space:]]*$/) {
            buf = ""
            next
          }

          buf = ""
        }
      ' "$f" 2>/dev/null || true
    )"

    if [[ -n "$out" ]]; then
      printf '%s' "$out"
      return 0
    fi
  done

  return 0
}

get_exported_declarations() {
  if (( $# == 0 )); then
    return 0
  fi

  awk '
    function trim(s) {
      sub(/^[[:space:]]+/, "", s)
      sub(/[[:space:]]+$/, "", s)
      return s
    }

    function clean(s) {
      sub(/[[:space:]]*\/\/.*$/, "", s)
      sub(/[[:space:]]*\{.*$/, "", s)
      return trim(s)
    }

    BEGIN {
      depth = 0
      constblock = 0
      typeblock = 0
      incomment = 0
    }

    {
      line = $0

      if (incomment) {
        if (line ~ /\*\//) incomment = 0
        next
      }

      if (line ~ /^[[:space:]]*\/\//) next

      if (line ~ /^[[:space:]]*\/\*/) {
        if (line !~ /\*\//) incomment = 1
        next
      }

      if (depth == 0) {
        if (constblock) {
          if (line ~ /^[[:space:]]*\)/) {
            constblock = 0
          } else {
            t = trim(line)
            if (t != "" && t !~ /^\/\// && t ~ /^[A-Z][A-Za-z0-9_]*([[:space:]=]|$)/) {
              print clean(t)
            }
          }
        } else if (typeblock) {
          if (line ~ /^[[:space:]]*\)/) {
            typeblock = 0
          } else {
            t = trim(line)
            if (t != "" && t !~ /^\/\// && t ~ /^[A-Z][A-Za-z0-9_]*/ && t ~ /struct[[:space:]]*(\{|[[:space:]]|$)/) {
              print clean(t)
            }
          }
        } else {
          if (line ~ /^[[:space:]]*const[[:space:]]*\(/) {
            constblock = 1
          } else if (line ~ /^[[:space:]]*const[[:space:]]+[A-Z]/) {
            print clean(line)
          } else if (line ~ /^[[:space:]]*type[[:space:]]*\(/) {
            typeblock = 1
          } else if (line ~ /^[[:space:]]*type[[:space:]]+[A-Z]/ && line ~ /struct[[:space:]]*(\{|[[:space:]]|$)/) {
            print clean(line)
          } else if (line ~ /^[[:space:]]*func[[:space:]]*\(/) {
            rest = line
            sub(/^[[:space:]]*func[[:space:]]*\([^)]*\)[[:space:]]*/, "", rest)
            if (rest ~ /^[A-Z]/) print clean(line)
          } else if (line ~ /^[[:space:]]*func[[:space:]]+[A-Z]/) {
            print clean(line)
          }
        }
      }

      tmp = line
      sub(/\/\/.*$/, "", tmp)
      openb = gsub(/\{/, "", tmp)
      closeb = gsub(/\}/, "", tmp)
      depth += openb - closeb
      if (depth < 0) depth = 0
    }
  ' "$@" 2>/dev/null | awk 'NF && !seen[$0]++'
}

get_imports() {
  if (( $# == 0 )); then
    return 0
  fi

  awk '
    BEGIN {
      incomment = 0
      importblock = 0
    }

    {
      line = $0

      if (incomment) {
        if (line ~ /\*\//) incomment = 0
        next
      }

      if (line ~ /^[[:space:]]*\/\//) next

      if (line ~ /^[[:space:]]*\/\*/) {
        if (line !~ /\*\//) incomment = 1
        next
      }

      if (importblock) {
        if (line ~ /^[[:space:]]*\)/) {
          importblock = 0
          next
        }

        s = line
        while (match(s, /"[^"]+"/)) {
          print substr(s, RSTART + 1, RLENGTH - 2)
          s = substr(s, RSTART + RLENGTH)
        }
        next
      }

      if (line ~ /^[[:space:]]*import[[:space:]]*\(/) {
        importblock = 1
        s = line
        sub(/^[[:space:]]*import[[:space:]]*\(/, "", s)

        if (s ~ /\)/) {
          importblock = 0
          sub(/\).*/, "", s)
        }

        while (match(s, /"[^"]+"/)) {
          print substr(s, RSTART + 1, RLENGTH - 2)
          s = substr(s, RSTART + RLENGTH)
        }
        next
      }

      if (line ~ /^[[:space:]]*import[[:space:]]+/) {
        s = line
        while (match(s, /"[^"]+"/)) {
          print substr(s, RSTART + 1, RLENGTH - 2)
          s = substr(s, RSTART + RLENGTH)
        }
      }
    }
  ' "$@" 2>/dev/null | LC_ALL=C sort -u
}

# ---------------------------------------------------------------------------
# Поиск go.mod и Go-файлов
# ---------------------------------------------------------------------------

go_mod_file=""
if [[ -f "$REPO_ABS/go.mod" ]]; then
  go_mod_file="$REPO_ABS/go.mod"
else
  while IFS= read -r -d '' f; do
    go_mod_file="$f"
    break
  done < <(
    find "$REPO_ABS" -mindepth 1 \( -name '.*' -prune \) -o \( -type f -name 'go.mod' -print0 \) 2>/dev/null | LC_ALL=C sort -z
  )
fi

go_file=""
while IFS= read -r -d '' f; do
  go_file="$f"
  break
done < <(
  find "$REPO_ABS" -mindepth 1 \( -name '.*' -prune \) -o \( -type f -name '*.go' -print0 \) 2>/dev/null | LC_ALL=C sort -z
)

if [[ -z "$go_mod_file" && -z "$go_file" ]]; then
  echo "Error: '$REPO_ABS' does not look like a Go repository: no go.mod or .go files found." >&2
  exit 1
fi

module_path=""
module_root_abs=""
module_root_rel=""

if [[ -n "$go_mod_file" ]]; then
  module_root_abs="$(cd "$(dirname "$go_mod_file")" && pwd -P)"
  module_path="$(
    awk '
      /^[[:space:]]*module[[:space:]]+/ {
        line = $0
        sub(/^[[:space:]]*module[[:space:]]+/, "", line)
        sub(/[[:space:]]*\/\/.*$/, "", line)
        sub(/[[:space:]]+$/, "", line)
        gsub(/^"|"$/, "", line)
        print line
        exit
      }
    ' "$go_mod_file" 2>/dev/null || true
  )"

  if [[ -n "$module_path" ]]; then
    module_root_rel="$(rel_path "$module_root_abs")"
  fi
fi

# ---------------------------------------------------------------------------
# Поиск Go-пакетов: каталоги, содержащие хотя бы один .go файл.
# ---------------------------------------------------------------------------

declare -A pkg_seen=()
pkg_rels=()

while IFS= read -r -d '' f; do
  d="${f%/*}"
  rel="$(rel_path "$d")"

  if [[ -z "${pkg_seen[$rel]:-}" ]]; then
    pkg_seen["$rel"]=1
    pkg_rels+=("$rel")
  fi
done < <(
  find "$REPO_ABS" -mindepth 1 \( -name '.*' -prune \) -o \( -type f -name '*.go' -print0 \) 2>/dev/null | LC_ALL=C sort -z
)

if (( ${#pkg_rels[@]} > 0 )); then
  mapfile -d '' pkg_rels < <(printf '%s\0' "${pkg_rels[@]}" | LC_ALL=C sort -z)
fi

# ---------------------------------------------------------------------------
# Соответствие импортных путей локальным пакетам, если известен модуль.
# ---------------------------------------------------------------------------

declare -A import_to_pkg_rel=()

if [[ -n "$module_path" && ${#pkg_rels[@]} -gt 0 ]]; then
  for rel in "${pkg_rels[@]}"; do
    abs="$(abs_from_rel "$rel")"
    rel_to_mod=""

    if [[ "$abs" == "$module_root_abs" ]]; then
      rel_to_mod="."
    elif [[ "$abs" == "$module_root_abs"/* ]]; then
      rel_to_mod="${abs#"$module_root_abs"/}"
    else
      continue
    fi

    if [[ "$rel_to_mod" == "." ]]; then
      import_path="$module_path"
    else
      import_path="$module_path/$rel_to_mod"
    fi

    import_to_pkg_rel["$import_path"]="$rel"
  done
fi

declare -A node_id=()
idx=0
if (( ${#pkg_rels[@]} > 0 )); then
  for rel in "${pkg_rels[@]}"; do
    node_id["$rel"]="p$idx"
    idx=$((idx + 1))
  done
fi

# ---------------------------------------------------------------------------
# Вывод
# ---------------------------------------------------------------------------

echo "=== Go repository scan ==="
echo "Path: $REPO_ABS"
if [[ -n "$module_path" ]]; then
  echo "Go module: $module_path"
  echo "Module root: $module_root_rel"
else
  echo "Go module: not found"
fi
echo

echo "=== Repository tree ==="
root_name="${REPO_ABS##*/}"
if [[ -z "$root_name" ]]; then
  root_name="/"
fi
printf '%s\n' "$root_name"
print_tree "$REPO_ABS" ""
echo

echo "=== Packages ==="

if (( ${#pkg_rels[@]} == 0 )); then
  echo "No Go packages found."
else
  for rel in "${pkg_rels[@]}"; do
    abs="$(abs_from_rel "$rel")"
    load_go_files "$abs" api

    echo "Package: $rel"

    if (( ${#GO_FILES[@]} == 0 )); then
      echo "  No readable .go files."
      echo
      continue
    fi

    if [[ -n "$module_path" ]]; then
      import_path=""
      if [[ "$abs" == "$module_root_abs" ]]; then
        import_path="$module_path"
      elif [[ "$abs" == "$module_root_abs"/* ]]; then
        import_path="$module_path/${abs#"$module_root_abs"/}"
      fi

      if [[ -n "$import_path" ]]; then
        echo "Import path: $import_path"
      fi
    fi

    pkg_name="$(get_package_name "${GO_FILES[@]}")"
    if [[ -n "$pkg_name" ]]; then
      echo "Package name: $pkg_name"
    fi

    pkg_comment="$(get_package_comment "${GO_FILES[@]}")"
    if [[ -n "$pkg_comment" ]]; then
      echo "Comment:"
      printf '%s\n' "$pkg_comment" | sed 's/^/  /'
    else
      echo "Comment: (none)"
    fi

    decls="$(get_exported_declarations "${GO_FILES[@]}")"
    echo "Exported declarations:"
    if [[ -n "$decls" ]]; then
      printf '%s\n' "$decls" | sed 's/^/  /'
    else
      echo "  (none)"
    fi

    echo
  done
fi

echo "=== Dependencies ==="

if (( ${#pkg_rels[@]} == 0 )); then
  echo "(none)"
else
  for rel in "${pkg_rels[@]}"; do
    abs="$(abs_from_rel "$rel")"
    load_go_files "$abs" api

    if (( ${#GO_FILES[@]} == 0 )); then
      imports=""
    else
      imports="$(get_imports "${GO_FILES[@]}")"
    fi

    echo "Package $rel imports:"
    if [[ -n "$imports" ]]; then
      printf '%s\n' "$imports" | sed 's/^/  /'
    else
      echo "  (none)"
    fi

    if [[ -n "$module_path" && -n "$imports" ]]; then
      while IFS= read -r imp; do
        [[ -z "$imp" ]] && continue
        target="${import_to_pkg_rel[$imp]:-}"
        if [[ -n "$target" && "$target" != "$rel" ]]; then
          printf '%s\t%s\n' "$rel" "$target" >> "$EDGES_FILE"
        fi
      done <<< "$imports"
    fi
  done
fi

if [[ -s "$EDGES_FILE" ]]; then
  LC_ALL=C sort -u "$EDGES_FILE" -o "$EDGES_FILE"
fi

echo

echo "=== Mermaid dependency graph ==="

if (( ${#pkg_rels[@]} == 0 )); then
  echo "(no packages)"
elif [[ -z "$module_path" ]]; then
  echo "Module path not found; local import mapping is unavailable. See textual dependencies above."
else
  echo 'flowchart TD'

  if (( ${#pkg_rels[@]} > 0 )); then
    for rel in "${pkg_rels[@]}"; do
      id="${node_id[$rel]}"
      label="${rel//\"/}"
      label="${label//\\/}"
      printf '  %s["%s"]\n' "$id" "$label"
    done
  fi

  if [[ -s "$EDGES_FILE" ]]; then
    while IFS=$'\t' read -r src dst; do
      [[ -z "$src" || -z "$dst" ]] && continue
      if [[ -n "${node_id[$src]:-}" && -n "${node_id[$dst]:-}" ]]; then
        printf '  %s --> %s\n' "${node_id[$src]}" "${node_id[$dst]}"
      fi
    done < "$EDGES_FILE"
  else
    echo "  %% No local dependencies found."
  fi
fi

exit 0
