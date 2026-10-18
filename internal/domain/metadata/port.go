package metadata

// ProductRepository provides access to media products from Amedia.
type ProductRepository interface {
	FindByID(id int64) (*Found, error)
}
