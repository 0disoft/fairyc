package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateFetchCLI(t *testing.T) {
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			// Offline restoration and check use only the authored native CLI. Make
			// the separately built companion unavailable throughout this replay.
			companion := filepath.Join(filepath.Dir(image), "fairyc-candidate-build.exe")
			if err := os.Remove(companion); err != nil {
				t.Fatal(err)
			}
			verifyCandidateFetchCLI(t, image)
			root := remoteCLIWorkspace(t)
			code, out, diagnostic := fetchCLIProcess(t, image, root, []string{"experimental", "fetch-packages", "--offline"}, "")
			if code != 0 || diagnostic != "" || !strings.Contains(out, "1 installed") {
				t.Fatalf("restore: %d %q %q", code, out, diagnostic)
			}
			for _, args := range [][]string{{"check"}, {"check", "--locked"}} {
				code, out, diagnostic = fetchCLIProcess(t, image, root, args, "")
				if code != 0 || diagnostic != "" || !strings.HasPrefix(out, "check ok:") {
					t.Fatalf("restored workspace %v: %d %q %q", args, code, out, diagnostic)
				}
			}
		})
	}
}
