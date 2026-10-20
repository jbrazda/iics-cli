package config

import (
	"testing"
	"time"
)

func TestIsProductionProfile(t *testing.T) {
	yes, no := true, false
	tests := []struct {
		name string
		p    *Profile
		want bool
	}{
		{"prd", nil, true},
		{"PROD", &Profile{}, true},
		{"production", nil, true},
		{"prod-eu", nil, true},
		{"prd-us", nil, true},
		{"dev", nil, false},
		{"preprod", nil, false},
		{"dev", &Profile{Production: &yes}, true},
		{"prd", &Profile{Production: &no}, false},
	}
	for _, tt := range tests {
		if got := IsProductionProfile(tt.name, tt.p); got != tt.want {
			t.Errorf("IsProductionProfile(%q, %+v) = %v, want %v", tt.name, tt.p, got, tt.want)
		}
	}
}

func TestSessionRemaining(t *testing.T) {
	e := &SessionEntry{CreatedAt: time.Now().Add(-10 * time.Minute)}
	if r := e.Remaining(); r < 19*time.Minute || r > 20*time.Minute {
		t.Errorf("Remaining = %v, want about 20m", r)
	}
	old := &SessionEntry{CreatedAt: time.Now().Add(-time.Hour)}
	if old.Remaining() > 0 || !old.IsExpired() {
		t.Error("an hour-old session should be expired")
	}
}
