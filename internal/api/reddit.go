package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
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
	http       *http.Client
	subreddits []string
}

// NewRedditClient creates a new Reddit API client.
func NewRedditClient(subreddits []string) *RedditClient {
	return &RedditClient{
		http:       &http.Client{Timeout: 15 * time.Second},
		subreddits: subreddits,
	}
}

// ID returns "reddit".
func (c *RedditClient) ID() string { return "reddit" }

// Name returns "Reddit".
func (c *RedditClient) Name() string { return "Reddit" }

// SortOptions returns valid sorts for Reddit.
func (c *RedditClient) SortOptions() []string {
	return []string{"hot", "new", "top", "rising"}
}

// FetchPosts fetches posts from all configured subreddits.
func (c *RedditClient) FetchPosts(sort string, limit int) ([]model.Post, error) {
	var allPosts []model.Post
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, sub := range c.subreddits {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			posts, err := c.fetchSubredditPosts(s, sort, limit)
			if err != nil {
				return
			}
			mu.Lock()
			allPosts = append(allPosts, posts...)
			mu.Unlock()
		}(sub)
	}
	wg.Wait()
	return allPosts, nil
}

// FetchComments fetches the comment tree for a Reddit post.
func (c *RedditClient) FetchComments(post model.Post, maxDepth int) ([]model.Comment, error) {
	// SourceID for Reddit is the JSON comment URL
	resp, err := c.redditGet(post.SourceID)
	if err != nil {
		return nil, fmt.Errorf("fetching comments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("comments returned status %d", resp.StatusCode)
	}

	var listings []redditListing
	if err := json.NewDecoder(resp.Body).Decode(&listings); err != nil {
		return nil, fmt.Errorf("decoding comments: %w", err)
	}

	if len(listings) < 2 {
		return nil, nil
	}

	return parseRedditComments(listings[1].Data.Children, 0, maxDepth), nil
}

func (c *RedditClient) redditGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", redditUA)
	return c.http.Do(req)
}

func (c *RedditClient) fetchSubredditPosts(subreddit, sort string, limit int) ([]model.Post, error) {
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
	for i, child := range listing.Data.Children {
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
			ID:           rp.ID,
			Title:        rp.Title,
			URL:          articleURL,
			SourceURL:    sourceURL,
			Author:       rp.Author,
			Points:       rp.Score,
			CommentCount: rp.NumComments,
			Source:       "reddit",
			SourceLabel:  fmt.Sprintf("r/%s", rp.Subreddit),
			SourceID:     commentURL,
			CreatedAt:    time.Unix(int64(rp.CreatedUTC), 0),
			Rank:         i,
			Text:         cleanRedditBody(rp.Selftext),
		})
	}
	return posts, nil
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
			ID:        rc.ID,
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
