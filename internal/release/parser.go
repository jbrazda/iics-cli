package release

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reTagLine = regexp.MustCompile(`(?i)^\s*tag:\s*` + "`?" + `([^` + "`" + `\s]+)` + "`?" + `(?:\s*<!--.*-->)?\s*$`)
)

func ParseDeploymentOptionsMarkdown(markdown string) (Options, error) {
	opts := DefaultOptions()
	if strings.TrimSpace(markdown) == "" {
		return opts, nil
	}

	lines := strings.Split(markdown, "\n")
	targets := make([]string, 0, 4)
	targetSeen := map[string]bool{}
	modeSeen := false

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		lower := strings.ToLower(line)

		if strings.Contains(lower, "- [x] full deployment") {
			opts.Mode = ModeFullDeployment
			modeSeen = true
			continue
		}
		if strings.Contains(lower, "- [x] selective - tag-based") {
			opts.Mode = ModeTagBased
			modeSeen = true
			continue
		}
		if m := reTagLine.FindStringSubmatch(line); len(m) == 2 {
			opts.Tag = strings.TrimSpace(m[1])
			continue
		}

		for _, env := range []string{"tst", "qa", "stg", "prod"} {
			if strings.Contains(lower, "- [x] "+strings.ToLower(env)) {
				if !targetSeen[env] {
					targetSeen[env] = true
					targets = append(targets, env)
				}
			}
		}

		if strings.Contains(lower, "- [x] connectors") {
			opts.IncludeConnectors = true
		}
		if strings.Contains(lower, "- [x] connections") {
			opts.IncludeConnections = true
		}
	}

	if len(targets) > 0 {
		opts.Targets = targets
	}
	if !modeSeen {
		// Defaults remain full deployment.
		opts.Mode = ModeFullDeployment
	}
	if err := ValidateOptions(&opts); err != nil {
		return Options{}, fmt.Errorf("invalid parsed deployment options: %w", err)
	}
	return opts, nil
}
