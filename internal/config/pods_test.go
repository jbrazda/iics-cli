package config

import (
	"strings"
	"testing"
)

func TestLoginURL(t *testing.T) {
	tests := []struct {
		region  string
		wantURL string
		wantErr bool
	}{
		{"US", "https://dm-us.informaticacloud.com/saas/public/core/v3/login", false},
		{"USW3", "https://dm-us.informaticacloud.com/saas/public/core/v3/login", false},
		{"usw3", "https://dm-us.informaticacloud.com/saas/public/core/v3/login", false},
		{"EMEA", "https://dm-em.informaticacloud.com/saas/public/core/v3/login", false},
		{"CAC1", "https://dm-na.informaticacloud.com/saas/public/core/v3/login", false},
		{"APSE1", "https://dm-ap.informaticacloud.com/saas/public/core/v3/login", false},
		{"APNE1", "https://dm1-ap.informaticacloud.com/saas/public/core/v3/login", false},
		{"USW1-1", "https://dm1-us.informaticacloud.com/saas/public/core/v3/login", false},
		{"INVALID", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			url, err := LoginURL(tt.region)
			if tt.wantErr {
				if err == nil {
					t.Errorf("LoginURL(%q) expected error, got nil", tt.region)
				}
				return
			}
			if err != nil {
				t.Errorf("LoginURL(%q) unexpected error: %v", tt.region, err)
				return
			}
			if url != tt.wantURL {
				t.Errorf("LoginURL(%q) = %q, want %q", tt.region, url, tt.wantURL)
			}
		})
	}
}

func TestValidRegions(t *testing.T) {
	regions := ValidRegions()
	if !strings.Contains(regions, "USW3") {
		t.Errorf("ValidRegions() should contain USW3, got: %s", regions)
	}
	if !strings.Contains(regions, "EMEA") {
		t.Errorf("ValidRegions() should contain EMEA, got: %s", regions)
	}
}
