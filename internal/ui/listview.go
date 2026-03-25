package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"technews-tui/internal/model"
)

// postItem wraps model.Post to satisfy list.Item and list.DefaultItem.
type postItem struct {
	post model.Post
}

func (i postItem) Title() string       { return i.post.Title }
func (i postItem) FilterValue() string { return i.post.Title }
func (i postItem) Description() string {
	age := timeAgo(i.post.CreatedAt)
	return fmt.Sprintf("%s  ▲ %d  %s  %d comments  %s",
		sourceTagStyle.Render(i.post.Source), i.post.Points, i.post.Author, i.post.CommentCount, age)
}

// ListModel manages the story list view.
type ListModel struct {
	list   list.Model
	posts  []model.Post
	width  int
	height int
}

func NewListModel() ListModel {
	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Tech News"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	return ListModel{list: l}
}

func (m *ListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.list.SetSize(w, h)
}

func (m *ListModel) SetTitle(title string) {
	m.list.Title = title
}

func (m *ListModel) SetPosts(posts []model.Post) {
	m.posts = posts
	items := make([]list.Item, len(posts))
	for i, p := range posts {
		items[i] = postItem{post: p}
	}
	m.list.SetItems(items)
}

// SelectedPost returns a pointer to the currently highlighted post, or nil.
func (m ListModel) SelectedPost() *model.Post {
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}
	pi := item.(postItem)
	return &pi.post
}

func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ListModel) View() string {
	return m.list.View()
}

// timeAgo returns a human-readable relative time string.
func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
