package meta

// DefaultStore defines the default store backend.
const DefaultStore = "local"

var backend map[string]Backend

// Backend is an interface which describes what a store should support.
type Backend interface {
	// meta store contructor.
	New(Config) error

	// Put write key-value into store.
	Put(bucket string, key string, value []byte) error

	// Get read object from store.
	Get(bucket string, key string) ([]byte, error)

	// Remove remove all data of the key.
	Remove(bucket string, key string) error

	// List return all objects with specify bucket.
	List(bucket string) ([][]byte, error)

	// Keys return all keys.
	Keys(bucket string) ([]string, error)

	// Path returns the path with the specified key.
	Path(key string) string
}

// Register registers a backend to be daemon's store.
func Register(name string, b Backend) {
	if backend == nil {
		backend = make(map[string]Backend)
	}
	backend[name] = b
}
