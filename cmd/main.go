package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"technews-tui/internal/config"
	"technews-tui/internal/ui"
)

func helpText() string {
	return strings.TrimSpace(`Usage:
  technews-tui [command] [flags]

Commands:
  config    Show config file path and current settings
  help      Show this help message

Flags:
  --reddit-subreddits  Override Reddit subreddits for this run (comma-separated)
  -h, --help  Show this help message`)
}

func printHelp() {
	fmt.Println(helpText())
}

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

func applyCLIOverrides(cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("technews-tui", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	redditSubreddits := fs.String("reddit-subreddits", "", "override Reddit subreddits for this run (comma-separated)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *redditSubreddits != "" {
		parts := strings.Split(*redditSubreddits, ",")
		var targets []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			targets = append(targets, p)
		}
		reddit := cfg.Sources["reddit"]
		reddit.Targets = targets
		cfg.Sources["reddit"] = reddit
	}

	return nil
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "config":
			printConfig()
			return
		case "help", "-h", "--help":
			printHelp()
			return
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		// Fallback defaults already handled by config.Load()
	}

	if err := applyCLIOverrides(cfg, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
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
