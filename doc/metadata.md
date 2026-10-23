# Bounded Context: Metadata

> **Status:** active  
> **Owner:** pipeline engineering  
> **Entropy level:** Low

## 1. Purpose

The metadata domain provides a catalog lookup service for resolving product IDs (series, season, episode, movie, or movie episode) into a flat, pipeline-ready `Product`. It loads a JSON catalog from the counterparty, searches a hierarchical tree structure, prunes the result to a self-contained subtree, and flattens that subtree into a single `Product` DTO.

**Out of scope:** Media file scanning (belongs to the *Ingest* domain), rendering script generation (belongs to the *Script* domain), file delivery or notification, and any user-facing interface. The domain knows nothing about *what* the media looks like on disk — only that a catalog entry exists for a given numeric ID.

## 2. Context & Environment

- **Users / Actors:** The automated pipeline is the sole actor. No human directly interacts with this domain; it is instantiated by the pipeline orchestrator (`app.Pipeline.RunOnce`) which loads the catalog once and passes it into `pipeline.Options.Metadata` for use during `RunOnce`.
- **Adjacent domains and systems:** 
  - **Ingest** — provides the `MediaUnit.ID` that the metadata domain resolves via `FindByID`.
  - **Script** — consumes the `Product` (via `callname`) for file naming derivation, using `SeriesTitle`, `SeasonNumber`, `EpisodeNumber`, and `Kind`.
  - **JSON metadata file** (`metadata.json`) — the sole data source; its schema is fixed by the counterparty and includes series, seasons, episodes, movies, and movie episodes.
  - **Filesystem (OS)** — the source of the JSON catalog file; accessed through infrastructure, never by the domain.
- **Key business rules:** A product ID resolves to exactly one catalog entry. The search order is movies first, then series. Descendants are pruned to the target's immediate path unless `WithDescendants` is requested. All returned data is cloned — mutations of the result never touch the repository's tree.

## 3. Boundaries & Interactions

- **Inbound interactions:** 
  - `New(path string) (*Repository, error)` — infrastructure constructor; reads, parses, and validates the JSON catalog. Fails fast on missing, broken, or empty files.
  - `Repository.FindByID(id int64) (*Product, error)` — the domain port. Acquires a read lock, searches the tree, prunes, and flattens.
  - `Root.FindByID(id int64, d Descendants) (*Found, bool)` — pure tree search on the loaded JSON structure.
  - `FoundToProduct(*Found) *Product` — pure flattening function that maps a pruned subtree to a domain DTO.
- **Outbound interactions:** 
  - Returns a `*Product` DTO to the application layer.
  - The domain itself makes no external calls; all I/O is delegated to infrastructure.
- **Domain Events:** None. The domain is synchronous and returns a DTO directly.
- **Dependencies:** 
  - No domain imports any other domain. `domain/metadata` is a leaf in the dependency graph.
  - `domain/script` depends on `domain/metadata` (via `mdomain.Product` in `ScriptConfig`).

## 4. Ubiquitous Language

| Term | Definition | Synonyms/Notes |
|------|------------|----------------|
| **Product** | Flat metadata record representing one catalog entry. | "catalog product", "product record" |
| **Kind** | Classification of the catalog object: movie, series, season, episode, or movie episode. | "catalog kind", "entity type" |
| **ProductRepository** | Domain port interface for resolving catalog IDs. | "repository", "catalog lookup" |
| **Repository** | Infrastructure implementation of ProductRepository (thread-safe, backed by JSON). | "catalog repo", "metadata repo" |
| **Root** | Top-level JSON container holding series and movies arrays. | "catalog root", "JSON root" |
| **Found** | A pruned subtree with an attached Kind, returned by tree search. | "lookup result", "pruned tree" |
| **Descendants** | Boolean flag controlling pruning depth: `WithDescendants` or `WithoutDescendants`. | "pruning mode", "expand children" |
| **SeriesTitle** | Root series/movie title, used by downstream domains for file naming. | "parent title", "show title" |
| **SeasonEp** | Formatted season-episode tag (e.g. `"s01e01"`), only for `series.season.episode`. | "S01E01", "season-episode code" |
| **ParentID** | ID of the parent catalog entry (season for episodes, movie for movie episodes, series for seasons). | "parent reference" |

