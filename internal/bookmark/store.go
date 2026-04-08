package bookmark

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Bookmark struct {
	Kind         string    `json:"kind"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	SourceURL    string    `json:"source_url"`
	Source       string    `json:"source"`
	SourceLabel  string    `json:"source_label"`
	SourceID     string    `json:"source_id"`
	Author       string    `json:"author"`
	Points       int       `json:"points"`
	CommentCount int       `json:"comment_count"`
	BookmarkedAt time.Time `json:"bookmarked_at"`
}

type Store struct {
	mu        sync.RWMutex
	path      string
	bookmarks []Bookmark
}

func NewStore(path string) *Store {
	return &Store{
		path: path,
	}
}

func DefaultPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "bookmarks.json"
	}
	return filepath.Join(configDir, "technews-tui", "bookmarks.json")
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &s.bookmarks)
}

func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.bookmarks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0o644)
}

func (s *Store) Add(b Bookmark) error {
	s.mu.Lock()
	s.bookmarks = append(s.bookmarks, b)
	s.mu.Unlock()
	return s.Save()
}

func (s *Store) Remove(sourceURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var updated []Bookmark
	for _, b := range s.bookmarks {
		if b.SourceURL != sourceURL {
			updated = append(updated, b)
		}
	}
	s.bookmarks = updated

	// Temporarily unlock to call Save
	s.mu.Unlock()
	err := s.Save()
	s.mu.Lock()
	return err
}

func (s *Store) Toggle(b Bookmark) (bool, error) {
	if s.Has(b.SourceURL) {
		return false, s.Remove(b.SourceURL)
	}
	if b.BookmarkedAt.IsZero() {
		b.BookmarkedAt = time.Now()
	}
	return true, s.Add(b)
}

func (s *Store) Has(sourceURL string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, b := range s.bookmarks {
		if b.SourceURL == sourceURL {
			return true
		}
	}
	return false
}

func (s *Store) List() []Bookmark {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy, reversed (newest first)
	copied := make([]Bookmark, len(s.bookmarks))
	for i, b := range s.bookmarks {
		copied[len(s.bookmarks)-1-i] = b
	}
	return copied
}
