package app

import (
	_ "embed"
	"strings"
)

//go:embed filters_exclude.txt
var defaultExcludeFilters string

//go:embed filters_include.txt
var defaultIncludeFilters string

// Filter represents a single filter rule.
type Filter struct {
	Pattern string `json:"pattern"`
	Type    string `json:"type"`    // "exclude" or "include"
	Enabled bool   `json:"enabled"`
	Builtin bool   `json:"builtin"`
}

// LoadDefaultFilters returns the built-in filter set.
func LoadDefaultFilters() []Filter {
	var filters []Filter

	for _, line := range strings.Split(defaultExcludeFilters, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		filters = append(filters, Filter{
			Pattern: line,
			Type:    "exclude",
			Enabled: true,
			Builtin: true,
		})
	}

	for _, line := range strings.Split(defaultIncludeFilters, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		filters = append(filters, Filter{
			Pattern: line,
			Type:    "include",
			Enabled: true,
			Builtin: true,
		})
	}

	return filters
}

// MatchesFilter checks if an event's title, location, or notes contain the pattern.
func MatchesFilter(pattern, title, location, notes string) bool {
	lower := strings.ToLower(pattern)
	return strings.Contains(strings.ToLower(title), lower) ||
		strings.Contains(strings.ToLower(location), lower) ||
		strings.Contains(strings.ToLower(notes), lower)
}
