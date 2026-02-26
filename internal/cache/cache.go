// Package cache provides a file-based TTL cache for eligible-role data.
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	dataFile = "eligible-roles.json"
	metaFile = "cache-meta"
)

// meta stores the timestamp of the last cache write.
type meta struct {
	WrittenAt time.Time `json:"written_at"`
}

// Cache is a simple file-based cache backed by a data file and a metadata
// sidecar that records when the data was written. Entries expire after the
// configured TTL.
type Cache struct {
	dir string
	ttl time.Duration
}

// New returns a Cache rooted in dir with the given TTL.
func New(dir string, ttl time.Duration) *Cache {
	return &Cache{dir: dir, ttl: ttl}
}

// Get returns the cached data and true if the cache is present and has not
// expired. Returns nil, false on any error or when the TTL has elapsed.
func (c *Cache) Get() ([]byte, bool) {
	m, err := c.readMeta()
	if err != nil {
		return nil, false
	}
	if time.Since(m.WrittenAt) > c.ttl {
		return nil, false
	}
	data, err := os.ReadFile(filepath.Join(c.dir, dataFile))
	if err != nil {
		return nil, false
	}
	return data, true
}

// Set writes data to the cache and records the current time in the metadata
// sidecar. Both files are written with 0o600 permissions.
func (c *Cache) Set(data []byte) error {
	if err := os.WriteFile(filepath.Join(c.dir, dataFile), data, 0o600); err != nil {
		return err
	}
	m := meta{WrittenAt: time.Now()}
	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.dir, metaFile), raw, 0o600)
}

// Age returns the time elapsed since the last cache write. Returns zero and
// an error if the metadata cannot be read (e.g. cache has never been written).
func (c *Cache) Age() (time.Duration, error) {
	m, err := c.readMeta()
	if err != nil {
		return 0, err
	}
	return time.Since(m.WrittenAt), nil
}

// readMeta loads and parses the metadata sidecar.
func (c *Cache) readMeta() (*meta, error) {
	raw, err := os.ReadFile(filepath.Join(c.dir, metaFile))
	if err != nil {
		return nil, err
	}
	var m meta
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
