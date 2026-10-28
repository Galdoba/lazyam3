# Исследование логирования — Анализ и рекомендации

## 1. Текущая архитектура логирования

### 1.1 Базовый механизм

- **Пакет**: `internal/infra/logging`
- **Основа**: `log/slog` (Go 1.21+)
- **Обёртка**: `Logger` — фасад над `*slog.Logger` с явно определённой поверхностью (`Debug`, `Info`, `Warn`, `Error`, `Errorf`, `With`, `Group`)
- **Мульти-хендлер**: логи дублируются на терминал (stdout/stderr) и в файл
- **Форматы**: JSON (по умолчанию) и Text с кастомным шаблоном
- **Цветение**: ANSI-коды для уровней в текстовом режиме
- **Конфигурация**: TOML `[logger]` — уровень, путь, формат, цвет, шаблоны

### 1.2 Встроенный логгер

```
AppModules.Logger → modules.Logger.Group("pipeline") → pipeline.Pipeline
                            → modules.Logger.Group("command") → cmd_process
```

### 1.3 Где логирование УЖЕ РАБОТАЕТ

| Файл | Уровень | Сообщения |
|------|---------|-----------|
| `cmd_process.go` | command | создание пайплайна, ошибки валидации, события unit done/failed, финальный отчёт |
| `pipeline.go` | pipeline | `Warn` при сканировании, `Info` при сканировании/нахождении директорий, `Warn` при fallback метаданных |

### 1.4 Паттерн закомментированных логов

Все закомментированные вызовы используют формат `//log=level msg="..."` — это артефакты планирования логирования, которое так и не было реализовано. Формат не является частью инфраструктуры.

---

## 2. Полный каталог закомментированных вызовов

### 2.1 Infrastructure layer

#### `infra/logging/logger.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 26 | debug | logger created with level/format | **УДАЛИТЬ** — инициализация логгера, логировать нечего |
| 29 | error | failed to create logger | **УДАЛИТЬ** — ошибка возвращается как err, логгера ещё нет |
| 76 | info | logger closed | **ОСТАВИТЬ** — полезный сигнал завершения |
| 115 | error | no logging handlers created | **УДАЛИТЬ** — это паника/фатальная ошибка, не лог |

#### `infra/config/config.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 223 | info | loading config from path | **ОСТАВИТЬ** — полезно для отладки конфигурации |
| 226 | error | failed to create configmanager | **ОСТАВИТЬ** — ошибка инициализации, помогает найти проблему |
| 244 | warn | using default configuration | **ОСТАВИТЬ** — критически важно, пользователь должен знать |

#### `infra/metadata/types.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 107 | info | loaded metadata: N series, M movies | **ОСТАВИТЬ** — полезно, загружается один раз |
| 111 | error | failed to read metadata file | **УДАЛИТЬ** — ошибка возвращается как err |
| 116 | error | failed to parse metadata JSON | **УДАЛИТЬ** — ошибка возвращается как err |
| 120 | error | metadata file is empty | **УДАЛИТЬ** — ошибка возвращается как err |

#### `infra/metadata/repository.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 18 | info | metadata repository created from path | **ОСТАВИТЬ** — один раз при старте |
| 22 | error | failed to load metadata | **УДАЛИТЬ** — ошибка возвращается как err |
| 29 | debug | metadata lookup: id=N | **ОСТАВИТЬ** — полезно для отладки метаданных |
| 35 | error | metadata repository not initialized | **УДАЛИТЬ** — это баг, паника уместнее |
| 41 | warn | product N not found | **ОСТАВИТЬ** — частый сценарий, полезно |

#### `infra/fso/filescanner.go`

| Строка | Уровень | Сообщение | Статис |
|--------|---------|-----------|--------|
| 72 | info | scanning directory: path | **ОСТАВИТЬ** — один раз за сканирование |
| 79 | error | failed to resolve root path | **УДАЛИТЬ** — ошибка возвращается как err |
| 84 | error | failed to stat root | **УДАЛИТЬ** — ошибка возвращается как err |
| 88 | error | root is not a directory | **УДАЛИТЬ** — ошибка возвращается как err |
| 95 | error | failed to read root directory | **УДАЛИТЬ** — ошибка возвращается как err |

