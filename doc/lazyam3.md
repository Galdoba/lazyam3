---
updated_at: 2026-04-29T10:52:21.403+10:00
tags:
  - project
  - tool
aliases:
---
## Logic

1. Сканируем папку входящих, находим файлы и афдишники
2. Проверяем файл на готовность
	- Что мы считаем готовым файлом?
		- можем открыть его на запись.
3. Собираем технические данные
	- получаем ffprobe.RawData
4. Собираем метаданные контента
	- получаем metadata.Found
5. Собираем скрипт обработки
	- Что нас интересует?
		- Количество и координаты субтитров/видео/аудиодорожек
			- под координатами понимаем код стрима для отсортированного по алфавиту списка файлов (`[0:a:1]`, `[1:s:0]`)
		- для видео
			- Размер
			- Интерлейс
		- для аудио:
			- Язык
			- Формат (layout)
		- для субтитров
6. уходим в начало

## Domain Modules

### Injest (3 - точно, 1 и 2 - не уверен)
Работает с метаданными физических файлов, их путями и ffmpeg-структурой.



### Metadata (4)
Работает с метаданными медиапродукта - поставляется Амедией.

Определения:
id - айдишник идет на запись. может попасть на сериал, сезон, эпизод сериала, фильм и эпизод фильма.
Kind - тип объекта в базе имеющий `id`. Варианты:
```go
type Kind string

const (
	KindMovie         Kind = "movie"
	KindMovieEpisode  Kind = "movie.episode"
	KindSeries        Kind = "series"
	KindSeason        Kind = "series.season"
	KindSeriesEpisode Kind = "series.season.episode"
)

```


Что сделали:
- распарсили `metadata.json`
- ищем по `id`

### Script (5)
Пишем скрипт по логике "Передали конфиг (DTO) - получили текст файла"

---

# Дерево репозитория

```
.
├── cmd/
│   └── amedia-pipeline/
│       └── main.go                 # сборка графа: config → logger → infra → use cases → cli.App
│
├── internal/
│   │
│   ├── presentation/               # СЛОЙ ПРЕДСТАВЛЕНИЯ (вход)
│   │   └── cli/                    # urfave/cli/v3. Одна команда — один файл.
│   │       ├── app.go              # *cli.Command, глобальные флаги (--config, --log-level), регистрация подкоманд
│   │       └── run.go              # единственная команда: флаги → RunInput → app/pipeline.Run → RunOutput
│   │
│   ├── app/                        # ЯДРО (Application): оркестрация, работает на портах и DTO
│   │   ├── pipeline/
│   │   │   ├── run.go              # оркестратор: ingest.Scan → metadata.Resolve → script.Build → renderer.Render
│   │   │   └── dto.go              # RunInput / RunOutput — DTO на границе presentation ↔ app
│   │   ├── ingest/
│   │   │   └── scan.go             # use case: дёрнуть FileScanner, прогнать readiness-фильтр, собрать tech
│   │   └── script/
│   │       └── build.go            # use case: принять доменные входы, вызвать domain/script.Build
│   │
│   ├── domain/                     # ДОМЕН: модели и правила. Никакого os/net/yaml. Никаких DTO.
│   │   ├── ingest/                 # язык физических файлов и потоков
│   │   │   ├── file.go             # RawFile, ReadyFile, Sidecar
│   │   │   ├── stream.go           # Stream, StreamKind, Layout, Language
│   │   │   ├── index.go            # построение координат [0:a:1], [1:s:0]
│   │   │   ├── readiness.go        # ReadinessPolicy (если правило стало нетривиальным; иначе пусто)
│   │   │   └── port.go             # порты: FileScanner, ReadinessChecker, TechnicalProbe
│   │   │
│   │   ├── metadata/               # язык медиапродукта Амедии
│   │   │   ├── kind.go             # Kind: movie, movie.episode, series, series.season, series.season.episode
│   │   │   ├── product.go          # Product, Season, Episode, EpisodeRef
│   │   │   └── port.go             # порт: ProductRepository
│   │   │
│   │   ├── script/                 # язык генерации bash-скрипта
│   │   │   ├── script.go           # Script, StreamSelection
│   │   │   ├── text.go             # type Text string — value object с телом скрипта
│   │   │   ├── config.go           # ScriptConfig — доменный вход
│   │   │   └── port.go             # порт: Renderer (принимает Destination + Text)
│   │   │
│   │   └── shared/                 # (опционально) общие value objects для доменов
│   │       └── taskid.go           # TaskID — если ingest и metadata используют один и тот же id
│   │
│   ├── infra/                      # РЕАЛИЗАЦИИ ПОРТОВ. Знает про внешний мир. Зависит от domain.
│   │   ├── config/
│   │   │   └── yaml.go             # загрузка YAML/ENV в типизированный Config
│   │   ├── logging/
│   │   │   └── slog.go             # slog-обёртка под лог-порт
│   │   ├── fs/                     # файловая система
│   │   │   ├── scanner.go          # реализует ingest.FileScanner (WalkDir, маски, sidecar)
│   │   │   ├── readiness_open.go   # реализует ingest.ReadinessChecker (O_WRONLY, вся правда про NFS/SMB)
│   │   │   ├── sidecar.go          # поиск sidecar-артефактов
│   │   │   └── script_writer.go    # реализует script.Renderer: пишет script.Text в Destination
│   │   ├── ffprobe/
│   │   │   ├── client.go           # запуск ffprobe -show_streams -print_format json → RawData
│   │   │   └── mapper.go           # RawData → доменные ingest.Stream
│   │   ├── amedia/
│   │   │   ├── http_client.go      # HTTP-клиент к API Амедии
│   │   │   ├── auth.go             # токены, mTLS
│   │   │   └── mapper.go           # JSON Амедии → metadata.Product
│   │   └── persistence/
│   │       └── postgres/
│   │           ├── product_repo.go # реализует metadata.ProductRepository
│   │           └── migrations/
│   │
│   └── platform/                   # STATELESS-СЕРВИСЫ. Не реализуют портов домена. Ничего не хранят.
│       ├── retry/
│       │   └── retry.go            # backoff, jitter — используется в amedia/ и ffprobe/
│       ├── httpx/
│       │   └── client.go           # http.Client с таймаутами, трейсингом, дефолтными заголовками
│       └── clock/
│           └── clock.go            # абстракция времени для тестов (readiness-таймауты, retry)
│
├── pkg/                            # только то, что реально переиспользуется снаружи. По умолчанию пусто.
│   └── ffmpeg/
│       └── coordinate.go           # формат [0:a:1] — если правда нужен вне репо
│
├── api/                            # контракты внешних систем (не код)
│   ├── amedia/
│   │   └── openapi.yaml
│   └── ffprobe/
│       └── schema.json
│
├── configs/
│   ├── config.yaml
│   └── config.local.yaml
│
├── testdata/
│   ├── incoming/                   # примеры входящей папки с файлами + sidecars
│   └── ffprobe/                    # фикстуры RawData для mapper-тестов
│
├── go.mod
└── README.md
```

