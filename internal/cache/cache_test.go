package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	c := New("/tmp/test", 5*time.Minute)
	if c.dir != "/tmp/test" {
		t.Errorf("dir = %q, want %q", c.dir, "/tmp/test")
	}
	if c.ttl != 5*time.Minute {
		t.Errorf("ttl = %v, want %v", c.ttl, 5*time.Minute)
	}
}

func TestSetAndGet(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 5*time.Minute)

	data := []byte(`[{"role":"reader"}]`)
	if err := c.Set(data); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	got, ok := c.Get()
	if !ok {
		t.Fatal("Get() returned ok=false, want true")
	}
	if string(got) != string(data) {
		t.Errorf("Get() data = %q, want %q", got, data)
	}
}

func TestGet_ExpiredTTL(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 1*time.Nanosecond) // extremely short TTL

	data := []byte(`{"test":true}`)
	if err := c.Set(data); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	// Sleep to guarantee TTL has elapsed.
	time.Sleep(2 * time.Millisecond)

	got, ok := c.Get()
	if ok {
		t.Errorf("Get() returned ok=true after TTL expired, data = %q", got)
	}
	if got != nil {
		t.Errorf("Get() data = %q, want nil", got)
	}
}

func TestGet_MissingMetaFile(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 5*time.Minute)

	// No files written - meta file does not exist.
	got, ok := c.Get()
	if ok {
		t.Errorf("Get() returned ok=true with no cache files, data = %q", got)
	}
	if got != nil {
		t.Errorf("Get() data = %q, want nil", got)
	}
}

func TestGet_CorruptMetaFile(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 5*time.Minute)

	// Write a valid data file but corrupt meta.
	if err := os.WriteFile(filepath.Join(dir, dataFile), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, metaFile), []byte(`not json`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, ok := c.Get()
	if ok {
		t.Errorf("Get() returned ok=true with corrupt meta, data = %q", got)
	}
	if got != nil {
		t.Errorf("Get() data = %q, want nil", got)
	}
}

func TestGet_MissingDataFile(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 5*time.Minute)

	// Write a valid meta file but no data file.
	m := meta{WrittenAt: time.Now()}
	raw, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(dir, metaFile), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	got, ok := c.Get()
	if ok {
		t.Errorf("Get() returned ok=true with missing data file, data = %q", got)
	}
	if got != nil {
		t.Errorf("Get() data = %q, want nil", got)
	}
}

func TestSet_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 5*time.Minute)

	if err := c.Set([]byte(`test`)); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	for _, name := range []string{dataFile, metaFile} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("Stat(%s) error: %v", name, err)
		}
		perm := info.Mode().Perm()
		if perm != 0o600 {
			t.Errorf("%s permissions = %o, want 0600", name, perm)
		}
	}
}

func TestSet_NonExistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	c := New(dir, 5*time.Minute)

	err := c.Set([]byte(`test`))
	if err == nil {
		t.Error("Set() in non-existent directory returned nil error, want error")
	}
}

func TestAge_ValidCache(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 5*time.Minute)

	if err := c.Set([]byte(`data`)); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	age, err := c.Age()
	if err != nil {
		t.Fatalf("Age() error: %v", err)
	}
	// Age should be very small - well under 1 second.
	if age > 1*time.Second {
		t.Errorf("Age() = %v, expected < 1s", age)
	}
	if age < 0 {
		t.Errorf("Age() = %v, expected >= 0", age)
	}
}

func TestAge_MissingMeta(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 5*time.Minute)

	age, err := c.Age()
	if err == nil {
		t.Errorf("Age() with no cache returned nil error, age = %v", age)
	}
	if age != 0 {
		t.Errorf("Age() = %v, want 0", age)
	}
}

func TestAge_CorruptMeta(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 5*time.Minute)

	if err := os.WriteFile(filepath.Join(dir, metaFile), []byte(`{garbage`), 0o600); err != nil {
		t.Fatal(err)
	}

	age, err := c.Age()
	if err == nil {
		t.Errorf("Age() with corrupt meta returned nil error, age = %v", age)
	}
	if age != 0 {
		t.Errorf("Age() = %v, want 0", age)
	}
}

func TestGet_MetaWithZeroTime(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 5*time.Minute)

	// Write meta with zero time - effectively always expired.
	m := meta{WrittenAt: time.Time{}}
	raw, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(dir, metaFile), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, dataFile), []byte(`data`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, ok := c.Get()
	if ok {
		t.Errorf("Get() returned ok=true with zero-time meta, data = %q", got)
	}
}
