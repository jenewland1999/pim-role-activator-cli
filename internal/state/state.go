// Package state manages local activation records in ~/.pim/activations.json.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

// Store manages loading, pruning, and saving activation records.
type Store struct {
	Path string // e.g. ~/.pim/activations.json
}

// New creates a Store for the given file path.
func New(path string) *Store {
	return &Store{Path: path}
}

// Load reads all non-expired activation records from disk.
// Returns an empty slice (not an error) when the file does not exist.
func (s *Store) Load() ([]model.ActivationRecord, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("state: read %s: %w", s.Path, err)
	}

	var records []model.ActivationRecord
	if err := json.Unmarshal(data, &records); err != nil {
		// Corrupt state file — start fresh rather than aborting.
		return nil, nil
	}

	// Prune expired entries in-place.
	now := time.Now().Unix()
	active := records[:0]
	for _, r := range records {
		if r.ExpiresEpoch > now {
			active = append(active, r)
		}
	}
	return active, nil
}

// Save persists records to disk as formatted JSON.
func (s *Store) Save(records []model.ActivationRecord) error {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("state: marshal activations: %w", err)
	}
	if err := os.WriteFile(s.Path, data, 0o600); err != nil {
		return fmt.Errorf("state: write %s: %w", s.Path, err)
	}
	return nil
}

// Append loads existing records, merges new ones, prunes expired entries and saves.
func (s *Store) Append(newRecords []model.ActivationRecord) error {
	existing, err := s.Load()
	if err != nil {
		return fmt.Errorf("state: append: %w", err)
	}
	all := append(existing, newRecords...)
	return s.Save(all)
}

// LookupJustification returns a map of composite key → justification
// built from the stored activation records.
func (s *Store) LookupJustification() map[string]string {
	records, _ := s.Load()
	m := make(map[string]string, len(records))
	for _, r := range records {
		m[r.LookupKey()] = r.Justification
	}
	return m
}
