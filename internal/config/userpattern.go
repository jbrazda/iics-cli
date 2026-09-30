package config

import (
	"fmt"
	"regexp"
	"strings"
)

// Default patterns for new user names and emails.
const (
	DefaultUserNamePattern = "{firstName}.{lastName}.{profileName}@{domain}"
	DefaultEmailPattern    = "{firstName}.{lastName}@{domain}"
)

// UserPatternPlaceholders lists the supported placeholders.
var UserPatternPlaceholders = []string{"firstName", "lastName", "firstInitial", "lastInitial", "profileName", "domain"}

// NewUserConfig holds the per-profile settings for new users.
type NewUserConfig struct {
	// Domain replaces {domain}; defaults to the domain of the profile username.
	Domain          string `yaml:"domain,omitempty" mapstructure:"domain"`
	UserNamePattern string `yaml:"userNamePattern,omitempty" mapstructure:"userNamePattern"`
	EmailPattern    string `yaml:"emailPattern,omitempty" mapstructure:"emailPattern"`
}

// NewUserPatterns returns the effective new-user settings of a profile,
// applying defaults for unset values.
func NewUserPatterns(p *Profile) NewUserConfig {
	out := NewUserConfig{UserNamePattern: DefaultUserNamePattern, EmailPattern: DefaultEmailPattern}
	if p == nil {
		return out
	}
	if i := strings.LastIndex(p.Username, "@"); i >= 0 {
		out.Domain = p.Username[i+1:]
	}
	if p.NewUser != nil {
		if p.NewUser.Domain != "" {
			out.Domain = p.NewUser.Domain
		}
		if p.NewUser.UserNamePattern != "" {
			out.UserNamePattern = p.NewUser.UserNamePattern
		}
		if p.NewUser.EmailPattern != "" {
			out.EmailPattern = p.NewUser.EmailPattern
		}
	}
	return out
}

// UserPatternValues are the inputs for ExpandUserPattern.
type UserPatternValues struct {
	FirstName, LastName, ProfileName, Domain string
}

var placeholderRe = regexp.MustCompile(`\{([^{}]*)\}`)

// ExpandUserPattern replaces the placeholders of pattern. Name values are
// lowercased with whitespace removed. It returns an error for unknown
// placeholders or when a used placeholder has no value.
func ExpandUserPattern(pattern string, v UserPatternValues) (string, error) {
	clean := func(s string) string {
		return strings.ToLower(strings.Join(strings.Fields(s), ""))
	}
	initial := func(s string) string {
		if r := []rune(clean(s)); len(r) > 0 {
			return string(r[0])
		}
		return ""
	}
	values := map[string]string{
		"firstName":    clean(v.FirstName),
		"lastName":     clean(v.LastName),
		"firstInitial": initial(v.FirstName),
		"lastInitial":  initial(v.LastName),
		"profileName":  clean(v.ProfileName),
		"domain":       strings.ToLower(strings.TrimSpace(v.Domain)),
	}
	var errs []string
	out := placeholderRe.ReplaceAllStringFunc(pattern, func(m string) string {
		name := m[1 : len(m)-1]
		val, known := values[name]
		switch {
		case !known:
			errs = append(errs, fmt.Sprintf("unknown placeholder %s", m))
		case val == "":
			errs = append(errs, fmt.Sprintf("no value for %s", m))
		}
		return val
	})
	if len(errs) > 0 {
		return "", fmt.Errorf("pattern %q: %s", pattern, strings.Join(errs, "; "))
	}
	return out, nil
}
