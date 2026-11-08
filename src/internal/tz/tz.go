// Package tz provides the prepopulated list of selectable timezones and a small
// search helper for picking one interactively.
package tz

import "strings"

var spacer = strings.NewReplacer("_", " ", "/", " ")

// Filter returns the zones matching query. Matching is case-insensitive, treats
// "/" and "_" as spaces, and requires every whitespace-separated term in the
// query to appear (so "new york" matches "America/New_York"). An empty query
// returns the full list.
func Filter(query string) []string {
	terms := strings.Fields(strings.ToLower(spacer.Replace(query)))
	if len(terms) == 0 {
		return Zones
	}
	out := make([]string, 0, 16)
	for _, z := range Zones {
		hay := strings.ToLower(spacer.Replace(z))
		if matchesAll(hay, terms) {
			out = append(out, z)
		}
	}
	return out
}

func matchesAll(hay string, terms []string) bool {
	for _, t := range terms {
		if !strings.Contains(hay, t) {
			return false
		}
	}
	return true
}

// IndexOf returns the position of name in Zones, or -1.
func IndexOf(name string) int {
	for i, z := range Zones {
		if z == name {
			return i
		}
	}
	return -1
}
