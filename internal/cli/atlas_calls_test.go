package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestExperimentalAtlasCallsListsExactCrossPackageCalls(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	rootMain := atlasFunctionSymbolForPackage(t, root, "example.session.app", "main")
	memberValue := atlasFunctionSymbolForPackage(t, root, "example.session.answer", "value")

	arguments := atlasCallsArguments("example.session.app", rootMain.Visible, "outgoing")
	code, first, stderr := runForTest(append([]string{"experimental", "atlas-calls"}, arguments...), root)
	if code != ExitSuccess || stderr != "" || !json.Valid([]byte(first)) || strings.HasSuffix(first, "\n") {
		t.Fatalf("workspace call catalog = (%d, %q, %q)", code, first, stderr)
	}
	catalog := decodeAtlasCalls(t, first)
	if catalog.CatalogVersion != "fairyc.workspace-call-catalog/v0" ||
		catalog.Catalog.Context.PackageUID != "example.session.app" || catalog.Catalog.Direction != "outgoing" ||
		catalog.Catalog.Subject.Symbol != rootMain.Visible || catalog.Catalog.Usage.Packages != 2 ||
		catalog.Catalog.Usage.Calls != 1 || catalog.Catalog.Usage.CrossPackageCalls != 1 ||
		catalog.Catalog.Usage.SelectedCalls != 1 || len(catalog.Catalog.Calls) != 1 || !catalog.Catalog.Complete {
		t.Fatalf("workspace call catalog = %+v", catalog)
	}
	call := catalog.Catalog.Calls[0]
	if call.Source.Symbol != rootMain.Visible || call.Target.Symbol != memberValue.Visible ||
		call.Source.PackageInstance == call.Target.PackageInstance || call.Certainty != "exact" || call.Provenance != "compiler-observed" {
		t.Fatalf("workspace call = %+v", call)
	}
	if strings.Contains(first, root) || strings.Contains(first, workspaceSessionRootSource) || strings.Contains(first, workspaceSessionMemberSource) {
		t.Fatal("workspace call catalog leaked a host path or source body")
	}

	locked := append(atlasCallsArguments("example.session.app", rootMain.Visible, "outgoing"), "--locked")
	code, lockedOutput, stderr := runForTest(append([]string{"experimental", "atlas-calls"}, locked...), root)
	if code != ExitSuccess || stderr != "" || lockedOutput != first {
		t.Fatalf("locked workspace call catalog = (%d, equal %t, %q)", code, lockedOutput == first, stderr)
	}

	incomingArguments := atlasCallsArguments("example.session.answer", memberValue.Full, "incoming")
	code, incomingOutput, stderr := runForTest(append([]string{"experimental", "atlas-calls"}, incomingArguments...), root)
	incoming := decodeAtlasCalls(t, incomingOutput)
	if code != ExitSuccess || stderr != "" || incoming.Catalog.Subject.Symbol != memberValue.Visible ||
		incoming.Catalog.Direction != "incoming" || len(incoming.Catalog.Calls) != 1 ||
		incoming.Catalog.Calls[0].Source.Symbol != rootMain.Visible {
		t.Fatalf("incoming workspace call catalog = (%d, %+v, %q)", code, incoming, stderr)
	}
}

func TestExperimentalAtlasCallsRejectsInvalidPolicySourceAndOutput(t *testing.T) {
	const source = "module main;\nimport answer as answer;\nfn i32 helper() { return 1; }\nfn i32 main() { return answer::value() + helper() - 1; }\n"
	root := writeWorkspaceSessionFixture(t, source)
	main := atlasFunctionSymbolForPackage(t, root, "example.session.app", "main")
	valid := atlasCallsArguments("example.session.app", main.Visible, "outgoing")
	for _, test := range []struct {
		name      string
		arguments []string
		contains  string
	}{
		{name: "missing package", arguments: removeAtlasCallsFlag(valid, "--package-uid"), contains: "requires --package-uid"},
		{name: "missing symbol", arguments: removeAtlasCallsFlag(valid, "--symbol"), contains: "requires --symbol"},
		{name: "missing direction", arguments: removeAtlasCallsFlag(valid, "--direction"), contains: "requires --direction"},
		{name: "duplicate", arguments: append(valid, "--locked", "--locked"), contains: "more than once"},
		{name: "unknown flag", arguments: append(valid, "--impact"), contains: "does not accept"},
		{name: "bad uid", arguments: replaceAtlasPackArgument(valid, "--package-uid", "Example"), contains: "lowercase dotted"},
		{name: "bad symbol", arguments: replaceAtlasPackArgument(valid, "--symbol", "main"), contains: "canonical visible or full"},
		{name: "bad direction", arguments: replaceAtlasPackArgument(valid, "--direction", "sideways"), contains: "does not support direction"},
		{name: "zero packages", arguments: replaceAtlasPackArgument(valid, "--max-packages", "0"), contains: "canonical decimal"},
		{name: "zero calls", arguments: replaceAtlasPackArgument(valid, "--max-calls", "0"), contains: "canonical decimal"},
		{name: "package budget", arguments: replaceAtlasPackArgument(valid, "--max-packages", "1"), contains: "packages 2 exceed 1"},
		{name: "call budget", arguments: replaceAtlasPackArgument(valid, "--max-calls", "1"), contains: "calls exceed 1"},
		{name: "tiny output", arguments: replaceAtlasPackArgument(valid, "--max-output-bytes", "1"), contains: "exceeds"},
	} {
		t.Run(test.name, func(t *testing.T) {
			code, stdout, stderr := runForTest(append([]string{"experimental", "atlas-calls"}, test.arguments...), root)
			if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, test.contains) {
				t.Fatalf("result = (%d, %q, %q), want %q", code, stdout, stderr, test.contains)
			}
		})
	}

	unknownPackage := atlasCallsArguments("example.missing", main.Visible, "outgoing")
	code, stdout, stderr := runForTest(append([]string{"experimental", "atlas-calls"}, unknownPackage...), root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "no package UID") {
		t.Fatalf("unknown package = (%d, %q, %q)", code, stdout, stderr)
	}
	unknownSymbol := replaceAtlasPackArgument(valid, "--symbol", "fysym1-sha256-128-"+strings.Repeat("0", 32))
	code, stdout, stderr = runForTest(append([]string{"experimental", "atlas-calls"}, unknownSymbol...), root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "symbol was not found") {
		t.Fatalf("unknown symbol = (%d, %q, %q)", code, stdout, stderr)
	}

	writeWorkspaceFile(t, root, "src/main.fy", "module main;\nfn i32 main( {\n")
	code, stdout, stderr = runForTest(append([]string{"experimental", "atlas-calls"}, valid...), root)
	if code != ExitPackageFailure || stdout != "" || stderr == "" {
		t.Fatalf("invalid source = (%d, %q, %q)", code, stdout, stderr)
	}
	writeWorkspaceFile(t, root, "src/main.fy", source)
	var errorText bytes.Buffer
	code = Run(append([]string{"experimental", "atlas-calls"}, valid...), root, failingWriter{}, &errorText)
	if code != ExitInternalFailure || !strings.Contains(errorText.String(), "write output failed") {
		t.Fatalf("output failure = (%d, %q)", code, errorText.String())
	}
}

