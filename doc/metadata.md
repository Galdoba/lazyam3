# Metadata Domain

The metadata domain handles the product catalog from the counterparty: resolving a product ID (series, season, episode, movie, or movie episode) into a flat, pipeline-ready `Product`.

## Overview

```
metadata.json → [parse tree] → FindByID(id) → [prune tree] → Found → [flatten] → Product
```

The domain defines the language of the catalog (`Product`, `Kind`) and the lookup contract (`ProductRepository`). The JSON storage format, tree search, and file loading live in infrastructure — they are storage concerns, not domain concerns.

## Domain Layer

### Types

**`Product`** — flat metadata record. One shape for all catalog kinds, ready for the pipeline.

```go
type Product struct {
    ID            int64    // Catalog ID (matches the ID in filenames)
    Kind          Kind     // What kind of catalog object this is
    Title         string
    SeriesTitle   string   // Parent series/movie title (own title for series/movie kinds); set by FoundToProduct — used by script naming
    ParentID      *int64   // Season ID (for season/episode), series ID (for season), movie ID (for movie episode)
    SeasonNumber  *int     // series.season, series.season.episode
    EpisodeNumber *int     // movie.episode, series.season.episode
    SeasonEp      string   // "s01e01" style code for series episodes
    Description   string
    Genres        []string // Inherited from parent series/movie for children
    Countries     []string // Inherited from parent series/movie for children
    KinopoiskID   string   // Inherited from parent series/movie for children
    IMDBID        string   // Inherited from parent series/movie for children
}
```

**`Kind`** — classifies the catalog object the ID refers to.

```go
const (
    KindMovie         Kind = "movie"
    KindMovieEpisode  Kind = "movie.episode"
    KindSeries        Kind = "series"
    KindSeason        Kind = "series.season"
    KindSeriesEpisode Kind = "series.season.episode"
)
```

### Pure Functions

None. The domain is types + port only; tree search walks the JSON storage structure and lives in infrastructure.

## Implementation Layer

### `ProductRepository` Interface (port)

```go
type ProductRepository interface {
    FindByID(id int64) (*Product, error)
}
```

Defines the contract for resolving catalog IDs to domain `Product`s.

### Infrastructure (`internal/infra/metadata/`)

**JSON types** (storage format): `Root` → `[]SeriesItem` / `[]MoviesItem` → `[]SeasonsItem` → `[]EpisodesItem`, plus `PersonsItem`.

**Lookup flow:**

```go
LoadRoot(path) → *Root                    // read + unmarshal metadata.json
Root.FindByID(id, d) → (*Found, bool)     // tree search, returns pruned self-contained copy
FoundToProduct(*Found) *Product           // flatten pruned tree to domain DTO
```

**`Root.FindByID(id int64, d Descendants) (*Found, bool)`**

Search order: movies first (movie → its episodes), then series (series → seasons → episodes). The result is a `Found{Kind, Data Root}` — a pruned copy of the tree with the target at its proper place:

- `WithoutDescendants` — only the target path (e.g. a season without its episodes)
- `WithDescendants` — the target keeps its children (e.g. a season with its episodes)

All returned data is cloned — mutations of the result never touch the repository's tree.

**`FoundToProduct(*Found) *Product`**

| Kind | Title/Description from | SeriesTitle | ParentID | SeasonNumber | EpisodeNumber | SeasonEp | Genres/IDs |
|---|---|---|---|---|---|---|---|
| `series` | series | own title | — | — | — | — | own |
| `series.season` | season | series | series | ✓ | — | — | inherited from series |
| `series.season.episode` | episode | series | season | ✓ | ✓ | `s%02de%02d` | inherited from series |
| `movie` | movie | own title | — | — | — | — | own |
| `movie.episode` | episode | movie | movie | — | ✓ | — | inherited from movie |

**`Repository`** — implements the port, thread-safe (`sync.RWMutex`):

```go
func New(path string) (*Repository, error)          // fail-fast: read/parse/empty errors
func (r *Repository) FindByID(id int64) (*mdomain.Product, error)  // always WithDescendants
```

Errors: `read metadata file: …`, `parse metadata JSON: …`, `empty metadata: no series or movies found`, `product <id> not found`.

### CLI Integration

```bash
lazyam3 process \
  --source-root /path/to/media \
  --metadata-path /path/to/metadata.json \
  --once
```

The `--metadata-path` flag falls back to the `paths.metadata_path` config field. The repository is created before the pipeline starts — a missing, broken, or empty catalog fails the command immediately. The result is passed into `pipeline.Options.Metadata`; `Pipeline.Run` queries it during `RunOnce`.

## Test Data

`testdata/metadata.json` — real catalog snapshot (13.6 MB): **690 series, 981 movies**.

Known IDs used by the integration test (`TestRepository_FindByID_RealFile`):

| ID | Kind | Title | Notes |
|---|---|---|---|
| 11120 | `series` | Белая королева | season 11393 → episode 12061 |
| 11393 | `series.season` | Сезон 1 | ParentID = 11120 |
| 12061 | `series.season.episode` | Любовь короля | ParentID = 11393, SeasonEp = `s01e01` |
| 9257 | `movie` | Поймать Мэри | no episodes (empty-children case) |
| 10100 | `movie.episode` | Фил Спектор | ParentID = 9261 |

Synthetic fixtures in tests cover pruning (`WithoutDescendants`), clone independence, nil/empty roots, and not-found errors.

## File Structure

```
internal/domain/metadata/
├── types.go         # Product, Kind + constants
└── port.go          # ProductRepository interface

internal/infra/metadata/
├── types.go         # JSON types, LoadRoot, Root.FindByID, Found, FoundToProduct, clone helpers
├── types_test.go    # 16 test functions: tree search, pruning, flattening, clone independence
├── repository.go    # Repository implements ProductRepository (thread-safe)
└── repository_test.go  # synthetic fixtures + real-file integration test
```
