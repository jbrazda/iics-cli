package config

import (
	"os"

	"golang.org/x/term"
)

// IsTerminal reports whether stdin is an interactive terminal.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
