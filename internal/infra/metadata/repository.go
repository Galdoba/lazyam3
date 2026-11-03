package metadata

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	mdomain "github.com/Galdoba/lazyam3/internal/domain/metadata"
	"github.com/Galdoba/lazyam3/internal/infra/logging"
)

// Repository loads metadata.json and provides FindByID access.
type Repository struct {
	mu           sync.RWMutex
	root         *Root
	logger       *logging.Logger
	metadataPath string
	lastMod      time.Time
}

// New creates a Repository that loads from the given path.
func New(logger *logging.Logger, path string) (*Repository, error) {
	root, err := LoadRoot(path)
	if err != nil {
		return nil, err
	}
	repo := &Repository{root: root, logger: logger, metadataPath: path}
	if stat, err := os.Stat(path); err == nil {
		repo.lastMod = stat.ModTime()
	}
	if logger != nil {
		logger.Info("metadata repository created from path", "path", path)
	}
	return repo, nil
}

// Reload stats the metadata file and reloads it if the modification time changed.
// Returns (changed, error). On reload failure, the previous snapshot is retained.
func (r *Repository) Reload() (bool, error) {
	stat, err := os.Stat(r.metadataPath)
	if err != nil {
		return false, err
	}
	if stat.ModTime().Equal(r.lastMod) {
		return false, nil
	}

	root, err := LoadRoot(r.metadataPath)
	if err != nil {
		return false, err
	}

	r.mu.Lock()
	r.root = root
	r.lastMod = stat.ModTime()
	r.mu.Unlock()

	if r.logger != nil {
		r.logger.Info("metadata reloaded", "path", r.metadataPath)
	}
	return true, nil
}

// FindByID implements domain/metadata.ProductRepository.
func (r *Repository) FindByID(id int64) (*mdomain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.root == nil {
		return nil, errors.New("metadata repository not initialized")
	}

	found, ok := r.root.FindByID(id, WithDescendants)
	if !ok {
		if r.logger != nil {
			r.logger.Warn("product not found in metadata", "id", id)
		}
		return nil, fmt.Errorf("product %d not found", id)
	}
	return FoundToProduct(found), nil
}
