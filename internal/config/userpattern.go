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

// NewUserConfig holds the settings for new users. The top-level config
// block holds global defaults; a profile block overrides them per field.
type NewUserConfig struct {
	// Domain replaces {domain}; defaults to the domain of the profile username.
	Domain          string `yaml:"domain,omitempty" mapstructure:"domain"`
	UserNamePattern string `yaml:"userNamePattern,omitempty" mapstructure:"userNamePattern"`
	EmailPattern    string `yaml:"emailPattern,omitempty" mapstructure:"emailPattern"`
}

// IsZero reports whether no field is set.
func (n NewUserConfig) IsZero() bool {
	return n == NewUserConfig{}
}

// Sources of an effective new-user value (see ResolveNewUser).
const (
	SourceProfile  = "profile"
	SourceGlobal   = "global"
	SourceUsername = "profile username"
	SourceDefault  = "built-in default"
)

// NewUserSources names where each effective new-user value comes from.
type NewUserSources struct {
	Domain, UserNamePattern, EmailPattern string
}

// ResolveNewUser returns the effective new-user settings of a profile and
// their sources. Each field resolves as: profile newUser, global newUser,
// built-in default. The domain falls back to the domain of the profile
// username. global and p may be nil.
func ResolveNewUser(global *NewUserConfig, p *Profile) (NewUserConfig, NewUserSources) {
	out := NewUserConfig{UserNamePattern: DefaultUserNamePattern, EmailPattern: DefaultEmailPattern}
	src := NewUserSources{UserNamePattern: SourceDefault, EmailPattern: SourceDefault}
	if p != nil {
		if i := strings.LastIndex(p.Username, "@"); i >= 0 {
			out.Domain, src.Domain = p.Username[i+1:], SourceUsername
		}
	}
	apply := func(n *NewUserConfig, source string) {
		if n == nil {
			return
		}
		if n.Domain != "" {
			out.Domain, src.Domain = n.Domain, source
		}
		if n.UserNamePattern != "" {
			out.UserNamePattern, src.UserNamePattern = n.UserNamePattern, source
		}
		if n.EmailPattern != "" {
			out.EmailPattern, src.EmailPattern = n.EmailPattern, source
		}
	}
	apply(global, SourceGlobal)
	if p != nil {
		apply(p.NewUser, SourceProfile)
	}
	return out, src
}

// NewUserPatterns returns the effective new-user settings of a profile
// (see ResolveNewUser).
func NewUserPatterns(global *NewUserConfig, p *Profile) NewUserConfig {
	out, _ := ResolveNewUser(global, p)
	return out
}

// ValidateUserPattern reports unknown placeholders in pattern.
func ValidateUserPattern(pattern string) error {
	_, err := ExpandUserPattern(pattern, UserPatternValues{FirstName: "a", LastName: "b", ProfileName: "c", Domain: "d"})
	return err
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
