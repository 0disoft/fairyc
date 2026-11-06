package cli

import (
	"strings"
	"testing"
)

// Required and optional flags come from the independent Go command policy.
func TestLinuxCLIHelp(t *testing.T) {
	help := linuxFetchHelp(t)
	for _, command := range []string{"fetch-packages", "atlas-impact", "atlas-mcp", "atlas-symbols list", "atlas-calls list", "atlas-pack", "atlas-edit replace-body"} {
		prefix := "fairyc experimental " + command + " "
		found := false
		for _, line := range strings.Split(usage, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, prefix) {
				found = true
				want := strings.Replace(line, "fairyc ", "lexer-candidate-linux ", 1) + "\n"
				if !strings.Contains(help, want) {
					t.Fatalf("Linux help does not match Go command policy: %q", want)
				}
			}
		}
		if !found {
			t.Fatalf("missing Go help owner for %q", command)
		}
	}
}
