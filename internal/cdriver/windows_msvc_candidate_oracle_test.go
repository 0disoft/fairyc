package cdriver

import (
	"os"
	"strings"
	"testing"
)

func TestCandidateMSVCToolchainOracle(t *testing.T) {
	path := os.Getenv("FAIRYC_MSVC_ORACLE")
	if path == "" {
		t.Skip("explicit candidate discovery oracle only")
	}
	toolchain, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	// go test may prepend its downloaded Go toolchain to the inherited PATH.
	// Compare the two compiler-owned entries independently of that test detail.
	search := strings.Split(environmentValue(toolchain.Environment, "PATH"), ";")
	if len(search) < 3 {
		t.Fatal("missing compiler-owned search path entries")
	}
	values := []string{toolchain.Path, toolchain.Version, environmentValue(toolchain.Environment, "INCLUDE"), environmentValue(toolchain.Environment, "LIB"), strings.Join(search[:2], ";") + ";"}
	if err := os.WriteFile(path, []byte(strings.Join(values, "\n")+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
}
