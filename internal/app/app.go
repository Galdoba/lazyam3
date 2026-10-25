package app

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Galdoba/lazyam3/internal/infra/cache"
	"github.com/Galdoba/lazyam3/internal/infra/config"
	"github.com/Galdoba/lazyam3/internal/infra/logging"
	"github.com/Galdoba/lazyam3/internal/infra/persistence/jsonstore"
)

// AppModules is a container for shared application dependencies.
//
// Rule: only Config and Logger are eagerly initialized — they are needed
// before business logic so that startup errors can be logged. Everything else
// is lazily created on first access through the corresponding method.
//
// Commands declare their needs by calling methods in Action.
// Before in commands is used only for fail-fast when it's truly needed.
type AppModules struct {
	Config config.Config
	Logger *logging.Logger

	cacheOnce sync.Once
	cache     *cache.Cache

	storeOnce sync.Once
	store     *jsonstore.Store
	storeErr  error
}

// New creates an empty container. Init populates it after flag parsing.
func New() *AppModules {
	return &AppModules{}
}

// Init loads the config (respecting the explicit path from the flag) and creates the logger.
// Called once from Root.Before. Idempotent.
//log=info msg="modules initialized"
func (m *AppModules) Init(configPath string) error {
	if m.Logger != nil {
		return nil // already initialized
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		//log=error msg="failed to load config: %v", err
		if errors.Is(err, config.ErrDefaultConfig) {
			return err
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger, err := logging.NewLogger(cfg.Logger)
	if err != nil {
		//log=error msg="failed to init logger: %v", err
		return fmt.Errorf("failed to init logger: %w", err)
	}

	m.Config = cfg
	m.Logger = logger
	return nil
}

// Cache returns the in-memory cache, creating it on first access.
func (m *AppModules) Cache() *cache.Cache {
	m.cacheOnce.Do(func() {
		m.cache = cache.New()
	})
	return m.cache
}

// Store returns the store, creating it on first access.
// Returns the same error on subsequent calls if initialization failed.
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

// Shutdown closes everything that was actually created.
// Order: resources first (store), then logger — so the logger can
// write shutdown errors. Idempotent.
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
