package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"technews-tui/internal/model"
)

type lemmyPost struct {
	Post struct {
		ID        int       `json:"id"`
		Name      string    `json:"name"`
		URL       string    `json:"url"`
		Body      string    `json:"body"`
		ApID      string    `json:"ap_id"`
		Published time.Time `json:"published"`
	} `json:"post"`
	Creator struct {
		Name string `json:"name"`
	} `json:"creator"`
	Counts struct {
		Score    int `json:"score"`
		Comments int `json:"comments"`
	} `json:"counts"`
}

type lemmyPostsResponse struct {
	Posts []lemmyPost `json:"posts"`
}

type lemmyComment struct {
	Comment struct {
		ID        int       `json:"id"`
		CreatorID int       `json:"creator_id"`
		PostID    int       `json:"post_id"`
		Content   string    `json:"content"`
		Published time.Time `json:"published"`
		Path      string    `json:"path"` // e.g. "0.1.2.3"
	} `json:"comment"`
	Creator struct {
		Name string `json:"name"`
	} `json:"creator"`
}

type lemmyCommentsResponse struct {
	Comments []lemmyComment `json:"comments"`
}

type LemmyClient struct {
	http      *http.Client
	instances []string
}

func NewLemmyClient(instances []string) *LemmyClient {
	return &LemmyClient{
		http:      &http.Client{Timeout: 15 * time.Second},
		instances: instances,
	}
}

func (c *LemmyClient) ID() string   { return "lemmy" }
func (c *LemmyClient) Name() string { return "Lemmy" }
func (c *LemmyClient) SortOptions() []string {
	return []string{"Hot", "New", "Active", "TopDay", "TopWeek", "TopAll"}
}

func (c *LemmyClient) FetchPosts(sort string, limit int) ([]model.Post, error) {
	if sort == "" {
		sort = "Hot"
	}

	var allPosts []model.Post
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, instance := range c.instances {
		wg.Add(1)
		go func(inst string) {
			defer wg.Done()
			posts, err := c.fetchInstancePosts(inst, sort, limit)
			if err != nil {
				return
			}
			mu.Lock()
			allPosts = append(allPosts, posts...)
			mu.Unlock()
		}(instance)
	}

	wg.Wait()
	return allPosts, nil
}

func (c *LemmyClient) fetchInstancePosts(instance, sort string, limit int) ([]model.Post, error) {
	url := fmt.Sprintf("https://%s/api/v3/post/list?sort=%s&limit=%d", instance, sort, limit)

	// Shorter timeout per instance
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lemmy instance %s returned status %d", instance, resp.StatusCode)
	}

	var res lemmyPostsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	var posts []model.Post
	for i, p := range res.Posts {
		articleURL := p.Post.URL
		sourceURL := p.Post.ApID
		if sourceURL == "" {
			sourceURL = fmt.Sprintf("https://%s/post/%d", instance, p.Post.ID)
		}

		createdAt := p.Post.Published
		if createdAt.IsZero() {
			createdAt = time.Now()
		}

		posts = append(posts, model.Post{
			ID:           fmt.Sprintf("%s|%d", instance, p.Post.ID),
			Title:        p.Post.Name,
			URL:          articleURL,
			SourceURL:    sourceURL,
			Author:       p.Creator.Name,
			Points:       p.Counts.Score,
			CommentCount: p.Counts.Comments,
			Source:       "lemmy",
			SourceID:     fmt.Sprintf("%s|%d", instance, p.Post.ID),
			CreatedAt:    createdAt,
			Rank:         i,
			Text:         StripHTML(p.Post.Body),
		})
	}
	return posts, nil
}

func (c *LemmyClient) FetchComments(post model.Post, maxDepth int) ([]model.Comment, error) {
	parts := strings.Split(post.SourceID, "|")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid Lemmy SourceID: %s", post.SourceID)
	}
	instance := parts[0]
	postID := parts[1]

	url := fmt.Sprintf("https://%s/api/v3/comment/list?post_id=%s&sort=Hot&limit=50", instance, postID)
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lemmy instance %s returned status %d", instance, resp.StatusCode)
	}

	var res lemmyCommentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return buildLemmyTree(res.Comments, maxDepth), nil
}

type node struct {
	comment  *model.Comment
	parentID int
}

func buildLemmyTree(flat []lemmyComment, maxDepth int) []model.Comment {
	nodes := make(map[int]*node)
	for _, lc := range flat {
		pathParts := strings.Split(lc.Comment.Path, ".")
		depth := len(pathParts) - 2
		if depth >= maxDepth {
			continue
		}

		n := &node{
			comment: &model.Comment{
				ID:        fmt.Sprintf("%d", lc.Comment.ID),
				Author:    lc.Creator.Name,
				Text:      StripHTML(lc.Comment.Content),
				CreatedAt: lc.Comment.Published,
				Depth:     depth,
			},
		}
		if len(pathParts) > 2 {
			fmt.Sscanf(pathParts[len(pathParts)-2], "%d", &n.parentID)
		}
		nodes[lc.Comment.ID] = n
	}

	var keys []int
	for k := range nodes {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, id := range keys {
		n := nodes[id]
		if n.parentID != 0 {
			if p, ok := nodes[n.parentID]; ok {
				p.comment.Children = append(p.comment.Children, *n.comment)
			}
		}
	}

	var result []model.Comment
	for _, id := range keys {
		n := nodes[id]
		// path for this id
		path := ""
		for _, lc := range flat {
			if lc.Comment.ID == id {
				path = lc.Comment.Path
				break
			}
		}
		if len(strings.Split(path, ".")) == 2 {
			result = append(result, *n.comment)
		}
	}

	return result
}
