package browser

import (
	"os/exec"
	"runtime"
	"strings"

	"technews-tui/internal/config"
)

// Open opens the given URL in the system default browser or the configured one.
func Open(browserCfg *config.BrowserConfig, url string) error {
	var cmd string
	args := make([]string, 0)

	useDefaultBrowser := browserCfg == nil
	if useDefaultBrowser {
		switch runtime.GOOS {
		case "darwin":
			cmd = "open"
		default:
			cmd = "xdg-open"
		}

	} else {
		cmd = browserCfg.Command
		if browserCfg.Arguments != "" {
			args = append(args, strings.Split(browserCfg.Arguments, " ")...)
		}
	}

	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
