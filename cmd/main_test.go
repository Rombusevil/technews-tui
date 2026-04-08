package main

import (
	"strings"
	"testing"

	"technews-tui/internal/config"
)

func TestHelpTextListsCommands(t *testing.T) {
	help := helpText()

	for _, want := range []string{
		"Usage:",
		"technews-tui [command]",
		"config",
		"help",
		"--help",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("help text missing %q:\n%s", want, help)
		}
	}
}

func TestApplyCLIOverridesUpdatesRedditSubreddits(t *testing.T) {
	cfg := &config.Config{
		Sources: map[string]config.SourceConfig{
			"reddit": {
				Enabled: true,
				Sort:    "hot",
				Targets: []string{"oldsub"},
			},
		},
	}

	if err := applyCLIOverrides(cfg, []string{"--reddit-subreddits", "golang,programming,linux"}); err != nil {
		t.Fatalf("applyCLIOverrides returned error: %v", err)
	}

	got := cfg.Sources["reddit"].Targets
	want := []string{"golang", "programming", "linux"}
	if len(got) != len(want) {
		t.Fatalf("targets len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("targets[%d] = %q, want %q (all=%v)", i, got[i], want[i], got)
		}
	}
}