#### `infra/fso/ffprobe.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 34 | debug | ffprobe prober created | **УДАЛИТЬ** — инициализация, один раз |
| 43 | debug | probing file: path | **НЕ ВВОДИТЬ** — вызывается для КАЖДОГО файла, шум |
| 45 | error | ffprobe: stat failed | **УДАЛИТЬ** — ошибка возвращается как err |
| 55 | error | ffprobe failed | **УДАЛИТЬ** — ошибка возвращается как err |
| 61 | error | ffprobe returned invalid JSON | **УДАЛИТЬ** — ошибка возвращается как err |

#### `infra/script/script.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 30 | debug | script renderer created | **УДАЛИТЬ** — инициализация, один раз |
| 41 | info | writing script to: path | **ОСТАВИТЬ** — один раз за юнит, полезно |
| 45 | error | failed to create script directory | **УДАЛИТЬ** — ошибка возвращается как err |
| 49 | error | failed to write script | **УДАЛИТЬ** — ошибка возвращается как err |
| 53 | error | failed to set mode | **УДАЛИТЬ** — ошибка возвращается как err |

#### `infra/cache/cache.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 14 | debug | cache created | **УДАЛИТЬ** — инициализация, один раз |
| 30 | debug | cache set key=N | **НЕ ВВОДИТЬ** — вызывается часто, cache не используется активно |
| 38 | debug | cache delete key=N | **НЕ ВВОДИТЬ** — см. выше |
| 46 | debug | cache cleared | **УДАЛИТЬ** — редко используется |

#### `infra/persistence/jsonstore/jsonstore.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 25 | info | jsonstore created at path | **ОСТАВИТЬ** — один раз при старте |
| 28 | error | store path is empty | **УДАЛИТЬ** — ошибка возвращается как err |
| 36 | error | failed to load store | **УДАЛИТЬ** — ошибка возвращается как err |
| 48 | error | failed to read store file | **УДАЛИТЬ** — ошибка возвращается как err |
| 55 | error | failed to unmarshal store | **УДАЛИТЬ** — ошибка возвращается как err |
| 58 | debug | store loaded: N entries | **ОСТАВИТЬ** — один раз при старте |
| 72 | debug | store set key=N | **НЕ ВВОДИТЬ** — вызывается часто |
| 77 | error | store is closed | **УДАЛИТЬ** — ошибка возвращается как err |
| 110 | info | store saved: N entries | **ОСТАВИТЬ** — один раз при shutdown |
| 121 | error | failed to marshal store | **УДАЛИТЬ** — ошибка возвращается как err |
| 125 | error | failed to write store file | **УДАЛИТЬ** — ошибка возвращается как err |
| 133 | info | store closed and saved | **ОСТАВИТЬ** — сигнал завершения |

### 2.2 Application layer

#### `app/ingest/ingest.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 20 | debug | ingest created with ffprobe=N | **УДАЛИТЬ** — инициализация |
| 29 | info | ingesting directory: path | **ОСТАВИТЬ** — один раз за юнит, дублирует pipeline |
| 33 | error | failed to read directory | **УДАЛИТЬ** — ошибка возвращается как err |
| 51 | error | failed to probe file | **УДАЛИТЬ** — ошибка возвращается как err |
| 57 | error | failed to parse data for unit | **УДАЛИТЬ** — ошибка возвращается как err |

#### `app/script/build.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 12 | info | building rendering script | **ОСТАВИТЬ** — один раз за юнит |
| 15 | warn | context cancelled | **ОСТАВИТЬ** — сигнал прерывания |
| 20 | error | build script failed | **УДАЛИТЬ** — ошибка возвращается как err |
| 23 | info | writing rendered script to path | **ОСТАВИТЬ** — завершение этапа |

