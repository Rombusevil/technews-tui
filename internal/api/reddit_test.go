package api

import "testing"

func TestRedditClient_GetSubredditPosts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	client := NewRedditClient()
	posts, err := client.GetSubredditPosts("programming", "hot", 3)
	if err != nil {
		t.Fatalf("GetSubredditPosts failed: %v", err)
	}
	if len(posts) == 0 {
		t.Fatal("expected at least one post")
	}
	p := posts[0]
	if p.Title == "" {
		t.Error("expected post to have a title")
	}
	if p.Source == "" {
		t.Error("expected post to have a source")
	}
	if p.CommentURL == "" {
		t.Error("expected post to have a comment URL")
	}
	if p.SourceURL == "" {
		t.Error("expected post to have a source URL")
	}
}

func TestRedditClient_GetComments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	client := NewRedditClient()

	// First get a post to get its comment URL
	posts, err := client.GetSubredditPosts("programming", "hot", 1)
	if err != nil {
		t.Fatalf("GetSubredditPosts failed: %v", err)
	}
	if len(posts) == 0 {
		t.Skip("no posts found to fetch comments from")
	}

	comments, err := client.GetComments(posts[0].CommentURL, 3)
	if err != nil {
		t.Fatalf("GetComments failed: %v", err)
	}
	// Comments might be empty if the post has no comments, that's OK
	t.Logf("fetched %d top-level comments", len(comments))
}
