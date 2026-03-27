package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"technews-tui/internal/config"
	"technews-tui/internal/ui"
)

func printConfig() {
	path := config.Path()
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Config file: %s\n\n", path)

	// Sort sources for stable output
	var keys []string
	for k := range cfg.Sources {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, id := range keys {
		sc := cfg.Sources[id]
		enabledStr := "enabled"
		if !sc.Enabled {
			enabledStr = "disabled"
		}
		fmt.Printf("[%s] (%s)\n", strings.ToUpper(id), enabledStr)
		fmt.Printf("  Sort:    %s\n", sc.Sort)
		if len(sc.Targets) > 0 {
			fmt.Printf("  Targets: %s\n", strings.Join(sc.Targets, ", "))
		}
		fmt.Println()
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "config" {
		printConfig()
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		// Fallback defaults already handled by config.Load()
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
