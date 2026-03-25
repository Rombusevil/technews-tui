package browser

import (
	"os/exec"
	"runtime"
)

// Open opens the given URL in the system default browser.
func Open(url string) error {
	var cmd string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}

	return exec.Command(cmd, url).Start()
}
