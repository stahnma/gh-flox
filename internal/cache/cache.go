package cache

import (
	"bytes"
	"encoding/gob"
	"log"
	"os"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

func init() {
	gob.Register(0) // register int for gob encoding of cached star counts
}

// Cache wraps go-cache with GOB persistence.
type Cache struct {
	inner *gocache.Cache
}

// New creates an empty cache.
func New() *Cache {
	return &Cache{inner: gocache.New(4*time.Hour, 6*time.Hour)}
}

// LoadFromFile loads a cache from a GOB file, returning a fresh cache on error.
func LoadFromFile(filename string) (*Cache, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return New(), nil
		}
		return nil, err
	}
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	items := map[string]gocache.Item{}
	if err := dec.Decode(&items); err != nil {
		log.Printf("Cache decode error (starting fresh): %v", err)
		return New(), nil
	}
	return &Cache{inner: gocache.NewFrom(4*time.Hour, 6*time.Hour, items)}, nil
}

// SaveToFile saves the cache to a GOB file.
func (c *Cache) SaveToFile(filename string) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(c.inner.Items()); err != nil {
		return err
	}
	return os.WriteFile(filename, buf.Bytes(), 0600)
}

// Get retrieves a value by key.
func (c *Cache) Get(key string) (any, bool) {
	return c.inner.Get(key)
}

// Set stores a value with default expiration.
func (c *Cache) Set(key string, val any) {
	c.inner.Set(key, val, gocache.DefaultExpiration)
}

// Flush clears all cached items.
func (c *Cache) Flush() {
	c.inner.Flush()
}