#### `app/pipeline/pipeline.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 43 | debug | pipeline created: source, destination | **УДАЛИТЬ** — инициализация |
| 70 | info | pipeline loop started | **ОСТАВИТЬ** — один раз при старте |
| 78 | info | pipeline stopped | **ОСТАВИТЬ** — завершение работы |
| 92 | warn | scan failed | **УЖЕ ЕСТЬ** — строка 91: `p.logger.Warn("scan failed"...` |
| 123 | error | scan failed | **ОСТАВИТЬ** — ошибка сканирования |
| 128 | info | scan started | **УДАЛИТЬ** — дублирует строку 127 |
| 135 | info | scan complete: N dirs | **ОСТАВИТЬ** — завершение сканирования |
| 142 | error | processing failed for unit | **ОСТАВИТЬ** — завершающий отчёт |
| 154 | info | processing unit: path | **УДАЛИТЬ** — дублирует строку 156 |
| 159 | error | ingest failed | **ОСТАВИТЬ** — ошибка этапа ingest |
| 167 | warn | metadata not found | **УЖЕ ЕСТЬ** — строки 165-166 |
| 170 | warn | metadata not found (fallback) | **УЖЕ ЕСТЬ** — строки 171-172 |
| 176 | info | dry run: skipping | **УДАЛИТЬ** — дублирует строку 177 |
| 188 | error | prepare files failed | **ОСТАВИТЬ** — ошибка этапа prepare |
| 194 | error | script build failed | **ОСТАВИТЬ** — ошибка этапа build |

### 2.3 Domain layer

#### `domain/script/script.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 31 | info | building script for destination | **УДАЛИТЬ** — domain не должен логировать |
| 35 | error | script config validation failed | **УДАЛИТЬ** — domain не должен логировать |
| 98 | error | unresolved placeholders | **УДАЛИТЬ** — domain не должен логировать |
| 110 | debug | selecting N audio tracks | **УДАЛИТЬ** — domain не должен логировать |
| 150 | debug | validating script config | **УДАЛИТЬ** — domain не должен логировать |
| 153 | error | destination must not be empty | **УДАЛИТЬ** — domain не должен логировать |
| 158 | error | at least one stream required | **УДАЛИТЬ** — domain не должен логировать |
| 171 | error | at least one video stream required | **УДАЛИТЬ** — domain не должен логировать |
| 185 | error | selection references non-existent stream | **УДАЛИТЬ** — domain не должен логировать |

#### `domain/callname/names_input.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 90 | debug | building call name for file | **УДАЛИТЬ** — domain не должен логировать |
| 103 | error | filename does not start with digit | **УДАЛИТЬ** — domain не должен логировать |
| 121 | info | building call names for N files | **УДАЛИТЬ** — domain не должен логировать |
| 161 | info | building call names for N files | **УДАЛИТЬ** — domain не должен логировать |
| 166 | error | call name for file failed | **УДАЛИТЬ** — domain не должен логировать |
| 176 | debug | derived output base | **УДАЛИТЬ** — domain не должен логировать |

#### `domain/callname/names_output.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 36 | info | predicting outputs for unit N | **УДАЛИТЬ** — domain не должен логировать |
| 81 | debug | hardsub detected: file | **УДАЛИТЬ** — domain не должен логировать |

#### `domain/ingest/unit.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 18 | debug | new media unit created | **УДАЛИТЬ** — domain не должен логировать |
| 25 | info | parsing media unit from N files | **УДАЛИТЬ** — domain не должен логировать |
| 31 | error | failed to parse probe data | **УДАЛИТЬ** — domain не должен логировать |
| 56 | error | failed to extract ID | **УДАЛИТЬ** — domain не должен логировать |
| 60 | error | inconsistent IDs | **УДАЛИТЬ** — domain не должен логировать |
| 95 | info | validating media unit: N streams | **УДАЛИТЬ** — domain не должен логировать |
| 108 | warn | duplicate stream coordinate | **УДАЛИТЬ** — domain не должен логировать |
| 120 | warn | video resolution below SD | **УДАЛИТЬ** — domain не должен логировать |
| 138 | warn | video bitrate suspiciously low | **УДАЛИТЬ** — domain не должен логировать |

