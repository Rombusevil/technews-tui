package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"technews-tui/internal/config"
)

// settingsDoneMsg signals the root model that the user is done editing settings.
type settingsDoneMsg struct {
	subreddits []string
	redditSort string
	hnSort     string
}

// Section identifies which part of settings the cursor is in.
type settingsSection int

const (
	sectionSubreddits settingsSection = iota
	sectionRedditSort
	sectionHNSort
)

// SettingsModel manages the subreddit + sort configuration view.
type SettingsModel struct {
	subreddits []string
	redditSort string
	hnSort     string
	section    settingsSection // which section is focused
	cursor     int             // subreddit list cursor (only used in sectionSubreddits)
	input      textinput.Model
	adding     bool // true = text input active
	width      int
	height     int
}

// NewSettingsModel creates a settings view pre-populated with current config.
func NewSettingsModel(subreddits []string, redditSort, hnSort string) SettingsModel {
	subs := make([]string, len(subreddits))
	copy(subs, subreddits)

	ti := textinput.New()
	ti.Placeholder = "subreddit name"
	ti.CharLimit = 50
	ti.Width = 30

	return SettingsModel{
		subreddits: subs,
		redditSort: redditSort,
		hnSort:     hnSort,
		input:      ti,
	}
}

func (m *SettingsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.adding {
			return m.updateInputMode(msg)
		}
		return m.updateBrowseMode(msg)
	}

	if m.adding {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m SettingsModel) updateBrowseMode(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up):
		m.moveCursorUp()
	case key.Matches(msg, keys.Down):
		m.moveCursorDown()
	case key.Matches(msg, keys.Add):
		if m.section == sectionSubreddits {
			m.adding = true
			m.input.SetValue("")
			m.input.Focus()
			return m, textinput.Blink
		}
	case key.Matches(msg, keys.Delete):
		if m.section == sectionSubreddits && len(m.subreddits) > 0 && m.cursor < len(m.subreddits) {
			m.subreddits = append(m.subreddits[:m.cursor], m.subreddits[m.cursor+1:]...)
			if m.cursor >= len(m.subreddits) && m.cursor > 0 {
				m.cursor--
			}
		}
	case msg.String() == "t":
		// Cycle sort options
		switch m.section {
		case sectionRedditSort:
			m.redditSort = cycleNext(config.RedditSorts, m.redditSort)
		case sectionHNSort:
			m.hnSort = cycleNext(config.HNSorts, m.hnSort)
		}
	case key.Matches(msg, keys.Back):
		return m, func() tea.Msg {
			return settingsDoneMsg{
				subreddits: m.subreddits,
				redditSort: m.redditSort,
				hnSort:     m.hnSort,
			}
		}
	}
	return m, nil
}

func (m *SettingsModel) moveCursorUp() {
	switch m.section {
	case sectionSubreddits:
		if m.cursor > 0 {
			m.cursor--
		} else {
			// wrap up to HN sort section
			m.section = sectionHNSort
		}
	case sectionRedditSort:
		m.section = sectionSubreddits
		if len(m.subreddits) > 0 {
			m.cursor = len(m.subreddits) - 1
		}
	case sectionHNSort:
		m.section = sectionRedditSort
	}
}

func (m *SettingsModel) moveCursorDown() {
	switch m.section {
	case sectionSubreddits:
		if m.cursor < len(m.subreddits)-1 {
			m.cursor++
		} else {
			m.section = sectionRedditSort
		}
	case sectionRedditSort:
		m.section = sectionHNSort
	case sectionHNSort:
		// wrap back to top
		m.section = sectionSubreddits
		m.cursor = 0
	}
}

func (m SettingsModel) updateInputMode(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		val := strings.TrimSpace(m.input.Value())
		if val != "" {
			val = strings.TrimPrefix(val, "r/")
			val = strings.TrimPrefix(val, "/r/")
			m.subreddits = append(m.subreddits, val)
			m.cursor = len(m.subreddits) - 1
		}
		m.adding = false
		m.input.Blur()
		return m, nil
	case "esc":
		m.adding = false
		m.input.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m SettingsModel) View() string {
	var b strings.Builder

	settingsTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF6600")).
		Padding(0, 1)

	sectionLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1)

	// --- Subreddits section ---
	b.WriteString(settingsTitleStyle.Render("Subreddits") + "\n\n")

	if len(m.subreddits) == 0 {
		b.WriteString(statusBarStyle.Render("  No subreddits. Press 'a' to add one.") + "\n")
	} else {
		for i, sub := range m.subreddits {
			selected := m.section == sectionSubreddits && i == m.cursor
			cur := "  "
			if selected {
				cur = selectedItemStyle.Render("> ")
			}
			name := fmt.Sprintf("r/%s", sub)
			if selected {
				name = lipgloss.NewStyle().Bold(true).Render(name)
			}
			b.WriteString(cur + name + "\n")
		}
	}

	b.WriteString("\n")

	// --- Add input ---
	if m.adding {
		b.WriteString("  Add subreddit: " + m.input.View() + "\n\n")
	}

	// --- Reddit sort section ---
	redditSelected := m.section == sectionRedditSort && !m.adding
	redditCur := "  "
	if redditSelected {
		redditCur = selectedItemStyle.Render("> ")
	}
	redditLabel := sectionLabelStyle.Render("Reddit sort:")
	redditVal := lipgloss.NewStyle().Bold(redditSelected).Render(m.redditSort)
	hint := ""
	if redditSelected {
		hint = statusBarStyle.Render("  (t to cycle)")
	}
	b.WriteString(fmt.Sprintf("%s%s %s%s\n", redditCur, redditLabel, redditVal, hint))

	// --- HN sort section ---
	hnSelected := m.section == sectionHNSort && !m.adding
	hnCur := "  "
	if hnSelected {
		hnCur = selectedItemStyle.Render("> ")
	}
	hnLabel := sectionLabelStyle.Render("HN sort:    ")
	hnVal := lipgloss.NewStyle().Bold(hnSelected).Render(m.hnSort)
	hint2 := ""
	if hnSelected {
		hint2 = statusBarStyle.Render("  (t to cycle)")
	}
	b.WriteString(fmt.Sprintf("%s%s %s%s\n", hnCur, hnLabel, hnVal, hint2))

	b.WriteString("\n")

	// --- Footer ---
	if m.adding {
		b.WriteString(statusBarStyle.Render("enter confirm • esc cancel"))
	} else {
		b.WriteString(statusBarStyle.Render("j/k navigate • a add • d delete • t cycle sort • esc save & back"))
	}

	return b.String()
}

// cycleNext returns the next item in the list after current, wrapping around.
func cycleNext(list []string, current string) string {
	for i, v := range list {
		if v == current {
			return list[(i+1)%len(list)]
		}
	}
	return list[0]
}
