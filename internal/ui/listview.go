package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"technews-tui/internal/bookmark"
	"technews-tui/internal/model"
)

// postItem wraps model.Post to satisfy list.Item and list.DefaultItem.
type postItem struct {
	post       model.Post
	bookmarked bool
}

func (i postItem) Title() string       { return i.post.Title }
func (i postItem) FilterValue() string { return i.post.Title }
func (i postItem) Description() string {
	age := timeAgo(i.post.CreatedAt)
	label := i.post.Source
	if i.post.SourceLabel != "" {
		label = i.post.SourceLabel
	}
	star := ""
	if i.bookmarked {
		star = "★ "
	}
	return fmt.Sprintf("%s%s  ▲ %d  %s  %d comments  %s",
		star, sourceTagStyle.Render(label), i.post.Points, i.post.Author, i.post.CommentCount, age)
}

type SortInfo struct {
	Name string
	Sort string
}

// ListModel manages the story list view.
type ListModel struct {
	list          list.Model
	posts         []model.Post
	bookmarkStore *bookmark.Store
	width         int
	height        int
	baseTitle     string
	sorts         []SortInfo
}

func NewListModel(store *bookmark.Store) ListModel {
	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle() // styles embedded in Title string directly
	m := ListModel{list: l, baseTitle: "Tech News", bookmarkStore: store}
	m.updateTitle()
	return m
}

func (m *ListModel) updateTitle() {
	title := titleStyle.Render(m.baseTitle)
	if len(m.sorts) > 0 {
		var lines []string
		for _, s := range m.sorts {
			lines = append(lines, fmt.Sprintf("%s: %s", s.Name, s.Sort))
		}
		sortLine := strings.Join(lines, "  ·  ")
		title += "\n" + subtitleStyle.Render(sortLine)
	}
	m.list.Title = title
}

func (m *ListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.list.SetSize(w, h)
}

func (m *ListModel) SetTitle(title string) {
	m.baseTitle = title
	m.updateTitle()
}

func (m *ListModel) SetSortInfo(sorts []SortInfo) {
	m.sorts = sorts
	m.updateTitle()
}

func (m *ListModel) SetPosts(posts []model.Post) {
	m.posts = posts
	items := make([]list.Item, len(posts))
	for i, p := range posts {
		bookmarked := false
		if m.bookmarkStore != nil {
			bookmarked = m.bookmarkStore.Has(p.SourceURL)
		}
		items[i] = postItem{post: p, bookmarked: bookmarked}
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
