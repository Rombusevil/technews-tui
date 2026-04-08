package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"technews-tui/internal/model"
)

var (
	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600"))
)

type flatComment struct {
	comment     model.Comment
	hasChildren bool
	childCount  int
	collapsed   bool
}

type searchMode int

const (
	searchModeNone   searchMode = iota // normal navigation
	searchModeTyping                   // input active
	searchModeActive                   // query submitted, highlights visible
)

type CommentModel struct {
	post         model.Post
	allComments  []model.Comment
	flat         []flatComment
	collapsedIDs map[string]bool
	bookmarked   bool
	bodyExpanded bool
	cursor       int
	offset       int
	width        int
	height       int
	loading      bool

	// Search
	searchMode   searchMode
	searchQuery  string
	searchInput  string // raw chars typed so far (in searchModeTyping)
	matchIndices []int  // flat indices that contain a match
	matchCursor  int    // which match is current (-1 = none)
}

func NewCommentModel(post model.Post) CommentModel {
	return CommentModel{
		post:         post,
		loading:      true,
		collapsedIDs: make(map[string]bool),
	}
}

func (m *CommentModel) SetBookmarked(bookmarked bool) {
	m.bookmarked = bookmarked
}

func (m *CommentModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *CommentModel) SetComments(comments []model.Comment) {
	m.allComments = comments
	m.loading = false
	m.flatten()
}

func (m *CommentModel) flatten() {
	m.flat = []flatComment{}
	var walk func([]model.Comment)
	walk = func(comments []model.Comment) {
		for _, c := range comments {
			childCount := countDescendants(c)
			collapsed := m.collapsedIDs[c.ID]
			m.flat = append(m.flat, flatComment{
				comment:     c,
				hasChildren: len(c.Children) > 0,
				childCount:  childCount,
				collapsed:   collapsed,
			})
			if !collapsed {
				walk(c.Children)
			}
		}
	}
	walk(m.allComments)
	if m.cursor >= len(m.flat) {
		m.cursor = len(m.flat) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func countDescendants(c model.Comment) int {
	count := len(c.Children)
	for _, child := range c.Children {
		count += countDescendants(child)
	}
	return count
}

func (m CommentModel) Update(msg tea.Msg) (CommentModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.searchMode {
		case searchModeTyping:
			return m.updateSearchTyping(msg)
		case searchModeActive:
			return m.updateSearchActive(msg)
		default:
			return m.updateNormal(msg)
		}
	}
	m.ensureCursorVisible()
	return m, nil
}

func (m CommentModel) updateNormal(msg tea.KeyMsg) (CommentModel, tea.Cmd) {
	switch {
	case msg.String() == "/":
		m.searchMode = searchModeTyping
		m.searchInput = ""
	case key.Matches(msg, keys.ExpandBody):
		m.bodyExpanded = !m.bodyExpanded
		m.flatten()
	case key.Matches(msg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, keys.Down):
		if m.cursor < len(m.flat)-1 {
			m.cursor++
		}
	case key.Matches(msg, keys.Toggle):
		if m.cursor >= 0 && m.cursor < len(m.flat) {
			fc := m.flat[m.cursor]
			if fc.hasChildren {
				m.collapsedIDs[fc.comment.ID] = !m.collapsedIDs[fc.comment.ID]
				m.flatten()
			}
		}
	case key.Matches(msg, keys.HalfUp):
		half := len(m.flat) / 2
		if half < 1 {
			half = 1
		}
		m.cursor -= half
		if m.cursor < 0 {
			m.cursor = 0
		}
	case key.Matches(msg, keys.HalfDown):
		half := len(m.flat) / 2
		if half < 1 {
			half = 1
		}
		m.cursor += half
		if m.cursor >= len(m.flat) {
			m.cursor = len(m.flat) - 1
		}
	}
	m.ensureCursorVisible()
	return m, nil
}

func (m CommentModel) updateSearchTyping(msg tea.KeyMsg) (CommentModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.searchQuery = m.searchInput
		m.computeMatches(m.searchQuery)
		if len(m.matchIndices) > 0 {
			m.searchMode = searchModeActive
			m.matchCursor = 0
			m.cursor = m.matchIndices[0]
			m.ensureCursorVisible()
		} else {
			m.searchMode = searchModeActive // still active, just no matches
		}
	case "esc":
		m.searchMode = searchModeNone
		m.searchInput = ""
		m.searchQuery = ""
		m.matchIndices = nil
	case "backspace", "ctrl+h":
		if len(m.searchInput) > 0 {
			runes := []rune(m.searchInput)
			m.searchInput = string(runes[:len(runes)-1])
		}
	default:
		// Append printable chars
		if len(msg.Runes) == 1 {
			m.searchInput += string(msg.Runes)
		}
	}
	return m, nil
}

