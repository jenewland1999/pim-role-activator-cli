package model

import "regexp"

// DecodeScopeFields extracts env and app labels from a scope name using the
// caller-supplied regexp. The regexp should define named capture groups:
//
//	(?P<env>…)  → displayed in the Env column
//	(?P<app>…)  → displayed in the App column
//
// Either group may be omitted from the pattern; missing groups return "—".
// Returns ("—", "—") when re is non-nil but the pattern does not match.
// Returns ("", "") when re is nil, signalling that the columns should be hidden.
//
// envLabels is an optional map that translates raw decoded env values to
// friendly display labels (e.g. {"P": "Prod", "D": "Dev", "Q": "QA"}).
// Pass nil to use the raw extracted value unchanged.
func DecodeScopeFields(scopeName string, re *regexp.Regexp, envLabels map[string]string) (env, app string) {
	if re == nil {
		return "", ""
	}
	match := re.FindStringSubmatch(scopeName)
	if match == nil {
		return "—", "—"
	}
	env = namedGroup(re, match, "env")
	app = namedGroup(re, match, "app")
	if env == "" {
		env = "—"
	}
	if app == "" {
		app = "—"
	}
	// Apply optional label mapping to the env value.
	if env != "—" && len(envLabels) > 0 {
		if label, ok := envLabels[env]; ok {
			env = label
		}
	}
	return env, app
}

// namedGroup returns the value of a named capture group from a regexp match,
// or "" if the group does not exist in the pattern.
func namedGroup(re *regexp.Regexp, match []string, name string) string {
	idx := re.SubexpIndex(name)
	if idx < 0 || idx >= len(match) {
		return ""
	}
	return match[idx]
}
