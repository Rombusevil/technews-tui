package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	helpOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(1, 2)

	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6600")).
			MarginBottom(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6600")).
			Width(18)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)

type helpEntry struct {
	key  string
	desc string
}

func renderHelp(title string, entries []helpEntry, width, height int) string {
	var b strings.Builder
	b.WriteString(helpTitleStyle.Render(title) + "\n")
	for _, e := range entries {
		b.WriteString(helpKeyStyle.Render(e.key) + helpDescStyle.Render(e.desc) + "\n")
	}
	content := helpOverlayStyle.Render(b.String())
	// Center the overlay
	lines := strings.Split(content, "\n")
	boxH := len(lines)
	boxW := 0
	for _, l := range lines {
		if len([]rune(l)) > boxW {
			boxW = len([]rune(l))
		}
	}
	topPad := (height - boxH) / 2
	leftPad := (width - boxW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	if topPad < 0 {
		topPad = 0
	}
	prefix := strings.Repeat(" ", leftPad)
	var out strings.Builder
	out.WriteString(strings.Repeat("\n", topPad))
	for _, l := range lines {
		out.WriteString(prefix + l + "\n")
	}
	return out.String()
}

var listHelpEntries = []helpEntry{
	{"↑/↓  j/k", "navigate"},
	{"enter", "view comments"},
	{"o", "open in browser"},
	{"c", "open comments link"},
	{"b", "toggle bookmark"},
	{"B", "view bookmarks"},
	{"tab", "cycle source filter"},
	{"r", "refresh"},
	{"s", "settings"},
	{"q / ctrl+c", "quit"},
	{"?", "close help"},
}

var commentHelpEntries = []helpEntry{
	{"↑/↓  j/k", "navigate comments"},
	{"ctrl+u / ctrl+d", "half page up/down"},
	{"enter / space", "fold/unfold thread"},
	{"e", "expand/collapse post body"},
	{"b", "toggle bookmark"},
	{"o", "open in browser"},
	{"c", "open comments link"},
	{"/", "search comments"},
	{"n / N", "next / prev match"},
	{"esc", "clear search / back"},
	{"?", "close help"},
}

var settingsHelpEntries = []helpEntry{
	{"↑/↓  j/k", "navigate"},
	{"a", "add subreddit"},
	{"d / x", "delete subreddit"},
	{"t", "cycle sort order"},
	{"esc", "save & back"},
	{"?", "close help"},
}

var bookmarkHelpEntries = []helpEntry{
	{"↑/↓  j/k", "navigate"},
	{"o", "open in browser"},
	{"c", "open comments link"},
	{"d", "remove bookmark"},
	{"esc / q", "back to list"},
	{"?", "close help"},
}
