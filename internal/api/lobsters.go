package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"technews-tui/internal/model"
)

const (
	lobstersBaseURL = "https://lobste.rs"
)

type lobstersPost struct {
	ShortID      string    `json:"short_id"`
	ShortIDURL   string    `json:"short_id_url"`
	CreatedAt    time.Time `json:"created_at"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	Score        int       `json:"score"`
	CommentCount int       `json:"comment_count"`
	Description  string    `json:"description"`
	CommentsURL  string    `json:"comments_url"`
	Submitter    struct {
		Username string `json:"username"`
	} `json:"submitter_user"`
	Tags []string `json:"tags"`
}

type lobstersComment struct {
	ShortID        string    `json:"short_id"`
	CreatedAt      time.Time `json:"created_at"`
	Score          int       `json:"score"`
	Comment        string    `json:"comment"`
	IndentLevel    int       `json:"indent_level"`
	CommentingUser struct {
		Username string `json:"username"`
	} `json:"commenting_user"`
}

type lobstersDetail struct {
	lobstersPost
	Comments []lobstersComment `json:"comments"`
}

type LobstersClient struct {
	http *http.Client
}

func NewLobstersClient() *LobstersClient {
	return &LobstersClient{
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *LobstersClient) ID() string   { return "lobsters" }
func (c *LobstersClient) Name() string { return "Lobsters" }
func (c *LobstersClient) SortOptions() []string {
	return []string{"hottest", "newest"}
}

func (c *LobstersClient) FetchPosts(sort string, limit int) ([]model.Post, error) {
	if sort == "" {
		sort = "hottest"
	}
	url := fmt.Sprintf("%s/%s.json", lobstersBaseURL, sort)
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lobste.rs returned status %d", resp.StatusCode)
	}

	var rawPosts []lobstersPost
	if err := json.NewDecoder(resp.Body).Decode(&rawPosts); err != nil {
		return nil, err
	}

	var posts []model.Post
	for i, p := range rawPosts {
		if i >= limit {
			break
		}
		articleURL := p.URL
		// If URL is the same as discussion, it's a self-post
		if articleURL == p.CommentsURL || articleURL == p.ShortIDURL {
			articleURL = ""
		}

		posts = append(posts, model.Post{
			ID:           p.ShortID,
			Title:        p.Title,
			URL:          articleURL,
			SourceURL:    p.ShortIDURL,
			Author:       p.Submitter.Username,
			Points:       p.Score,
			CommentCount: p.CommentCount,
			Source:       "lobsters",
			SourceID:     p.ShortID,
			CreatedAt:    p.CreatedAt,
			Rank:         i,
			Text:         StripHTML(p.Description),
		})
	}
	return posts, nil
}

func (c *LobstersClient) FetchComments(post model.Post, maxDepth int) ([]model.Comment, error) {
	url := fmt.Sprintf("%s/s/%s.json", lobstersBaseURL, post.SourceID)
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lobste.rs returned status %d", resp.StatusCode)
	}

	var detail lobstersDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, err
	}

	// Lobsters returns a flat list of comments with IndentLevel.
	// We need to convert it to a tree up to maxDepth.
	return buildLobstersTree(detail.Comments, 1, maxDepth), nil
}

func buildLobstersTree(flat []lobstersComment, targetLevel, maxDepth int) []model.Comment {
	if targetLevel > maxDepth || len(flat) == 0 {
		return nil
	}

	var result []model.Comment
	for i := 0; i < len(flat); i++ {
		c := flat[i]
		if c.IndentLevel == targetLevel {
			// Find children: all subsequent comments with IndentLevel > targetLevel
			var childrenFlat []lobstersComment
			for j := i + 1; j < len(flat); j++ {
				if flat[j].IndentLevel > targetLevel {
					childrenFlat = append(childrenFlat, flat[j])
				} else {
					break
				}
			}

			result = append(result, model.Comment{
				ID:        c.ShortID,
				Author:    c.CommentingUser.Username,
				Text:      StripHTML(c.Comment),
				CreatedAt: c.CreatedAt,
				Depth:     targetLevel - 1,
				Children:  buildLobstersTree(childrenFlat, targetLevel+1, maxDepth),
			})

			// Skip the children we just processed
			i += len(childrenFlat)
		}
	}
	return result
}
