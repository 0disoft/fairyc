package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/target"
)

func TestExperimentalTargetSpecEmitsDeterministicHostIndependentJSON(t *testing.T) {
	t.Parallel()
	for _, id := range []target.ID{target.X8664LinuxGNU, target.X8664WindowsMSVC} {
		arguments := []string{"experimental", "target-spec", string(id)}
		firstCode, firstOut, firstErr := runForTest(arguments, t.TempDir())
		secondCode, secondOut, secondErr := runForTest(arguments, t.TempDir())
		if firstCode != ExitSuccess || secondCode != ExitSuccess || firstErr != "" || secondErr != "" {
			t.Fatalf("Run(%q) first=(%d, %q), second=(%d, %q)", id, firstCode, firstErr, secondCode, secondErr)
		}
		if firstOut != secondOut {
			t.Fatalf("target-spec %q output is nondeterministic:\nfirst=%s\nsecond=%s", id, firstOut, secondOut)
		}
		var spec target.Spec
		if err := json.Unmarshal([]byte(firstOut), &spec); err != nil {
			t.Fatalf("target-spec %q output is not JSON: %v\n%s", id, err, firstOut)
		}
		if err := target.Validate(spec); err != nil {
			t.Fatalf("target-spec %q output is invalid: %v", id, err)
		}
		if spec.ID != id || spec.Evidence.Emittable || spec.Evidence.Conformant || spec.Evidence.Supported {
			t.Fatalf("target-spec %q identity or evidence = %+v", id, spec)
		}
		for _, forbidden := range []string{"executablePath", "sdkPath", "sysrootPath", "C:\\", "/usr/bin/"} {
			if strings.Contains(firstOut, forbidden) {
				t.Fatalf("target-spec %q output contains host-local data %q: %s", id, forbidden, firstOut)
			}
		}
	}
}

func TestStableTargetsReportsKnownMaturityWithoutReadingSource(t *testing.T) {
	t.Parallel()
	missingRoot := filepath.Join(t.TempDir(), "missing")
	firstCode, firstOut, firstErr := runForTest([]string{"targets"}, missingRoot)
	secondCode, secondOut, secondErr := runForTest([]string{"targets"}, missingRoot)
	if firstCode != ExitSuccess || secondCode != ExitSuccess || firstErr != "" || secondErr != "" {
		t.Fatalf("Run() first=(%d, %q), second=(%d, %q)", firstCode, firstErr, secondCode, secondErr)
	}
	if firstOut != secondOut {
		t.Fatalf("targets output is nondeterministic:\nfirst=%s\nsecond=%s", firstOut, secondOut)
	}
	want := `x86_64-linux-gnu
  schema: fairyc.target-spec/v0
  emission subset: scalar-and-flat-copy-resource-tagged-c11
  emittable: no
  conformant: no
  supported: no
  missing evidence:
    - complete-runtime
    - complete-standard-library
    - toolchain-baseline
    - toolchain-provisioning

x86_64-windows-msvc
  schema: fairyc.target-spec/v0
  emission subset: scalar-and-llp64-c-scalar-c11
  emittable: no
  conformant: no
  supported: no
  missing evidence:
    - bidirectional-c-abi-fixtures
    - c-layout-probes
    - complete-runtime
    - complete-standard-library
    - pe-coff-object-fixtures
    - toolchain-baseline
    - toolchain-provisioning
    - windows-x64-unwind-fixtures
`
	if firstOut != want {
		t.Fatalf("targets output = %q, want %q", firstOut, want)
	}
	for _, forbidden := range []string{"C:\\", "/usr/bin/", "Program Files"} {
		if strings.Contains(firstOut, forbidden) {
			t.Fatalf("targets output contains undeclared or host-local value %q: %s", forbidden, firstOut)
		}
	}
}

func TestStableTargetsRejectsArgumentsAndHandlesHelpAndOutputFailure(t *testing.T) {
	t.Parallel()
	code, stdout, stderr := runForTest([]string{"targets", "x86_64-linux-gnu"}, t.TempDir())
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "does not accept arguments") {
		t.Fatalf("targets argument = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	for _, flag := range []string{"-h", "--help"} {
		code, stdout, stderr = runForTest([]string{"targets", flag}, t.TempDir())
		if code != ExitSuccess || !strings.Contains(stdout, "fairyc targets") || stderr != "" {
			t.Fatalf("targets %s = code %d, stdout %q, stderr %q", flag, code, stdout, stderr)
		}
	}
	var failure bytes.Buffer
	code = Run([]string{"targets"}, t.TempDir(), failingWriter{}, &failure)
	if code != ExitInternalFailure || !strings.Contains(failure.String(), "write output failed") {
		t.Fatalf("targets output failure = code %d, stderr %q", code, failure.String())
	}
}

func TestExperimentalTargetSpecRejectsUnknownOrMissingTarget(t *testing.T) {
	t.Parallel()
	for _, arguments := range [][]string{
		{"experimental", "target-spec"},
		{"experimental", "target-spec", "aarch64-windows-msvc"},
		{"experimental", "target-spec", string(target.X8664LinuxGNU), "extra"},
	} {
		code, stdout, stderr := runForTest(arguments, t.TempDir())
		if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "target-spec") {
			t.Fatalf("Run(%v) = code %d, stdout %q, stderr %q", arguments, code, stdout, stderr)
		}
	}
}
