package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestExperimentalAtlasPackOverviewEmitsDeterministicCurrentPackagePack(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	arguments := atlasPackOverviewArguments("example.session.app")
	code, first, stderr := runForTest(append([]string{"experimental", "atlas-pack"}, arguments...), root)
	if code != ExitSuccess || stderr != "" || !json.Valid([]byte(first)) || strings.HasSuffix(first, "\n") {
		t.Fatalf("first atlas pack = (%d, %q, %q)", code, first, stderr)
	}
	pack := decodeAtlasPackOverview(t, first)
	if pack.PackVersion != "fairyc.context-pack/v1" || pack.Pack.Query.Kind != "overview" ||
		pack.Pack.PackageIdentity.UID != "example.session.app" || pack.Pack.Budget.MaxRecords != 64 ||
		pack.Pack.Usage.Records == 0 || len(pack.Pack.Documentation) != 0 || len(pack.Pack.SourceExcerpts) != 0 {
		t.Fatalf("atlas pack = %+v", pack)
	}
	if strings.Contains(first, root) || strings.Contains(first, workspaceSessionRootSource) {
		t.Fatal("atlas pack leaked a host path or source body")
	}

	locked := append(atlasPackOverviewArguments("example.session.app"), "--locked")
	code, second, stderr := runForTest(append([]string{"experimental", "atlas-pack"}, locked...), root)
	if code != ExitSuccess || stderr != "" || second != first {
		t.Fatalf("locked atlas pack = (%d, equal %t, %q)", code, second == first, stderr)
	}

	member := atlasPackOverviewArguments("example.session.answer")
	code, output, stderr := runForTest(append([]string{"experimental", "atlas-pack"}, member...), root)
	if code != ExitSuccess || stderr != "" || decodeAtlasPackOverview(t, output).Pack.PackageIdentity.UID != "example.session.answer" {
		t.Fatalf("member atlas pack = (%d, %q, %q)", code, output, stderr)
	}
}

func TestExperimentalAtlasPackOverviewSeparatesInvocationSourceAndOutputFailures(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	unknown := atlasPackOverviewArguments("example.missing")
	code, stdout, stderr := runForTest(append([]string{"experimental", "atlas-pack"}, unknown...), root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "no package UID") {
		t.Fatalf("unknown package = (%d, %q, %q)", code, stdout, stderr)
	}

	tiny := replaceAtlasPackArgument(atlasPackOverviewArguments("example.session.app"), "--max-output-bytes", "1")
	code, stdout, stderr = runForTest(append([]string{"experimental", "atlas-pack"}, tiny...), root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "exceeds") {
		t.Fatalf("small output budget = (%d, %q, %q)", code, stdout, stderr)
	}

	writeWorkspaceFile(t, root, "src/main.fy", "module main;\nfn i32 main( {\n")
	code, stdout, stderr = runForTest(append([]string{"experimental", "atlas-pack"}, atlasPackOverviewArguments("example.session.app")...), root)
	if code != ExitPackageFailure || stdout != "" || stderr == "" {
		t.Fatalf("invalid source = (%d, %q, %q)", code, stdout, stderr)
	}

	writeWorkspaceFile(t, root, "src/main.fy", workspaceSessionRootSource)
	var errorText bytes.Buffer
	code = Run(append([]string{"experimental", "atlas-pack"}, atlasPackOverviewArguments("example.session.app")...), root, failingWriter{}, &errorText)
	if code != ExitInternalFailure || !strings.Contains(errorText.String(), "write output failed") {
		t.Fatalf("output failure = (%d, %q)", code, errorText.String())
	}
}

func TestParseAtlasPackOverviewOptionsRejectsMissingDuplicateAndNoncanonicalPolicy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		arguments []string
		contains  string
	}{
		{name: "missing", arguments: nil, contains: "requires --package-uid"},
		{name: "bad uid", arguments: replaceAtlasPackArgument(atlasPackOverviewArguments("example.session.app"), "--package-uid", "Example"), contains: "lowercase dotted"},
		{name: "zero records", arguments: replaceAtlasPackArgument(atlasPackOverviewArguments("example.session.app"), "--max-records", "0"), contains: "1 through 65536"},
		{name: "noncanonical records", arguments: replaceAtlasPackArgument(atlasPackOverviewArguments("example.session.app"), "--max-records", "064"), contains: "canonical decimal"},
		{name: "selection overflow", arguments: replaceAtlasPackArgument(atlasPackOverviewArguments("example.session.app"), "--max-selection-bytes", "1073741825"), contains: "1 through 1073741824"},
		{name: "output overflow", arguments: replaceAtlasPackArgument(atlasPackOverviewArguments("example.session.app"), "--max-output-bytes", "1073741825"), contains: "1 through 1073741824"},
		{name: "duplicate", arguments: append(atlasPackOverviewArguments("example.session.app"), "--locked", "--locked"), contains: "more than once"},
		{name: "unknown", arguments: append(atlasPackOverviewArguments("example.session.app"), "--pretty"), contains: "does not accept"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			arguments := test.arguments
			if len(arguments) != 0 && arguments[0] == "overview" {
				arguments = arguments[1:]
			}
			if _, err := parseAtlasPackOverviewOptions(arguments); err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("parse error = %v, want containing %q", err, test.contains)
			}
		})
	}
}

func atlasPackOverviewArguments(packageUID string) []string {
	return []string{
		"overview",
		"--package-uid", packageUID,
		"--max-records", "64",
		"--max-selection-bytes", "1048576",
		"--max-output-bytes", "2097152",
	}
}

func replaceAtlasPackArgument(arguments []string, flag, value string) []string {
	result := append([]string(nil), arguments...)
	for index := range result {
		if result[index] == flag && index+1 < len(result) {
			result[index+1] = value
			return result
		}
	}
	return result
}

type atlasPackOverviewEnvelope struct {
	PackVersion string `json:"pack_version"`
	Pack        struct {
		PackageIdentity struct {
			UID string `json:"uid"`
		} `json:"package_identity"`
		Query struct {
			Kind string `json:"kind"`
		} `json:"query"`
		Budget struct {
			MaxRecords int `json:"max_records"`
		} `json:"budget"`
		Usage struct {
			Records int `json:"records"`
		} `json:"usage"`
		Documentation  []json.RawMessage `json:"documentation"`
		SourceExcerpts []json.RawMessage `json:"source_excerpts"`
	} `json:"pack"`
}

func decodeAtlasPackOverview(t *testing.T, output string) atlasPackOverviewEnvelope {
	t.Helper()
	var result atlasPackOverviewEnvelope
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	return result
}
