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
//log=info msg="metadata repository created from: %s", path
func New(path string) (*Repository, error) {
	root, err := LoadRoot(path)
	if err != nil {
		//log=error msg="failed to load metadata from %q: %v", path, err
		return nil, err
	}
	return &Repository{root: root}, nil
}

// FindByID implements domain/metadata.ProductRepository.
//log=debug msg="metadata lookup: id=%d", id
func (r *Repository) FindByID(id int64) (*mdomain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.root == nil {
		//log=error msg="metadata repository not initialized"
		return nil, errors.New("metadata repository not initialized")
	}

	found, ok := r.root.FindByID(id, WithDescendants)
	if !ok {
		//log=warn msg="product %d not found in metadata", id
		return nil, fmt.Errorf("product %d not found", id)
	}
	return FoundToProduct(found), nil
}
