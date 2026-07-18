package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/target"
)

func TestExperimentalTargetSpecEmitsDeterministicHostIndependentJSON(t *testing.T) {
	t.Parallel()
	arguments := []string{"experimental", "target-spec", string(target.X8664LinuxGNU)}
	firstCode, firstOut, firstErr := runForTest(arguments, t.TempDir())
	secondCode, secondOut, secondErr := runForTest(arguments, t.TempDir())
	if firstCode != ExitSuccess || secondCode != ExitSuccess || firstErr != "" || secondErr != "" {
		t.Fatalf("Run() first=(%d, %q), second=(%d, %q)", firstCode, firstErr, secondCode, secondErr)
	}
	if firstOut != secondOut {
		t.Fatalf("target-spec output is nondeterministic:\nfirst=%s\nsecond=%s", firstOut, secondOut)
	}
	var spec target.Spec
	if err := json.Unmarshal([]byte(firstOut), &spec); err != nil {
		t.Fatalf("target-spec output is not JSON: %v\n%s", err, firstOut)
	}
	if err := target.Validate(spec); err != nil {
		t.Fatalf("target-spec output is invalid: %v", err)
	}
	if spec.ID != target.X8664LinuxGNU || spec.Evidence.Emittable || spec.Evidence.Conformant || spec.Evidence.Supported {
		t.Fatalf("target-spec identity or evidence = %+v", spec)
	}
	for _, forbidden := range []string{"executablePath", "sdkPath", "sysrootPath", "\\\\", "/usr/bin/"} {
		if strings.Contains(firstOut, forbidden) {
			t.Fatalf("target-spec output contains host-local data %q: %s", forbidden, firstOut)
		}
	}
}

func TestExperimentalTargetSpecRejectsUnknownOrMissingTarget(t *testing.T) {
	t.Parallel()
	for _, arguments := range [][]string{
		{"experimental", "target-spec"},
		{"experimental", "target-spec", "x86_64-windows-msvc"},
		{"experimental", "target-spec", string(target.X8664LinuxGNU), "extra"},
	} {
		code, stdout, stderr := runForTest(arguments, t.TempDir())
		if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "target-spec") {
			t.Fatalf("Run(%v) = code %d, stdout %q, stderr %q", arguments, code, stdout, stderr)
		}
	}
}
