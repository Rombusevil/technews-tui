package api

import "testing"

func TestHNClient_GetTopPosts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	client := NewHNClient()
	posts, err := client.GetTopPosts(5)
	if err != nil {
		t.Fatalf("GetTopPosts failed: %v", err)
	}
	if len(posts) == 0 {
		t.Fatal("expected at least one post")
	}
	if posts[0].Title == "" {
		t.Error("expected post to have a title")
	}
	if posts[0].Source != "HN" {
		t.Errorf("expected source HN, got %q", posts[0].Source)
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "paragraph tags become newlines",
			input:    "<p>First paragraph.</p><p>Second paragraph.</p>",
			expected: "First paragraph.\n\nSecond paragraph.",
		},
		{
			name:     "links show text and URL",
			input:    `Check <a href="https://example.com">this link</a> out`,
			expected: "Check this link (https://example.com) out",
		},
		{
			name:     "code tags preserved",
			input:    "Use <code>fmt.Println</code> for output",
			expected: "Use `fmt.Println` for output",
		},
		{
			name:     "HTML entities decoded",
			input:    "one &gt; two &amp; three &lt; four",
			expected: "one > two & three < four",
		},
		{
			name:     "HN escaped URLs decoded",
			input:    `<a href="https:&#x2F;&#x2F;example.com">link</a>`,
			expected: "link (https://example.com)",
		},
		{
			name:     "italic tags",
			input:    "this is <i>italic</i> text",
			expected: "this is italic text",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripHTML(tt.input)
			if got != tt.expected {
				t.Errorf("StripHTML(%q)\ngot:  %q\nwant: %q", tt.input, got, tt.expected)
			}
		})
	}
}
