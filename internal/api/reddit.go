package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"technews-tui/internal/model"
)

const (
	redditBaseURL = "https://www.reddit.com"
	redditUA      = "go:technews-tui:v0.1 (by /u/technews-tui-app)"
)

// Reddit JSON API types

type redditListing struct {
	Data struct {
		Children []redditChild `json:"children"`
	} `json:"data"`
}

type redditChild struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

type redditPost struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Permalink   string  `json:"permalink"`
	Author      string  `json:"author"`
	Score       int     `json:"score"`
	NumComments int     `json:"num_comments"`
	CreatedUTC  float64 `json:"created_utc"`
	Selftext    string  `json:"selftext"`
	IsSelf      bool    `json:"is_self"`
	Subreddit   string  `json:"subreddit"`
	Over18      bool    `json:"over_18"`
}

type redditComment struct {
	ID         string          `json:"id"`
	Author     string          `json:"author"`
	Body       string          `json:"body"`
	Score      int             `json:"score"`
	CreatedUTC float64         `json:"created_utc"`
	Depth      int             `json:"depth"`
	Replies    json.RawMessage `json:"replies"` // can be "" or a Listing
}

// RedditClient fetches data from Reddit's public JSON API.
type RedditClient struct {
	http *http.Client
}

// NewRedditClient creates a new Reddit API client.
func NewRedditClient() *RedditClient {
	return &RedditClient{
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *RedditClient) redditGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", redditUA)
	return c.http.Do(req)
}

// GetSubredditPosts fetches posts from a subreddit using the given sort (hot, new, top, rising).
func (c *RedditClient) GetSubredditPosts(subreddit, sort string, limit int) ([]model.Post, error) {
	if sort == "" {
		sort = "hot"
	}
	url := fmt.Sprintf("%s/r/%s/%s.json?limit=%d&raw_json=1", redditBaseURL, subreddit, sort, limit)
	resp, err := c.redditGet(url)
	if err != nil {
		return nil, fmt.Errorf("fetching r/%s: %w", subreddit, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("r/%s returned status %d", subreddit, resp.StatusCode)
	}

	var listing redditListing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, fmt.Errorf("decoding r/%s: %w", subreddit, err)
	}

	var posts []model.Post
	for _, child := range listing.Data.Children {
		if child.Kind != "t3" {
			continue
		}
		var rp redditPost
		if err := json.Unmarshal(child.Data, &rp); err != nil {
			continue
		}
		if rp.Over18 {
			continue
		}

		articleURL := rp.URL
		if rp.IsSelf {
			articleURL = ""
		}
		sourceURL := fmt.Sprintf("https://www.reddit.com%s", rp.Permalink)
		commentURL := fmt.Sprintf("%s/r/%s/comments/%s.json?limit=100&depth=3&raw_json=1",
			redditBaseURL, rp.Subreddit, rp.ID)

		posts = append(posts, model.Post{
			ID:           0, // Reddit uses string IDs
			Title:        rp.Title,
			URL:          articleURL,
			SourceURL:    sourceURL,
			Author:       rp.Author,
			Points:       rp.Score,
			CommentCount: rp.NumComments,
			Source:       fmt.Sprintf("r/%s", rp.Subreddit),
			SourceID:     rp.ID,
			CreatedAt:    time.Unix(int64(rp.CreatedUTC), 0),
			CommentURL:   commentURL,
			Text:         cleanRedditBody(rp.Selftext),
		})
	}
	return posts, nil
}

// GetComments fetches the comment tree for a Reddit post via its comment URL.
func (c *RedditClient) GetComments(commentURL string, maxDepth int) ([]model.Comment, error) {
	resp, err := c.redditGet(commentURL)
	if err != nil {
		return nil, fmt.Errorf("fetching comments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("comments returned status %d", resp.StatusCode)
	}

	// Reddit returns a 2-element JSON array: [post_listing, comments_listing]
	var listings []redditListing
	if err := json.NewDecoder(resp.Body).Decode(&listings); err != nil {
		return nil, fmt.Errorf("decoding comments: %w", err)
	}

	if len(listings) < 2 {
		return nil, nil
	}

	return parseRedditComments(listings[1].Data.Children, 0, maxDepth), nil
}

func parseRedditComments(children []redditChild, depth, maxDepth int) []model.Comment {
	if depth >= maxDepth {
		return nil
	}

	var comments []model.Comment
	for _, child := range children {
		if child.Kind != "t1" {
			continue // skip "more" stubs
		}

		var rc redditComment
		if err := json.Unmarshal(child.Data, &rc); err != nil {
			continue
		}

		if rc.Author == "[deleted]" {
			continue
		}

		// Parse nested replies
		var childComments []model.Comment
		if len(rc.Replies) > 0 && !isEmptyString(rc.Replies) {
			var repliesListing redditListing
			if err := json.Unmarshal(rc.Replies, &repliesListing); err == nil {
				childComments = parseRedditComments(repliesListing.Data.Children, depth+1, maxDepth)
			}
		}

		// Reddit body is markdown — strip basic formatting for terminal
		text := cleanRedditBody(rc.Body)

		comments = append(comments, model.Comment{
			ID:        0,
			Author:    rc.Author,
			Text:      text,
			CreatedAt: time.Unix(int64(rc.CreatedUTC), 0),
			Depth:     depth,
			Children:  childComments,
		})
	}
	return comments
}

// isEmptyString checks if raw JSON is just an empty string "".
func isEmptyString(raw json.RawMessage) bool {
	return len(raw) == 0 || strings.TrimSpace(string(raw)) == `""`
}

// cleanRedditBody does minimal cleanup of Reddit markdown for terminal display.
func cleanRedditBody(body string) string {
	// Reddit API with raw_json=1 returns clean text, but may still have
	// markdown links, bold, etc. Do minimal cleanup.
	body = strings.ReplaceAll(body, "&amp;", "&")
	body = strings.ReplaceAll(body, "&lt;", "<")
	body = strings.ReplaceAll(body, "&gt;", ">")
	body = strings.TrimSpace(body)
	return body
}
