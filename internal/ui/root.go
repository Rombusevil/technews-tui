package ui

import (
	"fmt"
	"sort"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"technews-tui/internal/api"
	"technews-tui/internal/browser"
	"technews-tui/internal/config"
	"technews-tui/internal/model"
)

type viewState int

const (
	stateList viewState = iota
	stateComments
	stateSettings
)

// --- Messages ---

type postsLoadedMsg struct{ posts []model.Post }
type commentsLoadedMsg struct{ comments []model.Comment }
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// --- Root Model ---

type RootModel struct {
	state         viewState
	listModel     ListModel
	commentModel  CommentModel
	settingsModel SettingsModel
	hnClient      *api.HNClient
	redditClient  *api.RedditClient
	cfg           *config.Config
	allPosts      []model.Post // full unfiltered set
	sourceFilter  string       // "" = all, "HN", "r/linux", etc.
	sources       []string     // unique source names for cycling
	width         int
	height        int
	err           error
	loading       bool
	showHelp      bool
}

func NewRootModel(cfg *config.Config) RootModel {
	return RootModel{
		state:        stateList,
		listModel:    NewListModel(),
		hnClient:     api.NewHNClient(),
		redditClient: api.NewRedditClient(),
		cfg:          cfg,
		loading:      true,
	}
}

func (m RootModel) Init() tea.Cmd {
	return m.fetchPostsCmd()
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.listModel.SetSize(msg.Width, msg.Height)
		m.commentModel.SetSize(msg.Width, msg.Height)
		return m, nil

	case postsLoadedMsg:
		m.loading = false
		m.allPosts = msg.posts
		m.sources = uniqueSources(msg.posts)
		m.applyFilter()
		return m, nil

	case commentsLoadedMsg:
		m.commentModel.SetComments(msg.comments)
		return m, nil

	case settingsDoneMsg:
		m.cfg.Subreddits = msg.subreddits
		m.cfg.RedditSort = msg.redditSort
		m.cfg.HNSort = msg.hnSort
		_ = config.Save(m.cfg)
		m.state = stateList
		m.loading = true
		m.sourceFilter = "" // reset filter when sources may have changed
		return m, m.fetchPostsCmd()

	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		// ctrl+c always quits
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// ? toggles help overlay in any state
		if key.Matches(msg, keys.Help) {
			m.showHelp = !m.showHelp
			return m, nil
		}
		// esc also closes help if open
		if m.showHelp && key.Matches(msg, keys.Back) {
			m.showHelp = false
			return m, nil
		}
		// block all other keys when help is showing
		if m.showHelp {
			return m, nil
		}

		switch m.state {
		case stateList:
			return m.updateList(msg)
		case stateComments:
			return m.updateComments(msg)
		case stateSettings:
			return m.updateSettings(msg)
		}
	}

	// Delegate non-key messages to the active view
	switch m.state {
	case stateList:
		var cmd tea.Cmd
		m.listModel, cmd = m.listModel.Update(msg)
		return m, cmd
	case stateComments:
		var cmd tea.Cmd
		m.commentModel, cmd = m.commentModel.Update(msg)
		return m, cmd
	case stateSettings:
		var cmd tea.Cmd
		m.settingsModel, cmd = m.settingsModel.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m RootModel) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case msg.String() == "enter":
		post := m.listModel.SelectedPost()
		if post == nil {
			return m, nil
		}
		m.state = stateComments
		m.commentModel = NewCommentModel(*post)
		m.commentModel.SetSize(m.width, m.height)
		return m, m.fetchCommentsCmd(*post)

	case key.Matches(msg, keys.Open):
		post := m.listModel.SelectedPost()
		if post == nil {
			return m, nil
		}
		url := post.URL
		if url == "" {
			url = post.SourceURL
		}
		browser.Open(url) //nolint:errcheck
		return m, nil

	case key.Matches(msg, keys.Settings):
		m.state = stateSettings
		m.settingsModel = NewSettingsModel(m.cfg.Subreddits, m.cfg.RedditSort, m.cfg.HNSort)
		m.settingsModel.SetSize(m.width, m.height)
		return m, nil

	case key.Matches(msg, keys.Filter):
		m.cycleFilter()
		return m, nil

	case key.Matches(msg, keys.Refresh):
		m.loading = true
		return m, m.fetchPostsCmd()
	}

	var cmd tea.Cmd
	m.listModel, cmd = m.listModel.Update(msg)
	return m, cmd
}