#### `domain/ingest/extract.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 10 | debug | extracting ID from file | **УДАЛИТЬ** — domain не должен логировать |
| 23 | error | filename does not start with digit | **УДАЛИТЬ** — domain не должен логировать |
| 38 | debug | extracted ID N from file | **УДАЛИТЬ** — domain не должен логировать |

### 2.4 Platform layer

#### `platform/retry/retry.go`

| Строка | Уровень | Сообщение | Статус |
|--------|---------|-----------|--------|
| 31 | debug | retry attempt N/M failed | **УДАЛИТЬ** — retry не имеет доступа к логгеру |
| 38 | error | all retries exhausted | **УДАЛИТЬ** — retry не имеет доступа к логгеру |
| 45 | debug | retrying in N | **УДАЛИТЬ** — retry не имеет доступа к логгеру |
| 48 | debug | retry succeeded | **УДАЛИТЬ** — retry не имеет доступа к логгеру |

---

## 3. Сводная таблица решений

### 3.1 Категории решений

| Категория | Количество | Описание |
|-----------|-----------|----------|
| **ОСТАВИТЬ** | 22 | Полезные, обоснованные точки логирования |
| **УДАЛИТЬ** | 58 | Ошибки уже возвращаются как err, или domain, или инициализация |
| **НЕ ВВОДИТЬ** | 4 | Шумные вызовы (на каждый файл) |
| **УЖЕ ЕСТЬ** | 4 | Дублируют существующее логирование |

### 3.2 Детальная таблица по файлам

| Файл | ОСТАВИТЬ | УДАЛИТЬ | НЕ ВВОДИТЬ | УЖЕ ЕСТЬ | Итого |
|------|----------|---------|------------|----------|-------|
| `infra/logging/logger.go` | 1 | 3 | — | — | 4 |
| `infra/config/config.go` | 3 | — | — | — | 3 |
| `infra/metadata/types.go` | 1 | 3 | — | — | 4 |
| `infra/metadata/repository.go` | 2 | 1 | — | — | 3 |
| `infra/fso/filescanner.go` | 1 | 4 | — | — | 5 |
| `infra/fso/ffprobe.go` | — | 2 | 1 | — | 3 |
| `infra/script/script.go` | 1 | 3 | — | — | 4 |
| `infra/cache/cache.go` | — | 1 | 2 | — | 3 |
| `infra/persistence/jsonstore/jsonstore.go` | 3 | 5 | 1 | — | 9 |
| `app/ingest/ingest.go` | 1 | 3 | — | — | 4 |
| `app/script/build.go` | 3 | 1 | — | — | 4 |
| `app/pipeline/pipeline.go` | 5 | 2 | — | 4 | 11 |
| `domain/script/script.go` | — | 8 | — | — | 8 |
| `domain/callname/names_input.go` | — | 5 | — | — | 5 |
| `domain/callname/names_output.go` | — | 1 | — | — | 1 |
| `domain/ingest/unit.go` | — | 7 | — | — | 7 |
| `domain/ingest/extract.go` | — | 3 | — | — | 3 |
| `platform/retry/retry.go` | — | 4 | — | — | 4 |
| **ИТОГО** | **22** | **58** | **4** | **4** | **88** |

### 3.3 Обоснования по категориям

#### ОСТАВИТЬ (22 точки)

**Критерии:**
- Сигнал о завершении/запуске важного этапа
- Один раз за операцию (не на каждый элемент)
- Не дублирует существующее логирование
- Помогает отладке конфигурации и инициализации

**Примеры:**
- `config.Load` — загрузка конфига, fallback на дефолт
- `pipeline.Run` — старт/остановка пайплайна
- `script.Render` — запись скрипта на диск
- `jsonstore.Save` — сохранение стора при shutdown

#### УДАЛИТЬ (58 точек)

**Критерии:**
- Ошибка уже возвращается как `error` в вызывающий код
- Domain-слой (нарушает чистоту домена)
- Инициализация одного раза при старте
- `retry` пакет не имеет доступа к логгеру

