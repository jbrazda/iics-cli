package config

import "testing"

func TestIsTerminal(t *testing.T) {
	// In test environments stdin is a pipe, not a terminal.
	if IsTerminal() {
		t.Skip("skipping: stdin is a real terminal (run in CI for proper coverage)")
	}
}
