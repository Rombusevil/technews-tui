package ui

import (
	"strings"
	"testing"
	"time"

	"technews-tui/internal/model"
)

func TestHighlightMatchesWrapsFound(t *testing.T) {
	got := highlightMatches("Hello world", "world", "[", "]")
	if !strings.Contains(got, "[world]") {
		t.Fatalf("expected highlight wrap, got: %q", got)
	}
}

func TestHighlightMatchesCaseInsensitive(t *testing.T) {
	got := highlightMatches("Hello World", "world", "[", "]")
	if !strings.Contains(got, "[World]") {
		t.Fatalf("expected case-insensitive match preserving original case, got: %q", got)
	}
}

func TestHighlightMatchesNoMatchUnchanged(t *testing.T) {
	input := "Hello world"
	got := highlightMatches(input, "xyz", "[", "]")
	if got != input {
		t.Fatalf("expected unchanged string when no match, got: %q", got)
	}
}

func TestComputeMatchesFindsCorrectIndices(t *testing.T) {
	m := CommentModel{
		flat: []flatComment{
			{comment: model.Comment{ID: "1", Author: "alice", Text: "this is rust code", CreatedAt: time.Now()}},
			{comment: model.Comment{ID: "2", Author: "bob", Text: "hello world", CreatedAt: time.Now()}},
			{comment: model.Comment{ID: "3", Author: "carol", Text: "more rust here", CreatedAt: time.Now()}},
		},
	}
	m.computeMatches("rust")
	if len(m.matchIndices) != 2 {
		t.Fatalf("expected 2 matches, got %d: %v", len(m.matchIndices), m.matchIndices)
	}
	if m.matchIndices[0] != 0 || m.matchIndices[1] != 2 {
		t.Fatalf("expected indices [0, 2], got %v", m.matchIndices)
	}
}

func TestComputeMatchesMatchesAuthor(t *testing.T) {
	m := CommentModel{
		flat: []flatComment{
			{comment: model.Comment{ID: "1", Author: "alice", Text: "hello", CreatedAt: time.Now()}},
			{comment: model.Comment{ID: "2", Author: "bob", Text: "world", CreatedAt: time.Now()}},
		},
	}
	m.computeMatches("alice")
	if len(m.matchIndices) != 1 || m.matchIndices[0] != 0 {
		t.Fatalf("expected match on author, got %v", m.matchIndices)
	}
}