func (m CommentModel) updateSearchActive(msg tea.KeyMsg) (CommentModel, tea.Cmd) {
	switch msg.String() {
	case "n":
		if len(m.matchIndices) > 0 {
			m.matchCursor = (m.matchCursor + 1) % len(m.matchIndices)
			m.cursor = m.matchIndices[m.matchCursor]
			m.ensureCursorVisible()
		}
	case "N":
		if len(m.matchIndices) > 0 {
			m.matchCursor = (m.matchCursor - 1 + len(m.matchIndices)) % len(m.matchIndices)
			m.cursor = m.matchIndices[m.matchCursor]
			m.ensureCursorVisible()
		}
	case "/":
		// Re-enter typing mode with current query
		m.searchMode = searchModeTyping
		m.searchInput = m.searchQuery
	case "esc":
		m.searchMode = searchModeNone
		m.searchQuery = ""
		m.matchIndices = nil
	default:
		// Allow normal navigation while search is active
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m *CommentModel) ensureCursorVisible() {
	footerHeight := 1
	contentHeight := m.height - m.headerHeight() - footerHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	linesBefore := 0
	for i := 0; i < m.cursor; i++ {
		if i < len(m.flat) {
			linesBefore += m.commentHeight(m.flat[i])
		}
	}

	var currentCommentHeight int
	if m.cursor >= 0 && m.cursor < len(m.flat) {
		currentCommentHeight = m.commentHeight(m.flat[m.cursor])
	}

	linesAfter := linesBefore + currentCommentHeight

	if linesBefore < m.offset {
		m.offset = linesBefore
	} else if linesAfter > m.offset+contentHeight {
		m.offset = linesAfter - contentHeight
	}
}

func (m CommentModel) commentHeight(fc flatComment) int {
	if fc.collapsed {
		return 3 // header + "[N replies hidden]" + blank
	}
	maxWidth := m.width - (fc.comment.Depth * 4) - 4
	if maxWidth < 40 {
		maxWidth = 40
	}
	textLines := len(wrapText(fc.comment.Text, maxWidth))
	return 1 + textLines + 1 // header + text + blank
}

func (m CommentModel) View() string {
	if m.loading {
		return "\n  Loading comments..."
	}
	header := m.headerView()
	content := m.contentView()
	footer := m.footerView()
	return fmt.Sprintf("%s\n%s\n%s", header, content, footer)
}

func (m CommentModel) headerView() string {
	titleText := truncate(m.post.Title, m.width-4)
	if m.bookmarked {
		titleText = "★ " + titleText
	}
	title := titleStyle.Render(titleText)
	info := statusBarStyle.Render(fmt.Sprintf("▲ %d  %s  %d comments",
		m.post.Points, m.post.Author, m.post.CommentCount))

	parts := []string{title, info}

	if m.post.Text != "" {
		maxWidth := m.width - 4
		if maxWidth < 40 {
			maxWidth = 40
		}
		lines := wrapText(m.post.Text, maxWidth)
		maxLines := 5
		truncated := false
		if !m.bodyExpanded && len(lines) > maxLines {
			lines = lines[:maxLines]
			truncated = true
		}
		body := commentTextStyle.Render(strings.Join(lines, "\n"))
		parts = append(parts, "", body)
		if truncated {
			parts = append(parts, statusBarStyle.Render("  [e to expand post body...]"))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// headerHeight returns the number of terminal lines the header occupies.
func (m CommentModel) headerHeight() int {
	h := 2 // title + info
	if m.post.Text != "" {
		maxWidth := m.width - 4
		if maxWidth < 40 {
			maxWidth = 40
		}
		lines := wrapText(m.post.Text, maxWidth)
		maxLines := 5
		if !m.bodyExpanded && len(lines) > maxLines {
			h += 1 + maxLines + 1 // blank line + capped lines + hint line
		} else {
			h += 1 + len(lines) // blank line + text lines
		}
	}
	return h
}

func (m CommentModel) footerView() string {
	// Search typing mode: show the input bar
	if m.searchMode == searchModeTyping {
		return statusBarStyle.Render(fmt.Sprintf("/%s_  enter confirm • esc cancel", m.searchInput))
	}

	total := len(m.flat)
	cur := m.cursor + 1
	if total == 0 {
		cur = 0
	}

	bookmarkStatus := ""
	if m.bookmarked {
		bookmarkStatus = " • bookmarked"
	}

	// Search active mode: show search status
	if m.searchMode == searchModeActive {
		matchStatus := "no matches"
		if len(m.matchIndices) > 0 {
			matchStatus = fmt.Sprintf("match %d/%d", m.matchCursor+1, len(m.matchIndices))
		}
		return statusBarStyle.Render(fmt.Sprintf(
			"/%s  %s  n/N next/prev • esc clear%s • %d/%d",
			m.searchQuery, matchStatus, bookmarkStatus, cur, total))
	}

	return statusBarStyle.Render(fmt.Sprintf(
		"j/k navigate • c-u/c-d half page • enter/space fold • esc back • / search%s • %d/%d",
		bookmarkStatus, cur, total))
}

func (m CommentModel) contentView() string {
	if len(m.flat) == 0 {
		return "\n  No comments."
	}

	footerHeight := 1
	contentHeight := m.height - m.headerHeight() - footerHeight
	if contentHeight < 1 {
		return ""
	}

	// Render all visible comments into a single slice of lines
	var allLines []string
	for i, fc := range m.flat {
		rendered := m.renderSingleComment(fc, i == m.cursor, m.width)
		lines := strings.Split(rendered, "\n")
		// Remove trailing empty string from Split if rendered ends with \n
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		allLines = append(allLines, lines...)
	}

	// Apply scroll offset and cap at contentHeight
	start := m.offset
	if start > len(allLines) {
		start = len(allLines)
	}
	end := start + contentHeight
	if end > len(allLines) {
		end = len(allLines)
	}

	visible := allLines[start:end]

	// Pad with empty lines so footer stays pinned at the bottom
	for len(visible) < contentHeight {
		visible = append(visible, "")
	}

	return strings.Join(visible, "\n")
}

func (m CommentModel) renderSingleComment(fc flatComment, selected bool, width int) string {
	var b strings.Builder
	pipe := commentDepthStyle.Render(strings.Repeat("│ ", fc.comment.Depth))
	indent := strings.Repeat("  ", fc.comment.Depth)

	var selectIndicator string
	if selected {
		selectIndicator = selectedItemStyle.Render("▌ ")
	} else {
		selectIndicator = "  "
	}

	// Header: fold indicator + author + timestamp
	var foldIndicator string
	if fc.hasChildren {
		if fc.collapsed {
			foldIndicator = "▶ "
		} else {
			foldIndicator = "▼ "
		}
	} else {
		foldIndicator = "  "
	}

	author := commentAuthorStyle.Render(fc.comment.Author)
	age := statusBarStyle.Render(timeAgo(fc.comment.CreatedAt))
	b.WriteString(fmt.Sprintf("%s%s%s%s  %s\n", selectIndicator, pipe, foldIndicator, author, age))

	// Comment body
	if fc.collapsed && fc.hasChildren {
		b.WriteString(fmt.Sprintf("%s%s  %s\n", selectIndicator, pipe, statusBarStyle.Render(fmt.Sprintf("[%d replies hidden]", fc.childCount))))
	} else {
		maxWidth := width - (fc.comment.Depth * 4) - 4
		if maxWidth < 40 {
			maxWidth = 40
		}
		lines := wrapText(fc.comment.Text, maxWidth)
		for _, line := range lines {
			rendered := renderLineWithHighlight(line, m.searchQuery)
			b.WriteString(selectIndicator + pipe + indent + rendered + "\n")
		}
	}
	b.WriteString(selectIndicator + "\n")

	return b.String()
}

// renderLineWithHighlight renders a text line with search matches highlighted.
// Non-matching segments use commentTextStyle; matches use searchHighlightStyle.
func renderLineWithHighlight(line, query string) string {
	if query == "" {
		return commentTextStyle.Render(line)
	}
	q := strings.ToLower(query)
	lower := strings.ToLower(line)
	var result strings.Builder
	i := 0
	for {
		idx := strings.Index(lower[i:], q)
		if idx == -1 {
			result.WriteString(commentTextStyle.Render(line[i:]))
			break
		}
		abs := i + idx
		if abs > i {
			result.WriteString(commentTextStyle.Render(line[i:abs]))
		}
		result.WriteString(searchHighlightStyle.Render(line[abs : abs+len(q)]))
		i = abs + len(q)
	}
	return result.String()
}

// computeMatches populates matchIndices for all flat comments matching query.
func (m *CommentModel) computeMatches(query string) {
	m.matchIndices = nil
	m.matchCursor = 0
	if query == "" {
		return
	}
	q := strings.ToLower(query)
	for i, fc := range m.flat {
		if strings.Contains(strings.ToLower(fc.comment.Text), q) ||
			strings.Contains(strings.ToLower(fc.comment.Author), q) {
			m.matchIndices = append(m.matchIndices, i)
		}
	}
}

// highlightMatches wraps all case-insensitive occurrences of query in text with
// open/close delimiters. The original case of matched text is preserved.
// In production the open/close are ANSI escape sequences from lipgloss; in tests
// they are simple bracket strings for readability.
func highlightMatches(text, query, open, close string) string {
	if query == "" {
		return text
	}
	q := strings.ToLower(query)
	var result strings.Builder
	lower := strings.ToLower(text)
	i := 0
	for {
		idx := strings.Index(lower[i:], q)
		if idx == -1 {
			result.WriteString(text[i:])
			break
		}
		abs := i + idx
		result.WriteString(text[i:abs])
		result.WriteString(open)
		result.WriteString(text[abs : abs+len(query)])
		result.WriteString(close)
		i = abs + len(query)
	}
	return result.String()
}

// wrapText wraps s to width characters, splitting on word boundaries.
func wrapText(s string, width int) []string {
	if width <= 0 || s == "" {
		return []string{s}
	}
	var lines []string
	for _, paragraph := range strings.Split(s, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		current := words[0]
		for _, w := range words[1:] {
			if len(current)+1+len(w) > width {
				lines = append(lines, current)
				current = w
			} else {
				current += " " + w
			}
		}
		lines = append(lines, current)
	}
	return lines
}

// truncate shortens s to max runes, appending "…" if truncated.
func truncate(s string, max int) string {
	if max <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
