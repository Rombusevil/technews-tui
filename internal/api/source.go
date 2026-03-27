package api

import (
	"technews-tui/internal/model"
)

// Source defines the interface for content providers.
type Source interface {
	// ID returns a unique internal identifier (e.g., "hn", "reddit").
	ID() string
	// Name returns the display name (e.g., "HN", "Reddit").
	Name() string
	// FetchPosts fetches posts from the source using the given sort.
	FetchPosts(sort string, limit int) ([]model.Post, error)
	// FetchComments fetches the comment tree for a specific post.
	FetchComments(post model.Post, maxDepth int) ([]model.Comment, error)
	// SortOptions returns the valid sort options for this source.
	SortOptions() []string
}
