package metadata

import (
	"errors"
	"fmt"
	"sync"

	mdomain "github.com/Galdoba/lazyam3/internal/domain/metadata"
)

// Repository loads metadata.json and provides FindByID access.
type Repository struct {
	mu   sync.RWMutex
	root *Root
}

// New creates a Repository that loads from the given path.
func New(path string) (*Repository, error) {
	root, err := LoadRoot(path)
	if err != nil {
		return nil, err
	}
	return &Repository{root: root}, nil
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
		return nil, fmt.Errorf("product %d not found", id)
	}
	return FoundToProduct(found), nil
}
