package model

import "time"

// Post represents a unified content item from any source.
type Post struct {
	ID           string
	Title        string
	URL          string // The linked article URL (may be empty for self-posts)
	SourceURL    string // Discussion page URL (HN item page, Reddit comments page)
	Author       string
	Points       int
	CommentCount int
	Source       string // Routing ID: "hn", "reddit", etc.
	SourceLabel  string // Display name: "r/programming", "HN", etc.
	SourceID     string // Original ID in the source system
	CreatedAt    time.Time
	Rank         int    // Position in source's result list (0 = top)
	Text         string // Body text for self-posts (Ask HN, Reddit self-posts)
}

// Comment represents a single comment in a discussion tree.
type Comment struct {
	ID        string
	Author    string
	Text      string // Plain text (HTML already stripped)
	CreatedAt time.Time
	Depth     int       // Nesting level (0 = top-level)
	Children  []Comment // Nested replies
	Deleted   bool
}