func TestExperimentalAtlasCallsRejectsMissingAndUnknownActions(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	for _, test := range []struct {
		arguments []string
		contains  string
	}{
		{arguments: []string{"experimental", "atlas-calls"}, contains: `requires "list"`},
		{arguments: []string{"experimental", "atlas-calls", "impact"}, contains: "does not support action"},
	} {
		code, stdout, stderr := runForTest(test.arguments, root)
		if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, test.contains) {
			t.Fatalf("result = (%d, %q, %q), want %q", code, stdout, stderr, test.contains)
		}
	}
}

func atlasCallsArguments(packageUID, symbol, direction string) []string {
	return []string{
		"list", "--package-uid", packageUID, "--symbol", symbol, "--direction", direction,
		"--max-packages", "8", "--max-calls", "8", "--max-output-bytes", "2097152",
	}
}

func removeAtlasCallsFlag(arguments []string, flag string) []string {
	result := make([]string, 0, len(arguments)-2)
	for position := 0; position < len(arguments); position++ {
		if arguments[position] == flag && position+1 < len(arguments) {
			position++
			continue
		}
		result = append(result, arguments[position])
	}
	return result
}

func atlasFunctionSymbolForPackage(t *testing.T, root, packageUID, name string) atlasSymbolReference {
	t.Helper()
	arguments := []string{
		"list", "--package-uid", packageUID, "--kind", "function", "--name", name,
		"--max-symbols", "4", "--max-selection-bytes", "1048576", "--max-output-bytes", "2097152",
	}
	code, output, stderr := runForTest(append([]string{"experimental", "atlas-symbols"}, arguments...), root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("symbol catalog = (%d, %q, %q)", code, output, stderr)
	}
	catalog := decodeAtlasSymbols(t, output)
	if len(catalog.Catalog.Symbols) != 1 {
		t.Fatalf("package %q symbol %q catalog = %+v", packageUID, name, catalog)
	}
	return atlasSymbolReference{Visible: catalog.Catalog.Symbols[0].Symbol.Visible, Full: catalog.Catalog.Symbols[0].Symbol.Full}
}

type atlasCallsEnvelope struct {
	CatalogVersion string `json:"catalog_version"`
	Catalog        struct {
		Context struct {
			PackageUID string `json:"package_uid"`
		} `json:"context"`
		Subject struct {
			Symbol string `json:"symbol"`
		} `json:"subject"`
		Direction string `json:"direction"`
		Usage     struct {
			Packages          int `json:"packages"`
			Calls             int `json:"calls"`
			CrossPackageCalls int `json:"cross_package_calls"`
			SelectedCalls     int `json:"selected_calls"`
		} `json:"usage"`
		Calls []struct {
			Source struct {
				PackageInstance string `json:"package_instance"`
				Symbol          string `json:"symbol"`
			} `json:"source"`
			Target struct {
				PackageInstance string `json:"package_instance"`
				Symbol          string `json:"symbol"`
			} `json:"target"`
			Certainty  string `json:"certainty"`
			Provenance string `json:"provenance"`
		} `json:"calls"`
		Complete bool `json:"complete"`
	} `json:"catalog"`
}

func decodeAtlasCalls(t *testing.T, output string) atlasCallsEnvelope {
	t.Helper()
	var result atlasCallsEnvelope
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	return result
}
