package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"technews-tui/internal/model"
)

const (
	hnBaseURL     = "https://hacker-news.firebaseio.com/v0"
	hnItemURL     = hnBaseURL + "/item/%d.json"
	hnTopStories  = hnBaseURL + "/topstories.json"
	hnNewStories  = hnBaseURL + "/newstories.json"
	hnBestStories = hnBaseURL + "/beststories.json"
)

// hnItem is the raw JSON shape from the HN API.
type hnItem struct {
	ID          int    `json:"id"`
	By          string `json:"by"`
	Descendants int    `json:"descendants"`
	Kids        []int  `json:"kids"`
	Score       int    `json:"score"`
	Time        int64  `json:"time"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	Text        string `json:"text"`
	Deleted     bool   `json:"deleted"`
	Dead        bool   `json:"dead"`
}

// HNClient fetches data from the Hacker News Firebase API.
type HNClient struct {
	http *http.Client
}

// NewHNClient creates a new HN API client.
func NewHNClient() *HNClient {
	return &HNClient{
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// ID returns "hn".
func (c *HNClient) ID() string { return "hn" }

// Name returns "HN".
func (c *HNClient) Name() string { return "HN" }

// SortOptions returns valid sorts for HN.
func (c *HNClient) SortOptions() []string {
	return []string{"top", "new", "best"}
}

// FetchPosts fetches stories from HN using the given sort (top, new, best).
func (c *HNClient) FetchPosts(sort string, limit int) ([]model.Post, error) {
	endpoint := hnTopStories
	switch sort {
	case "new":
		endpoint = hnNewStories
	case "best":
		endpoint = hnBestStories
	}
	return c.fetchStories(endpoint, limit)
}

// FetchComments fetches the comment tree for a post.
func (c *HNClient) FetchComments(post model.Post, maxDepth int) ([]model.Comment, error) {
	id, err := strconv.Atoi(post.SourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid HN ID: %s", post.SourceID)
	}
	item, err := c.fetchItem(id)
	if err != nil {
		return nil, err
	}
	return c.fetchCommentTree(item.Kids, 0, maxDepth)
}

func (c *HNClient) fetchStories(endpoint string, limit int) ([]model.Post, error) {
	// 1. Fetch story IDs
	resp, err := c.http.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetching stories: %w", err)
	}
	defer resp.Body.Close()

	var ids []int
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return nil, fmt.Errorf("decoding story IDs: %w", err)
	}

	if len(ids) > limit {
		ids = ids[:limit]
	}

	// 2. Fetch each story concurrently, preserving order
	posts := make([]model.Post, len(ids))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	sem := make(chan struct{}, 20) // cap concurrency at 20

	for i, id := range ids {
		wg.Add(1)
		go func(idx, id int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			item, err := c.fetchItem(id)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}
			if item.Deleted || item.Dead {
				return
			}

			sourceURL := fmt.Sprintf("https://news.ycombinator.com/item?id=%d", item.ID)
			posts[idx] = model.Post{
				ID:           fmt.Sprintf("%d", item.ID),
				Title:        item.Title,
				URL:          item.URL,
				SourceURL:    sourceURL,
				Author:       item.By,
				Points:       item.Score,
				CommentCount: item.Descendants,
				Source:       "hn",
				SourceID:     fmt.Sprintf("%d", item.ID),
				CreatedAt:    time.Unix(item.Time, 0),
				Rank:         idx,
				Text:         StripHTML(item.Text),
			}
		}(i, id)
	}
	wg.Wait()

	// Filter out zero-value posts (deleted/dead items left a zero slot)
	var result []model.Post
	for _, p := range posts {
		if p.ID != "" {
			result = append(result, p)
		}
	}
	return result, nil
}

// fetchCommentTree fetches the comment tree for a list of child IDs.
func (c *HNClient) fetchCommentTree(ids []int, depth, maxDepth int) ([]model.Comment, error) {
	if depth >= maxDepth || len(ids) == 0 {
		return nil, nil
	}

	comments := make([]model.Comment, len(ids))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20)

	for i, id := range ids {
		wg.Add(1)
		go func(idx, id int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			item, err := c.fetchItem(id)
			if err != nil || item.Deleted || item.Dead {
				comments[idx] = model.Comment{Deleted: true}
				return
			}

			children, _ := c.fetchCommentTree(item.Kids, depth+1, maxDepth)

			comments[idx] = model.Comment{
				ID:        fmt.Sprintf("%d", item.ID),
				Author:    item.By,
				Text:      StripHTML(item.Text),
				CreatedAt: time.Unix(item.Time, 0),
				Depth:     depth,
				Children:  children,
			}
		}(i, id)
	}
	wg.Wait()

	// Filter out deleted/error comments
	var result []model.Comment
	for _, c := range comments {
		if !c.Deleted {
			result = append(result, c)
		}
	}
	return result, nil
}

func (c *HNClient) fetchItem(id int) (*hnItem, error) {
	resp, err := c.http.Get(fmt.Sprintf(hnItemURL, id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var item hnItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}
	return &item, nil
}
