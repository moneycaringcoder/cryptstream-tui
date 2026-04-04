package ui

import (
	"os"
)

func init() {
	// Force color output for tests so lipgloss renders ANSI codes
	os.Setenv("TERM", "xterm-256color")
	os.Setenv("COLORTERM", "truecolor")
	os.Setenv("CLICOLOR_FORCE", "1")
}