**Примеры:**
- `failed to read metadata file` — ошибка возвращается как err
- `new media unit created` — инициализация, один раз
- `filename does not start with digit` — domain validation, ошибка возвращается

#### НЕ ВВОДИТЬ (4 точки)

**Критерии:**
- Вызывается на КАЖДЫЙ файл/элемент (шум)
- Кэш не используется активно
- retry пакет не имеет логгера

**Примеры:**
- `probing file: path` — вызывается для каждого файла, создаст сотни логов
- `cache set key=N` — кэш пустой/не используется
- `retry attempt N/M` — retry не имеет логгера

#### УЖЕ ЕСТЬ (4 точки)

**Критерии:**
- Дублирует уже существующий вызов логирования
- Следующая строка после закомментированной уже логирует

**Примеры:**
- `scan failed` — строка 91: `p.logger.Warn("scan failed"...`
- `processing unit: path` — строка 156: `p.logger.Info("processing"...`
- `metadata not found` — строки 165-166, 171-172 уже логируют

---

## 4. Рекомендации по приоритетам

### Приоритет 1: Высокая ценность (конфигурация и инициализация)

1. **`infra/config/config.go`** — все 3 точки. Пользователь должен видеть, какой конфиг загружен, и предупреждение о дефолтном конфиге.
2. **`app/pipeline/pipeline.go`** — старт/остановка пайплайна. Ключевые точки жизненного цикла.
3. **`infra/logging/logger.go`** — `Close()` — сигнал завершения работы приложения.

### Приоритет 2: Средняя ценность (процессинг)

4. **`app/script/build.go`** — начало/конец построения скрипта. Помогает отследить прогресс.
5. **`infra/script/script.go`** — запись скрипта на диск. Финальный артефакт.
6. **`infra/persistence/jsonstore/jsonstore.go`** — сохранение стора. Данные важны.
7. **`infra/metadata/repository.go`** — создание репозитория, lookup. Помогает отладить метаданные.

### Приоритет 3: Низкая ценность (уже покрыто)

8. **`app/ingest/ingest.go`** — `Info` при ингесте директории. Полезно, но дублирует pipeline.
9. **`infra/fso/filescanner.go`** — `Info` при сканировании. Полезно, но дублирует pipeline.
10. **`infra/metadata/types.go`** — загрузка метаданных. Полезно при отладке.

### Удалить без сожаления

11. **Все domain-логирования** — 23 точки. Domain не должен знать о логгере.
12. **Шумные вызовы** — 4 точки. ffprobe на каждый файл, кэш, retry.
13. **Ошибки, возвращаемые как err** — 31 точка. Нет логгера в контексте.

---

## 5. Архитектурные замечания

### 5.1 Domain-слой

Все закомментированные логи в `domain/` нарушают принцип чистоты домена. Domain-пакет не должен импортировать `infra/logging`. Варианты:
- **Удалить** все domain-логи (рекомендуется)
- **Перенести** логи в application-слой, который вызывает domain-функции
- **Вернуть richer errors** — domain возвращает ошибки с контекстом, application логирует

### 5.2 retry-пакет

Пакет `platform/retry` не имеет доступа к логгеру. Логирование retry-повторов требует инъекции логгера, что усложняет API. Рекомендация: удалить эти логи.

### 5.3 Дублирование

В `pipeline.go` есть дублирование: закомментированные `//log=` и реальные `p.logger.Info/Warn` вызываются для одних и тех же событий. При активации `//log=` нужно удалить дублирующие вызовы.

### 5.4 Формат комментариев

Формат `//log=level msg="..."` не является частью инфраструктуры. Это ручной формат, который:
- Не парсится никаким инструментом
- Не форматируется линтерами (gocritic ругается на `commentFormatting`)
- Не имеет единого стиля (где-то `//log=`, где-то `// log=`)

Рекомендация: при активации логов заменить `//log=...` на реальные вызовы `logger.X()`.
