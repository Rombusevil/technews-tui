package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"technews-tui/internal/config"
	"technews-tui/internal/ui"
)

func main() {
	subreddits := flag.String("subreddits", "", "comma-separated subreddits (overrides config)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		cfg = &config.Config{Subreddits: config.DefaultSubreddits}
	}

	if *subreddits != "" {
		cfg.Subreddits = strings.Split(*subreddits, ",")
	}

	p := tea.NewProgram(
		ui.NewRootModel(cfg),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
