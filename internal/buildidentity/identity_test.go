package buildidentity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	buildIdentityHelperEnv  = "FAIRYC_TEST_BUILD_IDENTITY_HELPER"
	buildIdentityOutputEnv  = "FAIRYC_TEST_BUILD_IDENTITY_OUTPUT"
	buildIdentityDigestSize = len(Encoding) + 1 + 64
)

func TestComputeIsExactAndDomainLabeled(t *testing.T) {
	t.Parallel()
	first, err := compute(strings.NewReader("compiler-a"))
	if err != nil {
		t.Fatal(err)
	}
	again, err := compute(strings.NewReader("compiler-a"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := compute(strings.NewReader("compiler-b"))
	if err != nil {
		t.Fatal(err)
	}
	if first != again || first == second {
		t.Fatalf("identities are not deterministic and content-sensitive: %s, %s, %s", first, again, second)
	}
	if encoded := first.String(); len(encoded) != buildIdentityDigestSize || !strings.HasPrefix(encoded, Encoding+":") {
		t.Fatalf("identity text = %q", encoded)
	}
	if _, err := compute(strings.NewReader("")); err == nil {
		t.Fatal("empty executable identity was accepted")
	}
}

func TestCurrentMatchesIndependentProcessUsingSameExecutable(t *testing.T) {
	parent, err := Current()
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(t.TempDir(), "identity.txt")
	command := exec.Command(executable, "-test.run=^TestCurrentProcessHelper$", "-test.v=false")
	command.Env = append(os.Environ(), buildIdentityHelperEnv+"=1", buildIdentityOutputEnv+"="+outputPath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("identity subprocess failed: %v\n%s", err, output)
	}
	child, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(child) != parent.String() {
		t.Fatalf("child identity = %q, parent = %q", child, parent.String())
	}
}

func TestCurrentProcessHelper(t *testing.T) {
	if os.Getenv(buildIdentityHelperEnv) == "" {
		t.Skip("subprocess helper")
	}
	identity, err := Current()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv(buildIdentityOutputEnv), []byte(identity.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}
