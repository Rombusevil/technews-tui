package ui

import (
	"strings"
	"testing"
	"time"

	"technews-tui/internal/model"
)

func TestCommentModelCapsAndExpandsPostBody(t *testing.T) {
	post := model.Post{
		Title:        "Long self post",
		Author:       "alice",
		Points:       42,
		CommentCount: 7,
		CreatedAt:    time.Now(),
		Text:         strings.Repeat("word ", 300),
	}

	m := NewCommentModel(post)
	m.SetSize(80, 24)

	if got, want := m.headerHeight(), 9; got != want {
		t.Fatalf("collapsed headerHeight = %d, want %d", got, want)
	}

	header := m.headerView()
	if !strings.Contains(header, "expand post body") {
		t.Fatalf("collapsed header missing expand hint: %q", header)
	}

	m.bodyExpanded = true

	if got := m.headerHeight(); got <= 9 {
		t.Fatalf("expanded headerHeight = %d, want > 9", got)
	}

	expandedHeader := m.headerView()
	if strings.Contains(expandedHeader, "expand post body") {
		t.Fatalf("expanded header should not contain expand hint: %q", expandedHeader)
	}
}

func TestCommentModelShowsBookmarkedIndicator(t *testing.T) {
	post := model.Post{
		Title:        "Saved post",
		Author:       "alice",
		Points:       42,
		CommentCount: 7,
		CreatedAt:    time.Now(),
	}

	m := NewCommentModel(post)
	m.SetSize(80, 24)
	m.bookmarked = true

	header := m.headerView()
	if !strings.Contains(header, "★") {
		t.Fatalf("expected bookmarked header indicator, got: %q", header)
	}

	footer := m.footerView()
	if !strings.Contains(footer, "bookmarked") {
		t.Fatalf("expected bookmarked footer indicator, got: %q", footer)
	}
}
