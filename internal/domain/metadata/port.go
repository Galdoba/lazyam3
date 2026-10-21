package metadata

// ProductRepository provides access to media products.
type ProductRepository interface {
	FindByID(id int64) (*Product, error)
}
