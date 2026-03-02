package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

func futureEpoch() int64 {
	return time.Now().Add(1 * time.Hour).Unix()
}

func pastEpoch() int64 {
	return time.Now().Add(-1 * time.Hour).Unix()
}

func newStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	return New(filepath.Join(dir, "activations.json"))
}

func TestNew(t *testing.T) {
	s := New("/tmp/test/activations.json")
	if s.Path != "/tmp/test/activations.json" {
		t.Errorf("Path = %q, want %q", s.Path, "/tmp/test/activations.json")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	s := newStore(t)
	records, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if records != nil {
		t.Errorf("Load() = %v, want nil for missing file", records)
	}
}

func TestLoad_EmptyArray(t *testing.T) {
	s := newStore(t)
	if err := os.WriteFile(s.Path, []byte(`[]`), 0o600); err != nil {
		t.Fatal(err)
	}
	records, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("Load() length = %d, want 0", len(records))
	}
}

func TestLoad_CorruptJSON(t *testing.T) {
	s := newStore(t)
	if err := os.WriteFile(s.Path, []byte(`{not json!!!`), 0o600); err != nil {
		t.Fatal(err)
	}
	records, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v (corrupt files should return nil, nil)", err)
	}
	if records != nil {
		t.Errorf("Load() = %v, want nil for corrupt JSON", records)
	}
}

func TestLoad_PrunesExpiredRecords(t *testing.T) {
	s := newStore(t)
	records := []model.ActivationRecord{
		{
			Scope:            "/subscriptions/sub-1",
			RoleDefinitionID: "role-a",
			RoleName:         "Reader",
			Justification:    "active",
			ExpiresEpoch:     futureEpoch(),
		},
		{
			Scope:            "/subscriptions/sub-1",
			RoleDefinitionID: "role-b",
			RoleName:         "Writer",
			Justification:    "expired",
			ExpiresEpoch:     pastEpoch(),
		},
	}
	writeRecords(t, s.Path, records)
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Load() length = %d, want 1 (expired should be pruned)", len(got))
	}
	if got[0].Justification != "active" {
		t.Errorf("justification = %q, want %q", got[0].Justification, "active")
	}
}

func TestLoad_AllExpired(t *testing.T) {
	s := newStore(t)
	records := []model.ActivationRecord{
		{Scope: "s1", RoleDefinitionID: "r1", ExpiresEpoch: pastEpoch()},
		{Scope: "s2", RoleDefinitionID: "r2", ExpiresEpoch: pastEpoch()},
	}
	writeRecords(t, s.Path, records)
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Load() length = %d, want 0 (all expired)", len(got))
	}
}

func TestLoad_PreservesAllFields(t *testing.T) {
	s := newStore(t)
	exp := futureEpoch()
	records := []model.ActivationRecord{
		{
			Scope:            "/subscriptions/sub-1/resourceGroups/rg-test",
			RoleDefinitionID: "role-def-123",
			RoleName:         "Contributor",
			ScopeName:        "rg-test",
			Justification:    "testing fields",
			Duration:         "PT1H",
			ActivatedAt:      "2026-03-02T10:00:00Z",
			ExpiresEpoch:     exp,
		},
	}
	writeRecords(t, s.Path, records)
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Load() length = %d, want 1", len(got))
	}
	r := got[0]
	if r.Scope != "/subscriptions/sub-1/resourceGroups/rg-test" {
		t.Errorf("Scope = %q", r.Scope)
	}
	if r.RoleDefinitionID != "role-def-123" {
		t.Errorf("RoleDefinitionID = %q", r.RoleDefinitionID)
	}
	if r.RoleName != "Contributor" {
		t.Errorf("RoleName = %q", r.RoleName)
	}
	if r.ScopeName != "rg-test" {
		t.Errorf("ScopeName = %q", r.ScopeName)
	}
	if r.Justification != "testing fields" {
		t.Errorf("Justification = %q", r.Justification)
	}
	if r.Duration != "PT1H" {
		t.Errorf("Duration = %q", r.Duration)
	}
	if r.ActivatedAt != "2026-03-02T10:00:00Z" {
		t.Errorf("ActivatedAt = %q", r.ActivatedAt)
	}
	if r.ExpiresEpoch != exp {
		t.Errorf("ExpiresEpoch = %d, want %d", r.ExpiresEpoch, exp)
	}
}

func TestSave_RoundTrip(t *testing.T) {
	s := newStore(t)
	exp := futureEpoch()
	records := []model.ActivationRecord{
		{
			Scope:            "/subscriptions/sub-1",
			RoleDefinitionID: "role-a",
			RoleName:         "Reader",
			Justification:    "test justification",
			ExpiresEpoch:     exp,
		},
	}
	if err := s.Save(records); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load() after Save() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Load() length = %d, want 1", len(got))
	}
	if got[0].Justification != "test justification" {
		t.Errorf("Justification = %q, want %q", got[0].Justification, "test justification")
	}
}

