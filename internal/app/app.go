package app

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Galdoba/lazyam3/internal/app/cache"
	"github.com/Galdoba/lazyam3/internal/app/config"
	"github.com/Galdoba/lazyam3/internal/app/logging"
	"github.com/Galdoba/lazyam3/internal/app/storage/jsonstore"
)

// AppModules — контейнер общих зависимостей приложения.
//
// Правило: eager только Config и Logger — они нужны до бизнес-логики,
// чтобы можно было логировать ошибки запуска. Всё остальное создаётся
// лениво при первом обращении через соответствующий метод.
//
// Команды объявляют потребности, вызывая методы в Action.
// Before в командах — только для fail-fast, если это реально нужно.
type AppModules struct {
	Config config.Config
	Logger *logging.Logger

	cacheOnce sync.Once
	cache     *cache.Cache

	storeOnce sync.Once
	store     *jsonstore.Store
	storeErr  error
}

// New создаёт пустой контейнер. Init наполняет его после парсинга флагов.
func New() *AppModules {
	return &AppModules{}
}

// Init загружает конфиг (с учётом явного пути из флага) и создаёт логгер.
// Вызывается один раз из Root.Before. Идемпотентен.
func (m *AppModules) Init(configPath string) error {
	if m.Logger != nil {
		return nil // уже инициализировано
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger, err := logging.NewLogger(cfg.Logger)
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}

	m.Config = cfg
	m.Logger = logger
	return nil
}

// Cache возвращает in-memory кэш, создавая его при первом обращении.
func (m *AppModules) Cache() *cache.Cache {
	m.cacheOnce.Do(func() {
		m.cache = cache.New()
	})
	return m.cache
}

// Store возвращает хранилище, создавая его при первом обращении.
// Возвращает ту же ошибку при повторных вызовах, если инициализация упала.
func (m *AppModules) Store() (*jsonstore.Store, error) {
	m.storeOnce.Do(func() {
		if m.Logger == nil {
			m.storeErr = fmt.Errorf("app modules not initialized: call Init first")
			return
		}
		m.store, m.storeErr = jsonstore.New(m.Config.Storage.Path)
	})
	return m.store, m.storeErr
}

// Shutdown закрывает всё, что было реально создано.
// Порядок: сначала ресурсы (store), потом логгер — чтобы логгер мог
// записать ошибки закрытия. Идемпотентен.
func (m *AppModules) Shutdown() error {
	if m == nil {
		return nil
	}
	var errs []error

	if m.store != nil {
		errs = append(errs, m.store.Close())
	}
	if m.Logger != nil {
		errs = append(errs, m.Logger.Close())
	}

	return errors.Join(errs...)
}
