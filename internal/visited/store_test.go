package visited

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreOperations(t *testing.T) {
	// Setup temp file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "visited.json")

	store := NewStore(path)

	// Has should be false initially
	if store.Has("source_url_1") {
		t.Error("expected Has to be false for empty store")
	}

	b1 := item{
		SourceURL: "source_url_1",
		VisitedAt: time.Now(),
	}

	// Test AddOrUpdate
	if err := store.AddOrUpdate(b1.SourceURL); err != nil {
		t.Fatalf("AddOrUpdate failed: %v", err)
	}

	if !store.Has("source_url_1") {
		t.Error("expected Has to be true after AddOrUpdate")
	}

	if err := store.AddOrUpdate(b1.SourceURL); err != nil {
		t.Fatalf("AddOrUpdate failed: %v", err)
	}

	if len(store.List()) > 1 {
		t.Fatalf("Only 1 item with b1.SourceURL should exist.")
	}

	// Test Toggle off
	added, err := store.Toggle(b1)
	if err != nil {
		t.Fatalf("Toggle failed: %v", err)
	}
	if added {
		t.Error("Toggle should return false when removing")
	}
	if store.Has("source_url_1") {
		t.Error("expected Has to be false after Toggle off")
	}

	// Test Toggle on
	added, err = store.Toggle(b1)
	if err != nil {
		t.Fatalf("Toggle failed: %v", err)
	}
	if !added {
		t.Error("Toggle should return true when adding")
	}

	// Test persistence
	store2 := NewStore(path)
	if err := store2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	list := store2.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 url, got %d", len(list))
	}
	if list[0].SourceURL != "source_url_1" {
		t.Errorf("expected Title 'Test Post', got %q", list[0].SourceURL)
	}

	// Test Remove
	if err := store.Remove("source_url_1"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if store.Has("source_url_1") {
		t.Error("expected Has to be false after Remove")
	}
}
