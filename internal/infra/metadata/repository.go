package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/Galdoba/lazyam3/internal/domain/metadata"
)

// Repository loads metadata.json and provides FindByID access.
type Repository struct {
	mu   sync.RWMutex
	root *metadata.Root
}

// New creates a Repository that loads from the given path.
func New(path string) (*Repository, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metadata file: %w", err)
	}

	var root metadata.Root
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse metadata JSON: %w", err)
	}

	if len(root.Series) == 0 && len(root.Movies) == 0 {
		return nil, fmt.Errorf("empty metadata: no series or movies found")
	}

	return &Repository{root: &root}, nil
}

// FindByID implements domain/metadata.ProductRepository.
func (r *Repository) FindByID(id int64) (*metadata.Found, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.root == nil {
		return nil, fmt.Errorf("metadata repository not initialized")
	}

	found, ok := r.root.FindByID(id, metadata.WithDescendants)
	if !ok {
		return nil, fmt.Errorf("product %d not found", id)
	}
	return found, nil
}