func TestSave_EmptySlice(t *testing.T) {
	s := newStore(t)
	if err := s.Save([]model.ActivationRecord{}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	data, err := os.ReadFile(s.Path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	var raw []interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Errorf("saved file is not valid JSON array: %v", err)
	}
	if len(raw) != 0 {
		t.Errorf("saved array length = %d, want 0", len(raw))
	}
}

func TestSave_FilePermissions(t *testing.T) {
	s := newStore(t)
	if err := s.Save([]model.ActivationRecord{}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	info, err := os.Stat(s.Path)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestSave_OverwritesExisting(t *testing.T) {
	s := newStore(t)
	exp := futureEpoch()
	first := []model.ActivationRecord{
		{Scope: "s1", RoleDefinitionID: "r1", Justification: "first", ExpiresEpoch: exp},
	}
	if err := s.Save(first); err != nil {
		t.Fatalf("Save(first) error: %v", err)
	}
	second := []model.ActivationRecord{
		{Scope: "s2", RoleDefinitionID: "r2", Justification: "second", ExpiresEpoch: exp},
	}
	if err := s.Save(second); err != nil {
		t.Fatalf("Save(second) error: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Load() length = %d, want 1", len(got))
	}
	if got[0].Justification != "second" {
		t.Errorf("Justification = %q, want %q", got[0].Justification, "second")
	}
}

func TestSave_NonExistentDir(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "nonexistent", "activations.json"))
	err := s.Save([]model.ActivationRecord{})
	if err == nil {
		t.Error("Save() in non-existent directory returned nil error, want error")
	}
}

func TestAppend_ToEmpty(t *testing.T) {
	s := newStore(t)
	exp := futureEpoch()
	records := []model.ActivationRecord{
		{Scope: "s1", RoleDefinitionID: "r1", Justification: "new", ExpiresEpoch: exp},
	}
	if err := s.Append(records); err != nil {
		t.Fatalf("Append() error: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Load() length = %d, want 1", len(got))
	}
	if got[0].Justification != "new" {
		t.Errorf("Justification = %q, want %q", got[0].Justification, "new")
	}
}

func TestAppend_MergesWithExisting(t *testing.T) {
	s := newStore(t)
	exp := futureEpoch()
	existing := []model.ActivationRecord{
		{Scope: "s1", RoleDefinitionID: "r1", Justification: "existing", ExpiresEpoch: exp},
	}
	if err := s.Save(existing); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	added := []model.ActivationRecord{
		{Scope: "s2", RoleDefinitionID: "r2", Justification: "added", ExpiresEpoch: exp},
	}
	if err := s.Append(added); err != nil {
		t.Fatalf("Append() error: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Load() length = %d, want 2", len(got))
	}
}

func TestAppend_PrunesExpiredDuringMerge(t *testing.T) {
	s := newStore(t)
	exp := futureEpoch()
	expired := []model.ActivationRecord{
		{Scope: "s1", RoleDefinitionID: "r1", Justification: "expired", ExpiresEpoch: pastEpoch()},
	}
	writeRecords(t, s.Path, expired)
	fresh := []model.ActivationRecord{
		{Scope: "s2", RoleDefinitionID: "r2", Justification: "fresh", ExpiresEpoch: exp},
	}
	if err := s.Append(fresh); err != nil {
		t.Fatalf("Append() error: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Load() length = %d, want 1", len(got))
	}
	if got[0].Justification != "fresh" {
		t.Errorf("Justification = %q, want %q", got[0].Justification, "fresh")
	}
}

func TestLookupJustification_WithRecords(t *testing.T) {
	s := newStore(t)
	exp := futureEpoch()
	records := []model.ActivationRecord{
		{Scope: "/sub/1", RoleDefinitionID: "role-a", Justification: "reason one", ExpiresEpoch: exp},
		{Scope: "/sub/2", RoleDefinitionID: "role-b", Justification: "reason two", ExpiresEpoch: exp},
	}
	if err := s.Save(records); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	m := s.LookupJustification()
	if len(m) != 2 {
		t.Fatalf("LookupJustification() length = %d, want 2", len(m))
	}
	key1 := "/sub/1|role-a"
	if m[key1] != "reason one" {
		t.Errorf("m[%q] = %q, want %q", key1, m[key1], "reason one")
	}
	key2 := "/sub/2|role-b"
	if m[key2] != "reason two" {
		t.Errorf("m[%q] = %q, want %q", key2, m[key2], "reason two")
	}
}

func TestLookupJustification_EmptyFile(t *testing.T) {
	s := newStore(t)
	m := s.LookupJustification()
	if m == nil {
		t.Fatal("LookupJustification() = nil, want non-nil empty map")
	}
	if len(m) != 0 {
		t.Errorf("LookupJustification() length = %d, want 0", len(m))
	}
}

func TestLookupJustification_ExcludesExpired(t *testing.T) {
	s := newStore(t)
	exp := futureEpoch()
	records := []model.ActivationRecord{
		{Scope: "/sub/1", RoleDefinitionID: "role-a", Justification: "active", ExpiresEpoch: exp},
		{Scope: "/sub/2", RoleDefinitionID: "role-b", Justification: "gone", ExpiresEpoch: pastEpoch()},
	}
	writeRecords(t, s.Path, records)
	m := s.LookupJustification()
	if len(m) != 1 {
		t.Fatalf("LookupJustification() length = %d, want 1", len(m))
	}
	key := "/sub/1|role-a"
	if m[key] != "active" {
		t.Errorf("m[%q] = %q, want %q", key, m[key], "active")
	}
}

func writeRecords(t *testing.T, path string, records []model.ActivationRecord) {
	t.Helper()
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatalf("marshal records: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write records: %v", err)
	}
}
