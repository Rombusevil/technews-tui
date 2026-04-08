package ui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6600")).
			Bold(true).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)

	commentAuthorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF6600")).
				Bold(true)

	commentTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	commentDepthStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))

	sourceTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6600")).
			Bold(true)

	searchHighlightStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#FFFF00")).
				Foreground(lipgloss.Color("#000000"))
)
