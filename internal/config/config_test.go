package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `defaultProfile: dev
profiles:
  dev:
    name: "Dev Org"
    region: "USW3"
    username: "dev@company.com"
    password: "secret"
  prod:
    name: "Prod Org"
    region: "EMEA"
    username: "prod@company.com"
    password: "prodsecret"
    loginUrl: "https://custom.example.com/login"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("writing test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.DefaultProfile != "dev" {
		t.Errorf("expected default profile 'dev', got %s", cfg.DefaultProfile)
	}
	if len(cfg.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(cfg.Profiles))
	}

	dev := cfg.Profiles["dev"]
	if dev.Username != "dev@company.com" {
		t.Errorf("expected dev username 'dev@company.com', got %s", dev.Username)
	}
	if dev.Region != "USW3" {
		t.Errorf("expected dev region 'USW3', got %s", dev.Region)
	}

	prod := cfg.Profiles["prod"]
	if prod.LoginURL != "https://custom.example.com/login" {
		t.Errorf("expected prod loginUrl, got %s", prod.LoginURL)
	}
}

func TestLoadConfigNonExistent(t *testing.T) {
	cfg, err := Load("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("expected no error for missing config, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if len(cfg.Profiles) != 0 {
		t.Errorf("expected empty profiles, got %d", len(cfg.Profiles))
	}
}

func TestResolveProfile(t *testing.T) {
	cfg := &Config{
		DefaultProfile: "dev",
		Profiles: map[string]*Profile{
			"dev": {
				Name:     "Dev",
				Region:   "USW3",
				Username: "dev@co.com",
				Password: "pass",
			},
		},
	}

	p, err := cfg.ResolveProfile("dev")
	if err != nil {
		t.Fatalf("ResolveProfile() error: %v", err)
	}
	if p.Username != "dev@co.com" {
		t.Errorf("expected username 'dev@co.com', got %s", p.Username)
	}
}

func TestResolveProfileEnvOverride(t *testing.T) {
	cfg := &Config{
		DefaultProfile: "dev",
		Profiles: map[string]*Profile{
			"dev": {
				Name:     "Dev",
				Region:   "USW3",
				Username: "dev@co.com",
				Password: "pass",
			},
		},
	}

	t.Setenv("IICS_USERNAME", "override@co.com")
	t.Setenv("IICS_REGION", "EMEA")

	p, err := cfg.ResolveProfile("dev")
	if err != nil {
		t.Fatalf("ResolveProfile() error: %v", err)
	}
	if p.Username != "override@co.com" {
		t.Errorf("expected overridden username, got %s", p.Username)
	}
	if p.Region != "EMEA" {
		t.Errorf("expected overridden region EMEA, got %s", p.Region)
	}
}

func TestResolveProfileMissingCredentials(t *testing.T) {
	cfg := &Config{
		Profiles: map[string]*Profile{},
	}

	// Clear any env vars that might interfere
	t.Setenv("IICS_USERNAME", "")
	t.Setenv("IICS_PASSWORD", "")

	_, err := cfg.ResolveProfile("nonexistent")
	if err == nil {
		t.Error("expected error for missing credentials")
	}
}

func TestDeriveCaiURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "https://use4.dm-us.informaticacloud.com/saas",
			want:  "https://use4-cai.dm-us.informaticacloud.com",
		},
		{
			input: "https://dm-ap.informaticacloud.com/saas",
			want:  "https://dm-ap-cai.informaticacloud.com",
		},
		{
			input: "https://dm1-em.informaticacloud.com/saas",
			want:  "https://dm1-em-cai.informaticacloud.com",
		},
		{
			input: "",
			want:  "",
		},
		{
			input: "not-a-url",
			want:  "",
		},
		{
			input: "https://nodot",
			want:  "",
		},
	}

	for _, tc := range tests {
		got := DeriveCaiURL(tc.input)
		if got != tc.want {
			t.Errorf("DeriveCaiURL(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestProfileGetLoginURL(t *testing.T) {
	// With explicit loginUrl
	p := &Profile{LoginURL: "https://custom.example.com/login"}
	url, err := p.GetLoginURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://custom.example.com/login" {
		t.Errorf("expected custom URL, got %s", url)
	}

	// With region
	p2 := &Profile{Region: "USW3"}
	url2, err := p2.GetLoginURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url2 != "https://dm-us.informaticacloud.com/saas/public/core/v3/login" {
		t.Errorf("expected resolved URL, got %s", url2)
	}

	// With neither
	p3 := &Profile{}
	_, err = p3.GetLoginURL()
	if err == nil {
		t.Error("expected error with no region or loginUrl")
	}
}
