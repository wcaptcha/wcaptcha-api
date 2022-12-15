package store

type Storer interface {
	// Init storage
	Init() error

	// Put an object into storage
	Put(key string, obj any) error

	// Get an object from storage
	Get(key string, obj any) error

	// Check if an object is in storage
	KeyExists(key string) (bool, error)

	// List objects by prefix
	List(prefix string) ([]string, error)

	// Delete an object
	Delete(key string) error
}
