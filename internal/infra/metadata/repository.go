package metadata

import (
	"errors"
	"fmt"
	"sync"

	mdomain "github.com/Galdoba/lazyam3/internal/domain/metadata"
	"github.com/Galdoba/lazyam3/internal/infra/logging"
)

// Repository loads metadata.json and provides FindByID access.
type Repository struct {
	mu     sync.RWMutex
	root   *Root
	logger *logging.Logger
}

// New creates a Repository that loads from the given path.
func New(logger *logging.Logger, path string) (*Repository, error) {
	root, err := LoadRoot(path)
	if err != nil {
		return nil, err
	}
	repo := &Repository{root: root, logger: logger}
	if logger != nil {
		logger.Info("metadata repository created from path", "path", path)
	}
	return repo, nil
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