## 5. Key Scenarios / Use Cases

### Scenario 1: Resolve a series by ID
- **Given** a metadata catalog loaded from `metadata.json`
- **When** `FindByID(11120)` is called
- **Then** a `Product` is returned with `ID=11120`, `Kind="series"`, `Title="Белая королева"`, `SeriesTitle="Белая королева"`
- **And** `Genres`, `Countries`, `KinopoiskID`, `IMDBID` are populated from the series entry

### Scenario 2: Resolve a season by ID
- **Given** a loaded catalog
- **When** `FindByID(11393)` is called (season 1 of "Белая королева")
- **Then** a `Product` is returned with `Kind="series.season"`, `SeasonNumber=1`
- **And** `ParentID` points to the series ID (11120), `SeriesTitle="Белая королева"`
- **And** `Genres` and `Countries` are inherited from the parent series

### Scenario 3: Resolve a series episode by ID
- **Given** a loaded catalog
- **When** `FindByID(12061)` is called (episode 1 of season 1)
- **Then** a `Product` is returned with `Kind="series.season.episode"`, `EpisodeNumber=1`
- **And** `SeasonEp` is formatted as `"s01e01"`
- **And** `ParentID` points to the season ID (11393), `SeriesTitle` is the series title
- **And** genres are inherited from the series, not the season

### Scenario 4: Resolve a movie by ID
- **Given** a loaded catalog
- **When** `FindByID(9257)` is called
- **Then** a `Product` is returned with `Kind="movie"`, `Title` set to the movie's title, `SeriesTitle` equal to the same title
- **And** `Genres`, `Countries`, `KinopoiskID`, `IMDBID` are populated from the movie entry

### Scenario 5: Resolve a multi-part movie episode by ID
- **Given** a loaded catalog
- **When** `FindByID(10100)` is called (Фил Спектор, a movie episode)
- **Then** a `Product` is returned with `Kind="movie.episode"`, `EpisodeNumber` set
- **And** `ParentID` points to the movie ID (9261), `SeriesTitle` is the movie title
- **And** genres are inherited from the movie

### Scenario 6: Not found — returns an error
- **Given** a loaded catalog
- **When** `FindByID(99999)` is called with an ID not present in the tree
- **Then** an error is returned with the message `"product 99999 not found"`

### Scenario 7: Empty catalog — fails at construction
- **Given** a JSON file with empty `series` and `movies` arrays
- **When** `New(path)` is called
- **Then** an error is returned with the message `"empty metadata: no series or movies found"`

### Scenario 8: Concurrent lookups are safe
- **Given** a loaded `Repository`
- **When** multiple goroutines call `FindByID` concurrently
- **Then** all calls complete successfully without data races (guaranteed by `sync.RWMutex`)

## 6. High-Level Data Model (Key Entities)

- **Product** — flat metadata record, one shape for all catalog kinds.
  - Key fields: `ID` (int64), `Kind` (Kind), `Title`, `SeriesTitle`, `ParentID` (*int64), `SeasonNumber` (*int), `EpisodeNumber` (*int), `SeasonEp` (string), `Description`, `Genres` ([]string), `Countries` ([]string), `KinopoiskID` (string), `IMDBID` (string).
  - Key invariants: `SeriesTitle` is always populated; `Genres`/`Countries`/`KinopoiskID`/`IMDBID` are inherited from the root entity (series or movie) for all descendants; `ParentID` is nil for top-level entries (series, movie).

- **Kind** — enum classifying catalog objects: `movie`, `movie.episode`, `series`, `series.season`, `series.season.episode`.

- **ProductRepository** — domain port: `FindByID(id int64) (*Product, error)`. Implemented by infrastructure.