func (m RootModel) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.settingsModel, cmd = m.settingsModel.Update(msg)
	return m, cmd
}

func (m RootModel) updateComments(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Back), key.Matches(msg, keys.Quit):
		m.state = stateList
		return m, nil

	case key.Matches(msg, keys.Open):
		url := m.commentModel.post.URL
		if url == "" {
			url = m.commentModel.post.SourceURL
		}
		browser.Open(url) //nolint:errcheck
		return m, nil
	}

	var cmd tea.Cmd
	m.commentModel, cmd = m.commentModel.Update(msg)
	return m, cmd
}

func (m RootModel) View() string {
	if m.loading {
		return "\n  Loading stories..."
	}
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press r to retry.", m.err)
	}

	if m.showHelp {
		switch m.state {
		case stateComments:
			return renderHelp("Comment View", commentHelpEntries, m.width, m.height)
		case stateSettings:
			return renderHelp("Settings", settingsHelpEntries, m.width, m.height)
		default:
			return renderHelp("Tech News TUI", listHelpEntries, m.width, m.height)
		}
	}

	switch m.state {
	case stateComments:
		return m.commentModel.View()
	case stateSettings:
		return m.settingsModel.View()
	default:
		return m.listModel.View()
	}
}

// --- Commands ---

func (m RootModel) fetchPostsCmd() tea.Cmd {
	return func() tea.Msg {
		var allPosts []model.Post
		var mu sync.Mutex
		var wg sync.WaitGroup

		// Fetch HN
		hnSort := m.cfg.HNSort
		redditSort := m.cfg.RedditSort
		wg.Add(1)
		go func() {
			defer wg.Done()
			posts, err := m.hnClient.GetPosts(hnSort, 30)
			if err != nil {
				return
			}
			mu.Lock()
			allPosts = append(allPosts, posts...)
			mu.Unlock()
		}()

		// Fetch each subreddit
		for _, sub := range m.cfg.Subreddits {
			wg.Add(1)
			go func(s string) {
				defer wg.Done()
				posts, err := m.redditClient.GetSubredditPosts(s, redditSort, 25)
				if err != nil {
					return
				}
				mu.Lock()
				allPosts = append(allPosts, posts...)
				mu.Unlock()
			}(sub)
		}

		wg.Wait()

		// Sort by creation time, newest first
		sort.Slice(allPosts, func(i, j int) bool {
			return allPosts[i].CreatedAt.After(allPosts[j].CreatedAt)
		})

		return postsLoadedMsg{allPosts}
	}
}

// --- Source filter helpers ---

func (m *RootModel) applyFilter() {
	if m.sourceFilter == "" {
		m.listModel.SetPosts(m.allPosts)
		m.listModel.SetTitle("Tech News")
	} else {
		var filtered []model.Post
		for _, p := range m.allPosts {
			if p.Source == m.sourceFilter {
				filtered = append(filtered, p)
			}
		}
		m.listModel.SetPosts(filtered)
		m.listModel.SetTitle("Tech News [" + m.sourceFilter + "]")
	}
}

func (m *RootModel) cycleFilter() {
	if len(m.sources) == 0 {
		return
	}
	if m.sourceFilter == "" {
		m.sourceFilter = m.sources[0]
	} else {
		found := false
		for i, s := range m.sources {
			if s == m.sourceFilter {
				if i+1 < len(m.sources) {
					m.sourceFilter = m.sources[i+1]
				} else {
					m.sourceFilter = "" // wrap back to All
				}
				found = true
				break
			}
		}
		if !found {
			m.sourceFilter = ""
		}
	}
	m.applyFilter()
}

func uniqueSources(posts []model.Post) []string {
	seen := map[string]bool{}
	var sources []string
	for _, p := range posts {
		if !seen[p.Source] {
			seen[p.Source] = true
			sources = append(sources, p.Source)
		}
	}
	sort.Strings(sources)
	return sources
}

func (m RootModel) fetchCommentsCmd(post model.Post) tea.Cmd {
	return func() tea.Msg {
		var comments []model.Comment
		var err error

		switch {
		case post.CommentURL != "":
			// Reddit-style: fetch via URL
			comments, err = m.redditClient.GetComments(post.CommentURL, 3)
		default:
			// HN-style: fetch via comment IDs
			comments, err = m.hnClient.GetComments(post.CommentIDs, 3)
		}

		if err != nil {
			return errMsg{err}
		}
		return commentsLoadedMsg{comments}
	}
}
