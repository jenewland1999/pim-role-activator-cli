// Package cache provides a file-based TTL cache for eligible-role data.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// meta stores the timestamp of the last cache write.
type meta struct {
	WrittenAt time.Time `json:"written_at"`
}

// Cache is a simple file-based cache backed by a data file and a metadata
// sidecar that records when the data was written. Entries expire after the
// configured TTL. The prefix distinguishes independent caches within the
// same directory (e.g. "eligible-roles", "active-roles").
type Cache struct {
	dir    string
	ttl    time.Duration
	prefix string
}

// New returns a Cache rooted in dir with the given TTL. Prefix is prepended to
// the cache filenames so that multiple independent caches can coexist in the
// same directory.
func New(dir string, ttl time.Duration, prefix string) *Cache {
	return &Cache{dir: dir, ttl: ttl, prefix: prefix}
}

// dataPath returns the absolute path to the cache data file.
func (c *Cache) dataPath() string {
	return filepath.Join(c.dir, c.prefix+"-data.json")
}

// metaPath returns the absolute path to the cache metadata sidecar.
func (c *Cache) metaPath() string {
	return filepath.Join(c.dir, c.prefix+"-meta.json")
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
	data, err := os.ReadFile(c.dataPath())
	if err != nil {
		return nil, false
	}
	return data, true
}

// Set writes data to the cache and records the current time in the metadata
// sidecar. Both files are written atomically via write-to-temp-then-rename to
// prevent corruption if the process is interrupted mid-write. Files are created
// with 0o600 permissions.
func (c *Cache) Set(data []byte) error {
	if err := atomicWriteFile(c.dataPath(), data); err != nil {
		return fmt.Errorf("cache: write data: %w", err)
	}
	m := meta{WrittenAt: time.Now()}
	raw, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("cache: marshal meta: %w", err)
	}
	if err := atomicWriteFile(c.metaPath(), raw); err != nil {
		return fmt.Errorf("cache: write meta: %w", err)
	}
	return nil
}

// atomicWriteFile writes data to a temporary file in the same directory as
// target and then renames it into place. Because os.Rename is atomic on POSIX
// and Windows (same volume), a crash at any point will leave either the old
// file intact or the fully-written new file — never a partial write.
func atomicWriteFile(target string, data []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, filepath.Base(target)+".tmp*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	// Clean up the temp file on any error path.
	success := false
	defer func() {
		if !success {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, target); err != nil {
		return err
	}
	success = true
	return nil
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
	raw, err := os.ReadFile(c.metaPath())
	if err != nil {
		return nil, err
	}
	var m meta
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