- **Found** — infrastructure type representing a pruned subtree with attached Kind. Wraps a `Root` that has been pruned to contain only the target element and its required ancestry chain. Always returned with `WithDescendants` by the repository port.

- **Root** — infrastructure type: `Series []SeriesItem`, `Movies []MoviesItem`. The top-level JSON container loaded from disk.

**Relationships:**
- Product 1—1 Kind
- Product is resolved by ProductRepository
- Found 1—1 Root (pruned subtree)
- Root 1—N SeriesItem, 1—N MoviesItem
- SeriesItem 0—N SeasonsItem
- SeasonsItem 0—N EpisodesItem
- MoviesItem 0—N EpisodesItem

## 7. Non-Functional Requirements & Constraints

- **Authorization:** N/A — the domain runs as part of an automated pipeline with no user-facing access control.
- **Performance/Scale:** Designed for batch processing of hundreds to thousands of media units. Tree search is O(n) where n is the total number of catalog entries (series + movies + seasons + episodes). The real catalog (`testdata/metadata.json`) contains ~690 series and ~981 movies, making search fast enough for per-unit pipeline lookups. Cloning is O(m) where m is the pruned subtree size (typically small).
- **Data:** No persistence. The catalog is loaded once at startup and held in memory. The JSON schema is external (counterparty-defined) and may change; the domain assumes the schema is stable within a pipeline run. No PII is stored.
- **External constraints:** Depends on the counterparty's JSON metadata file format. The file is read once at construction via `os.ReadFile` and `json.Unmarshal`. The domain assumes ffprobe is not involved.

## 8. Acceptance Criteria

- [ ] `LoadRoot` correctly reads and parses `metadata.json`, validating that at least one series or movie exists.
- [ ] `Root.FindByID` searches movies first, then series, returning the correct Kind for all five catalog types.
- [ ] `FindByID` with `WithDescendants` returns the full subtree (series with all seasons+episodes, season with all episodes, movie with all episodes).
- [ ] `FindByID` with `WithoutDescendants` prunes the target's children to zero (series with 0 seasons, season with 0 episodes).
- [ ] `FoundToProduct` correctly maps all five Kind variants: sets `SeriesTitle`, `ParentID`, `SeasonNumber`, `EpisodeNumber`, `SeasonEp`, and inherits `Genres`/`Countries`/`KinopoiskID`/`IMDBID` from the root entity.
- [ ] `FoundToProduct` returns `nil` for `nil` input or empty data slices.
- [ ] All returned data is cloned — mutating a `Product` or `Found` never touches the repository's underlying tree (`TestFindByID_CloneIsIndependent`).
- [ ] `Repository.FindByID` is thread-safe under concurrent access (`sync.RWMutex`).
- [ ] `Repository.New` fails fast with descriptive errors: file read, JSON parse, empty catalog.
- [ ] `Repository.FindByID` returns `"product <id> not found"` for IDs absent from the tree.
- [ ] Integration tests pass with the real `testdata/metadata.json` (690 series, 981 movies): series (11120), season (11393), episode (12061), movie (9257), movie episode (10100).
- [ ] CLI `--metadata-path` flag correctly wires the repository into the pipeline; missing or broken catalog fails the command immediately.

## 9. Open Questions

- [ ] Should the domain expose a `Product.WithPrimaryFile() Stream` convenience method, or should the application layer derive this from `Streams` by selecting the first video stream? (Currently not implemented.)
- [ ] Should `PersonsItem` (directors, actors) be mapped to a field on `Product`, or is discarding them acceptable? Currently stored in JSON but never used.
- [ ] The `int64` → `int` conversion for `SeasonNumber` and `EpisodeNumber` uses a direct `int(i)` cast, which could overflow on 32-bit systems for very large values. Should this use `int(i)` with a bounds check or switch to `int64` throughout?
- [ ] Should `SeasonEp` be populated for `movie.episode` kinds as well, or is it intentionally exclusive to `series.season.episode`?
- [ ] The movies-first search order means a movie and series sharing the same ID would resolve to the movie. Is this ordering intentional, or should it be configurable?
