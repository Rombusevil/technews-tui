package bookmark

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreOperations(t *testing.T) {
	// Setup temp file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bookmarks.json")

	store := NewStore(path)
	
	// Has should be false initially
	if store.Has("source_url_1") {
		t.Error("expected Has to be false for empty store")
	}
	
	b1 := Bookmark{
		Kind:         "post",
		Title:        "Test Post",
		SourceURL:    "source_url_1",
		BookmarkedAt: time.Now(),
	}

	// Test Add
	if err := store.Add(b1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if !store.Has("source_url_1") {
		t.Error("expected Has to be true after Add")
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
		t.Fatalf("expected 1 bookmark, got %d", len(list))
	}
	if list[0].Title != "Test Post" {
		t.Errorf("expected Title 'Test Post', got %q", list[0].Title)
	}

	// Test Remove
	if err := store.Remove("source_url_1"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if store.Has("source_url_1") {
		t.Error("expected Has to be false after Remove")
	}
}
