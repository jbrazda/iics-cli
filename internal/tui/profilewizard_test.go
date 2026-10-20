package tui

import (
	"reflect"
	"testing"

	"github.com/jbrazda/iics-cli/internal/config"
)

func TestProfileFormResult(t *testing.T) {
	existing := &config.Profile{
		Username: "admin@acme.com", Password: config.KeyringSentinel, Region: "usw3",
		BaseAPIURL: "https://usw3.dm-us.informaticacloud.com/saas",
		NewUser:    &config.NewUserConfig{EmailPattern: "{lastName}@{domain}"},
	}
	in := ProfileWizardInput{Name: "dev", Existing: existing}
	f := newProfileForm(in)
	if f.region != "USW3" || f.caiURL != "https://usw3-cai.dm-us.informaticacloud.com" || f.makeDflt {
		t.Errorf("form = %+v", f)
	}

	// Keeping the keychain password skips the keychain question.
	res := f.result(in)
	if res.Profile.Password != config.KeyringSentinel || res.StoreInKeychain {
		t.Errorf("kept keychain password: password=%q store=%v", res.Profile.Password, res.StoreInKeychain)
	}
	if res.Profile.LoginURL != "https://dm-us.informaticacloud.com/saas/public/core/v3/login" {
		t.Errorf("login URL = %q", res.Profile.LoginURL)
	}

	// New password, custom login URL, cleared new user overrides.
	f.password = "secret"
	f.region, f.loginURL = customLoginURL, " https://example.com/login "
	f.emPat = ""
	res = f.result(in)
	p := res.Profile
	if p.Password != "secret" || !res.StoreInKeychain || p.Region != "" || p.LoginURL != "https://example.com/login" || p.NewUser != nil {
		t.Errorf("result = %+v store=%v", p, res.StoreInKeychain)
	}
	if existing.Password != config.KeyringSentinel || existing.NewUser == nil {
		t.Error("existing profile was modified")
	}

	got := ProfileChanges(existing, p)
	want := []string{
		"Region: usw3 -> -",
		"Login URL: - -> https://example.com/login",
		"CAI URL: - -> https://usw3-cai.dm-us.informaticacloud.com",
		"Email pattern: {lastName}@{domain} -> -",
		"Password: changed",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ProfileChanges = %q\nwant %q", got, want)
	}
}

func TestProfileFormNewProfile(t *testing.T) {
	f := newProfileForm(ProfileWizardInput{Name: "prod"})
	if !f.production || !f.makeDflt || f.region != "" || f.hasPassword {
		t.Errorf("new form = %+v", f)
	}
	if err := optionalPattern("{nope}"); err == nil {
		t.Error("optionalPattern accepted an unknown placeholder")
	}
	if err := validLoginURL("example.com"); err == nil {
		t.Error("validLoginURL accepted a URL without scheme")
	}
}
