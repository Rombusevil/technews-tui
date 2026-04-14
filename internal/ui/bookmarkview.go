package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"technews-tui/internal/bookmark"
	"technews-tui/internal/browser"
	"technews-tui/internal/config"
)

type bookmarkDoneMsg struct{}

type bookmarkItem struct {
	bookmark bookmark.Bookmark
}

func (i bookmarkItem) Title() string       { return i.bookmark.Title }
func (i bookmarkItem) FilterValue() string { return i.bookmark.Title }
func (i bookmarkItem) Description() string {
	age := timeAgo(i.bookmark.BookmarkedAt)
	label := i.bookmark.Source
	if i.bookmark.SourceLabel != "" {
		label = i.bookmark.SourceLabel
	}
	return fmt.Sprintf("%s  ▲ %d  %s  %d comments  ★ %s",
		sourceTagStyle.Render(label), i.bookmark.Points, i.bookmark.Author, i.bookmark.CommentCount, age)
}

type BookmarkModel struct {
	list   list.Model
	store  *bookmark.Store
	width  int
	height int
	cfg    *config.Config
}

func NewBookmarkModel(store *bookmark.Store, cfg *config.Config) BookmarkModel {
	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Bookmarks"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle

	m := BookmarkModel{list: l, store: store, cfg: cfg}
	m.reloadItems()
	return m
}

func (m *BookmarkModel) reloadItems() {
	if m.store == nil {
		return
	}
	bookmarks := m.store.List()
	items := make([]list.Item, len(bookmarks))
	for i, b := range bookmarks {
		items[i] = bookmarkItem{bookmark: b}
	}
	m.list.SetItems(items)
}

func (m *BookmarkModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.list.SetSize(w, h)
}

func (m BookmarkModel) SelectedBookmark() *bookmark.Bookmark {
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}
	bi := item.(bookmarkItem)
	return &bi.bookmark
}

func (m BookmarkModel) Init() tea.Cmd {
	return nil
}

func (m BookmarkModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't intercept keys if filter is active
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, keys.Quit), key.Matches(msg, keys.Back):
			return m, func() tea.Msg { return bookmarkDoneMsg{} }

		case key.Matches(msg, keys.Open):
			if b := m.SelectedBookmark(); b != nil {
				url := b.URL
				if url == "" {
					url = b.SourceURL
				}
				browser.Open(m.cfg.Browser, url) //nolint:errcheck
			}
			return m, nil

		case key.Matches(msg, keys.Comments):
			if b := m.SelectedBookmark(); b != nil {
				browser.Open(m.cfg.Browser, b.SourceURL) //nolint:errcheck
			}
			return m, nil

		case key.Matches(msg, keys.Delete):
			if b := m.SelectedBookmark(); b != nil {
				_ = m.store.Remove(b.SourceURL)
				m.reloadItems()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m BookmarkModel) View() string {
	return m.list.View()
}
