package cli

import (
	"runtime/debug"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

func TestPracticalVersion(t *testing.T) {
	var stdout, stderr strings.Builder
	code := Run([]string{"--version"}, "missing-workspace", &stdout, &stderr)
	if code != ExitSuccess || stderr.Len() != 0 || !strings.Contains(stdout.String(), "sdk: "+sdkstdlib.SDKIdentity+" "+sdkstdlib.Version+"\n") || !strings.Contains(stdout.String(), "language: "+sdkstdlib.LanguageRevision+"\n") {
		t.Fatalf("version: %d %q %q", code, stdout.String(), stderr.String())
	}
	for _, arguments := range [][]string{{"--version", "--json"}, {"--version", "extra"}} {
		stdout.Reset()
		stderr.Reset()
		if code := Run(arguments, "missing-workspace", &stdout, &stderr); code != ExitInvalidInvocation || stdout.Len() != 0 {
			t.Fatalf("arguments: %d %q", code, stdout.String())
		}
	}
	stderr.Reset()
	if code := Run([]string{"--version"}, "missing-workspace", failingWriter{}, &stderr); code != ExitInternalFailure {
		t.Fatalf("broken output: %d %q", code, stderr.String())
	}
	if !strings.Contains(usage, "fairyc --version\n") {
		t.Fatal("version absent from help")
	}
}

func TestPracticalVersionBuildIdentity(t *testing.T) {
	for _, info := range []*debug.BuildInfo{nil, {Main: debug.Module{Version: "(devel)"}}, {Settings: []debug.BuildSetting{{Key: "vcs.modified", Value: "invalid"}}}} {
		if got := versionText(info); !strings.Contains(got, "revision: unknown\nmodified: unknown\n") {
			t.Fatalf("invented identity: %q", got)
		}
	}
	info := &debug.BuildInfo{Main: debug.Module{Version: "v0.1.0"}, Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "0123456789abcdef"}, {Key: "vcs.modified", Value: "true"}}}
	if got := versionText(info); !strings.Contains(got, "compiler: v0.1.0\n") || !strings.Contains(got, "revision: 0123456789abcdef\nmodified: true\n") {
		t.Fatalf("lost build identity: %q", got)
	}
}
