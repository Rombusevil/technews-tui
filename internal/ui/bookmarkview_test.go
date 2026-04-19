package ui

import (
	"strings"
	"technews-tui/internal/config"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"technews-tui/internal/bookmark"
)

func TestBookmarkViewDelete(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/bookmarks.json"
	store := bookmark.NewStore(path)
	
	b1 := bookmark.Bookmark{
		Kind: "post",
		Title: "Test Bookmark",
		SourceURL: "source_url_1",
		BookmarkedAt: time.Now(),
	}
	
	_ = store.Add(b1)

	m := NewBookmarkModel(store, &config.Config{})
	m.SetSize(80, 24)

	// Verify it shows up
	view := m.View()
	if !strings.Contains(view, "Test Bookmark") {
		t.Fatalf("expected bookmark view to contain title, got: %q", view)
	}

	// Press 'd' to delete
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	newModel, cmd := m.Update(msg)
	
	if cmd != nil {
		t.Error("expected no command returned from delete")
	}

	m = newModel.(BookmarkModel)
	view = m.View()
	
	if strings.Contains(view, "Test Bookmark") {
		t.Fatalf("expected bookmark to be deleted from view")
	}

	if store.Has("source_url_1") {
		t.Fatal("expected bookmark to be removed from store")
	}
}
