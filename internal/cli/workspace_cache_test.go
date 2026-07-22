package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

func TestExperimentalWorkspaceCacheReusesStableMIRAcrossInvocations(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	arguments := workspaceCacheArguments()
	firstCode, firstOut, firstErr := runForTest(append([]string{"experimental", "check-workspace-cache"}, arguments...), root)
	if firstCode != ExitSuccess || firstErr != "" {
		t.Fatalf("cold cache command = (%d, %q, %q)", firstCode, firstOut, firstErr)
	}
	first := decodeWorkspaceCacheReport(t, firstOut)
	if first.SchemaVersion != workspaceCacheSchema || first.Cache.Storage != workspacecompiler.StorageDisk ||
		first.Cache.Outcome != workspacecompiler.OutcomeFallback || !first.Cache.Lowered || first.Cache.Misses != 2 ||
		first.Cache.Stored != 2 || first.Cache.StoreFailed != 0 || first.Cache.RetainedEntries != 2 {
		t.Fatalf("cold cache report = %+v", first)
	}
	if !strings.HasPrefix(first.CompilerBuild, buildidentity.Encoding+":") || first.CacheDirectory != "build/fairyc-cache-test" {
		t.Fatalf("cold identity/path = %+v", first)
	}

	secondCode, secondOut, secondErr := runForTest(append([]string{"experimental", "check-workspace-cache"}, arguments...), root)
	if secondCode != ExitSuccess || secondErr != "" {
		t.Fatalf("warm cache command = (%d, %q, %q)", secondCode, secondOut, secondErr)
	}
	second := decodeWorkspaceCacheReport(t, secondOut)
	if second.CompilerBuild != first.CompilerBuild || second.Cache.Outcome != workspacecompiler.OutcomeAllHit ||
		second.Cache.Lowered || second.Cache.Hits != 2 || second.Cache.Misses != 0 || second.Cache.VerifiedHits != 2 {
		t.Fatalf("warm cache report = %+v", second)
	}
}

func TestExperimentalWorkspaceCacheRejectsCorruptionAndRecovers(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	arguments := workspaceCacheArguments()
	code, output, stderr := runForTest(append([]string{"experimental", "check-workspace-cache"}, arguments...), root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("cold cache command = (%d, %q, %q)", code, output, stderr)
	}
	cacheRoot := filepath.Join(root, "build", "fairyc-cache-test", "v1")
	var corrupted string
	err := filepath.WalkDir(cacheRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if corrupted == "" && !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			corrupted = path
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if corrupted == "" {
		t.Fatal("no cache generation found to corrupt")
	}
	if err := os.WriteFile(corrupted, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, output, stderr = runForTest(append([]string{"experimental", "check-workspace-cache"}, arguments...), root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("recovery cache command = (%d, %q, %q)", code, output, stderr)
	}
	recovered := decodeWorkspaceCacheReport(t, output)
	if recovered.Cache.Outcome != workspacecompiler.OutcomeFallback || !recovered.Cache.Lowered ||
		recovered.Cache.Hits != 1 || recovered.Cache.Misses != 1 || recovered.Cache.RecordRejected == 0 || recovered.Cache.Stored != 2 {
		t.Fatalf("recovery cache report = %+v", recovered)
	}
}

func TestExperimentalWorkspaceCacheDoesNotCreateCacheForInvalidWorkspace(t *testing.T) {
	root := t.TempDir()
	arguments := workspaceCacheArguments()
	code, output, stderr := runForTest(append([]string{"experimental", "check-workspace-cache"}, arguments...), root)
	if code != ExitPackageFailure || output != "" || stderr == "" {
		t.Fatalf("invalid workspace command = (%d, %q, %q)", code, output, stderr)
	}
	cacheRoot := filepath.Join(root, "build", "fairyc-cache-test")
	if _, err := os.Stat(cacheRoot); !os.IsNotExist(err) {
		t.Fatalf("cache path after invalid workspace = %v, want not exist", err)
	}
}

func TestParseWorkspaceCacheOptionsRejectsUnsafeOrAmbiguousPolicies(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		arguments []string
		contains  string
	}{
		{name: "missing", arguments: nil, contains: "requires --target"},
		{name: "unknown target", arguments: replaceWorkspaceCacheArgument(workspaceCacheArguments(), "--target", "unknown"), contains: "known --target"},
		{name: "outside build", arguments: replaceWorkspaceCacheArgument(workspaceCacheArguments(), "--cache-dir", ".cache/fairyc"), contains: "ignored build/"},
		{name: "path escape", arguments: replaceWorkspaceCacheArgument(workspaceCacheArguments(), "--cache-dir", "build/../outside"), contains: "canonical relative"},
		{name: "backslash", arguments: replaceWorkspaceCacheArgument(workspaceCacheArguments(), "--cache-dir", `build\cache`), contains: "forward slashes"},
		{name: "noncanonical entries", arguments: replaceWorkspaceCacheArgument(workspaceCacheArguments(), "--cache-max-entries", "08"), contains: "canonical decimal"},
		{name: "record exceeds total", arguments: replaceWorkspaceCacheArgument(workspaceCacheArguments(), "--cache-max-record-bytes", "67108865"), contains: "not to exceed --cache-max-bytes"},
		{name: "artifact exceeds record", arguments: replaceWorkspaceCacheArgument(workspaceCacheArguments(), "--cache-max-record-bytes", "1024"), contains: "not to exceed --cache-max-record-bytes"},
		{name: "temp age overflow", arguments: replaceWorkspaceCacheArgument(workspaceCacheArguments(), "--cache-temp-max-age-seconds", "31536001"), contains: "1 through 31536000"},
		{name: "duplicate", arguments: append(workspaceCacheArguments(), "--locked", "--locked"), contains: "more than once"},
		{name: "unknown", arguments: append(workspaceCacheArguments(), "--remote"), contains: "does not accept"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseWorkspaceCacheOptions(test.arguments); err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("parse error = %v, want containing %q", err, test.contains)
			}
		})
	}
}

func workspaceCacheArguments() []string {
	return []string{
		"--target", "x86_64-windows-msvc",
		"--cache-dir", "build/fairyc-cache-test",
		"--cache-max-entries", "8",
		"--cache-max-bytes", "67108864",
		"--cache-max-record-bytes", "8388608",
		"--cache-max-artifact-bytes", "4194304",
		"--cache-temp-max-age-seconds", "60",
	}
}

func replaceWorkspaceCacheArgument(arguments []string, flag, value string) []string {
	result := append([]string(nil), arguments...)
	for index := range result {
		if result[index] == flag && index+1 < len(result) {
			result[index+1] = value
			return result
		}
	}
	return result
}

func decodeWorkspaceCacheReport(t *testing.T, output string) workspaceCacheReport {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(output))
	decoder.DisallowUnknownFields()
	var report workspaceCacheReport
	if err := decoder.Decode(&report); err != nil {
		t.Fatal(err)
	}
	return report
}
