package visited

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type item struct {
	SourceURL string    `json:"source_url"`
	VisitedAt time.Time `json:"visited_at"`
}

type Store struct {
	mu          sync.RWMutex
	path        string
	visitedUrls []item
}

func NewStore(path string) *Store {
	return &Store{
		path: path,
	}
}

func DefaultPath() string {
	fileName := "visited.json"
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fileName
	}
	return filepath.Join(configDir, "technews-tui", fileName)
}

func (s *Store) Load() error {
	s.mu.Lock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	err = json.Unmarshal(data, &s.visitedUrls)
	s.mu.Unlock()

	s.PurgeOld()
	return err
}

func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.visitedUrls, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0o644)
}

func (s *Store) AddOrUpdate(link string) error {
	s.mu.Lock()
	var foundLinkAt *int
	for i, cur := range s.visitedUrls {
		if cur.SourceURL == link {
			foundLinkAt = &i
			break
		}
	}
	if foundLinkAt != nil {
		s.visitedUrls[*foundLinkAt].VisitedAt = time.Now()
	} else {
		s.visitedUrls = append(s.visitedUrls, item{SourceURL: link, VisitedAt: time.Now()})
	}
	s.mu.Unlock()
	return s.Save()
}

func (s *Store) PurgeOld() {
	s.mu.Lock()
	defer s.mu.Unlock()

	daysToKeep := time.Duration(14)
	for i := len(s.visitedUrls) - 1; i >= 0; i-- {
		if s.visitedUrls[i].VisitedAt.Add(time.Hour * 24 * daysToKeep).Before(time.Now()) {
			s.visitedUrls = append(s.visitedUrls[:i], s.visitedUrls[i+1:]...)
		}
	}
}

func (s *Store) Remove(sourceURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var updated []item
	for _, b := range s.visitedUrls {
		if b.SourceURL != sourceURL {
			updated = append(updated, b)
		}
	}
	s.visitedUrls = updated

	// Temporarily unlock to call Save
	s.mu.Unlock()
	err := s.Save()
	s.mu.Lock()
	return err
}

func (s *Store) Toggle(b item) (bool, error) {
	if s.Has(b.SourceURL) {
		return false, s.Remove(b.SourceURL)
	}
	if b.VisitedAt.IsZero() {
		b.VisitedAt = time.Now()
	}
	return true, s.AddOrUpdate(b.SourceURL)
}

func (s *Store) Has(sourceURL string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, b := range s.visitedUrls {
		if b.SourceURL == sourceURL {
			return true
		}
	}
	return false
}

func (s *Store) List() []item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy, reversed (newest first)
	copied := make([]item, len(s.visitedUrls))
	for i, b := range s.visitedUrls {
		copied[len(s.visitedUrls)-1-i] = b
	}
	return copied
}
