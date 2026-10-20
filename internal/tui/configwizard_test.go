package tui

import (
	"reflect"
	"testing"

	"github.com/jbrazda/iics-cli/internal/config"
)

func TestConfigFormApply(t *testing.T) {
	before := &config.Config{HTTPTimeout: 60}
	f := newConfigForm(before)
	if f.timeout != "60" || !f.responsive || !f.menu {
		t.Errorf("form = %+v", f)
	}

	// Unchanged form: no changes, defaults stay unset.
	after := f.apply(before)
	if lines := ConfigChanges(before, after); len(lines) != 0 {
		t.Errorf("unexpected changes %v", lines)
	}
	if after.Style.ResponsiveTables != nil || after.UI.Menu != nil || after.NewUser != nil {
		t.Errorf("defaults were written: %+v", after)
	}

	f.domain, f.emPat = " acme.com ", "{firstInitial}{lastName}@{domain}"
	f.theme, f.timeout, f.menu = "minimal", "", false
	after = f.apply(before)
	want := []string{
		"New user domain: - -> acme.com",
		"Email pattern: - -> {firstInitial}{lastName}@{domain}",
		"Table theme: - -> minimal",
		"HTTP timeout: 60 -> -",
		"Main menu: true -> false",
	}
	if got := ConfigChanges(before, after); !reflect.DeepEqual(got, want) {
		t.Errorf("ConfigChanges = %q\nwant %q", got, want)
	}
	if before.NewUser != nil || before.Style.Theme != "" {
		t.Error("apply modified the input config")
	}
}