## Пометки по связям

**Направление зависимостей**
```
presentation → app → domain
infra        → domain (реализует порты)
platform     → никого из домена
cmd          → всё (единственное место склейки)
```

**Кто кого не знает**
- `domain/ingest` не знает `domain/metadata` и `domain/script`.
- `domain/metadata` не знает `domain/ingest` и `domain/script`.
- `domain/script` не знает `domain/ingest` и `domain/metadata`.
- Все три знают только `domain/shared` (если он есть).
- `presentation/cli` знает только про `app/pipeline.Run` и его DTO.
- `infra/*` знает только про порты соответствующего домена.

**Где DTO, а где доменные типы**

| Граница | Что передаётся |
|---|---|
| presentation ↔ app | DTO (`RunInput`, `RunOutput`) |
| app ↔ domain | доменные типы (`ReadyFile`, `Product`, `ScriptConfig`, `Text`) |
| app ↔ infra | доменные типы через порты |
| domain → domain | только через оркестратор, замапленные в термины получателя |

**Где `os`, `net`, `database/sql`, YAML**
Только в `internal/infra/*` и `internal/platform/*`.

**Где циклы, вызовы портов, фильтрация**
Только в `internal/app/*`.

**Где чистые правила и модели**
Только в `internal/domain/*`.

**Проверка на вшивость**

| Вопрос | Ответ |
|---|---|
| `os.OpenFile(O_WRONLY)` | `infra/fs/readiness_open.go` |
| `for _, f := range files` | `app/ingest/scan.go` |
| `type Kind string` | `domain/metadata/kind.go` |
| `[0:a:1]` | `domain/ingest/index.go` |
| `ffprobe -show_streams` | `infra/ffprobe/client.go` |
| `--input-dir` | `presentation/cli/run.go` |
| `pipeline = scan → build → render` | `app/pipeline/run.go` |
| `Renderer` интерфейс | `domain/script/port.go` |
| запись `Text` в файл | `infra/fs/script_writer.go` |
| retry с backoff | `platform/retry/retry.go` |
| склейка всего | `cmd/amedia-pipeline/main.go` |

Если что-то не влезает в таблицу — это либо новый домен (тогда своя папка в `domain/`, `app/`, `infra/`), либо platform-сервис, либо сигнал, что слой выбран неверно.