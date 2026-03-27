package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"technews-tui/internal/model"
)

const (
	devtoBaseURL = "https://dev.to/api"
)

type devtoArticle struct {
	ID                   int       `json:"id"`
	Title                string    `json:"title"`
	Description          string    `json:"description"`
	URL                  string    `json:"url"`
	CommentsCount        int       `json:"comments_count"`
	PublicReactionsCount int       `json:"public_reactions_count"`
	PublishedAt          time.Time `json:"published_at"`
	User                 struct {
		Name string `json:"name"`
	} `json:"user"`
}

type devtoComment struct {
	IDCode    string    `json:"id_code"`
	BodyHTML  string    `json:"body_html"`
	CreatedAt time.Time `json:"created_at"`
	User      struct {
		Name string `json:"name"`
	} `json:"user"`
	Children []devtoComment `json:"children"`
}

type DevToClient struct {
	http *http.Client
}

func NewDevToClient() *DevToClient {
	return &DevToClient{
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *DevToClient) ID() string   { return "devto" }
func (c *DevToClient) Name() string { return "Dev.to" }
func (c *DevToClient) SortOptions() []string {
	return []string{"default", "latest", "top", "rising"}
}

func (c *DevToClient) FetchPosts(sort string, limit int) ([]model.Post, error) {
	url := fmt.Sprintf("%s/articles?per_page=%d", devtoBaseURL, limit)
	switch sort {
	case "latest":
		url = fmt.Sprintf("%s/articles/latest?per_page=%d", devtoBaseURL, limit)
	case "top":
		url = fmt.Sprintf("%s/articles?top=7&per_page=%d", devtoBaseURL, limit)
	case "rising":
		url = fmt.Sprintf("%s/articles?state=rising&per_page=%d", devtoBaseURL, limit)
	}

	resp, err := c.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dev.to returned status %d", resp.StatusCode)
	}

	var articles []devtoArticle
	if err := json.NewDecoder(resp.Body).Decode(&articles); err != nil {
		return nil, err
	}

	var posts []model.Post
	for i, a := range articles {
		posts = append(posts, model.Post{
			ID:           fmt.Sprintf("%d", a.ID),
			Title:        a.Title,
			URL:          a.URL, // dev.to articles are the discussion page usually
			SourceURL:    a.URL,
			Author:       a.User.Name,
			Points:       a.PublicReactionsCount,
			CommentCount: a.CommentsCount,
			Source:       "devto",
			SourceID:     fmt.Sprintf("%d", a.ID),
			CreatedAt:    a.PublishedAt,
			Rank:         i,
			Text:         StripHTML(a.Description),
		})
	}
	return posts, nil
}

func (c *DevToClient) FetchComments(post model.Post, maxDepth int) ([]model.Comment, error) {
	url := fmt.Sprintf("%s/comments?a_id=%s", devtoBaseURL, post.SourceID)
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dev.to returned status %d", resp.StatusCode)
	}

	var rawComments []devtoComment
	if err := json.NewDecoder(resp.Body).Decode(&rawComments); err != nil {
		return nil, err
	}

	return buildDevToTree(rawComments, 0, maxDepth), nil
}

func buildDevToTree(raw []devtoComment, depth, maxDepth int) []model.Comment {
	if depth >= maxDepth || len(raw) == 0 {
		return nil
	}

	var result []model.Comment
	for _, rc := range raw {
		result = append(result, model.Comment{
			ID:        rc.IDCode,
			Author:    rc.User.Name,
			Text:      StripHTML(rc.BodyHTML),
			CreatedAt: rc.CreatedAt,
			Depth:     depth,
			Children:  buildDevToTree(rc.Children, depth+1, maxDepth),
		})
	}
	return result
}
