package cbackend_test

import "regexp"

// Syntax sugar shifts byte ranges but must leave executable operations identical.
// Native location tests separately verify the exact, unnormalized payloads.
func withoutTrapLocations(source string) string {
	pattern := regexp.MustCompile(`fy_runtime_trap_at_v0\(([^,\n]+), "(?:\\[0-7]{3})*", UINT32_C\(\d+\), UINT64_C\(\d+\), UINT64_C\(\d+\)\)`)
	return pattern.ReplaceAllString(source, "fy_runtime_trap($1)")
}
