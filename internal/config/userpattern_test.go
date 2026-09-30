package config

import (
	"strings"
	"testing"
)

func TestNewUserPatterns(t *testing.T) {
	d := NewUserPatterns(&Profile{Username: "admin@Acme.com"})
	if d.Domain != "Acme.com" || d.UserNamePattern != DefaultUserNamePattern || d.EmailPattern != DefaultEmailPattern {
		t.Errorf("defaults = %+v", d)
	}
	c := NewUserPatterns(&Profile{Username: "admin@acme.com", NewUser: &NewUserConfig{Domain: "corp.example", EmailPattern: "{firstInitial}{lastName}@{domain}"}})
	if c.Domain != "corp.example" || c.UserNamePattern != DefaultUserNamePattern || c.EmailPattern != "{firstInitial}{lastName}@{domain}" {
		t.Errorf("custom = %+v", c)
	}
	if n := NewUserPatterns(&Profile{Username: "admin"}); n.Domain != "" {
		t.Errorf("no @ in username should give no domain, got %q", n.Domain)
	}
}

func TestExpandUserPattern(t *testing.T) {
	v := UserPatternValues{FirstName: "Mary Ann", LastName: "O'Neil", ProfileName: "Dev", Domain: "Acme.com"}
	tests := []struct {
		pattern, want, wantErr string
	}{
		{DefaultUserNamePattern, "maryann.o'neil.dev@acme.com", ""},
		{DefaultEmailPattern, "maryann.o'neil@acme.com", ""},
		{"{firstInitial}{lastInitial}-{profileName}", "mo-dev", ""},
		{"{first}.{lastName}", "", "unknown placeholder {first}"},
	}
	for _, tt := range tests {
		got, err := ExpandUserPattern(tt.pattern, v)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("%s: error = %v, want %q", tt.pattern, err, tt.wantErr)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("%s: got %q, %v; want %q", tt.pattern, got, err, tt.want)
		}
	}
	if _, err := ExpandUserPattern(DefaultEmailPattern, UserPatternValues{FirstName: "a", LastName: "b"}); err == nil || !strings.Contains(err.Error(), "no value for {domain}") {
		t.Errorf("missing domain should fail, got %v", err)
	}
}
