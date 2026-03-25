package api

import (
	"encoding/json"
	"fmt"
	"net/http"
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

// GetPosts fetches N stories from HN using the given sort (top, new, best).
func (c *HNClient) GetPosts(sort string, limit int) ([]model.Post, error) {
	endpoint := hnTopStories
	switch sort {
	case "new":
		endpoint = hnNewStories
	case "best":
		endpoint = hnBestStories
	}
	return c.fetchStories(endpoint, limit)
}

// GetTopPosts fetches the top N stories from HN (kept for backwards compat).
func (c *HNClient) GetTopPosts(limit int) ([]model.Post, error) {
	return c.fetchStories(hnTopStories, limit)
}

func (c *HNClient) fetchStories(endpoint string, limit int) ([]model.Post, error) {
	// 1. Fetch story IDs
	resp, err := c.http.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetching top stories: %w", err)
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
				ID:           item.ID,
				Title:        item.Title,
				URL:          item.URL,
				SourceURL:    sourceURL,
				Author:       item.By,
				Points:       item.Score,
				CommentCount: item.Descendants,
				Source:       "HN",
				SourceID:     fmt.Sprintf("%d", item.ID),
				CreatedAt:    time.Unix(item.Time, 0),
				CommentIDs:   item.Kids,
				Text:         StripHTML(item.Text),
			}
		}(i, id)
	}
	wg.Wait()

	// Filter out zero-value posts (deleted/dead items left a zero slot)
	var result []model.Post
	for _, p := range posts {
		if p.ID != 0 {
			result = append(result, p)
		}
	}
	return result, nil
}

// GetComments fetches the comment tree for a post up to maxDepth levels deep.
func (c *HNClient) GetComments(ids []int, maxDepth int) ([]model.Comment, error) {
	return c.fetchCommentTree(ids, 0, maxDepth)
}

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
				ID:        item.ID,
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
