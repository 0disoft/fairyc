package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestExperimentalAtlasSymbolsListsAndPagesExactCurrentIDs(t *testing.T) {
	const source = "module main;\nimport answer as answer;\nfn i32 helper() { return 1; }\nfn i32 main() { return answer::value() + helper() - 1; }\n"
	root := writeWorkspaceSessionFixture(t, source)
	arguments := atlasSymbolsArguments("example.session.app")
	code, first, stderr := runForTest(append([]string{"experimental", "atlas-symbols"}, arguments...), root)
	if code != ExitSuccess || stderr != "" || !json.Valid([]byte(first)) || strings.HasSuffix(first, "\n") {
		t.Fatalf("first symbol catalog = (%d, %q, %q)", code, first, stderr)
	}
	catalog := decodeAtlasSymbols(t, first)
	if catalog.CatalogVersion != "fairyc.symbol-catalog/v0" || catalog.Catalog.Context.PackageUID != "example.session.app" ||
		len(catalog.Catalog.Symbols) != 1 || catalog.Catalog.Complete || catalog.Catalog.NextAfter == "" {
		t.Fatalf("symbol catalog = %+v", catalog)
	}
	if strings.Contains(first, root) || strings.Contains(first, source) {
		t.Fatal("symbol catalog leaked a host path or source body")
	}

	nextArguments := append(atlasSymbolsArguments("example.session.app"), "--after", catalog.Catalog.NextAfter)
	code, next, stderr := runForTest(append([]string{"experimental", "atlas-symbols"}, nextArguments...), root)
	nextCatalog := decodeAtlasSymbols(t, next)
	if code != ExitSuccess || stderr != "" || len(nextCatalog.Catalog.Symbols) != 1 ||
		nextCatalog.Catalog.Symbols[0].Symbol.Full == catalog.Catalog.Symbols[0].Symbol.Full {
		t.Fatalf("next symbol catalog = (%d, %+v, %q)", code, nextCatalog, stderr)
	}

	locked := append(atlasSymbolsArguments("example.session.app"), "--locked")
	code, lockedOutput, stderr := runForTest(append([]string{"experimental", "atlas-symbols"}, locked...), root)
	if code != ExitSuccess || stderr != "" || lockedOutput != first {
		t.Fatalf("locked symbol catalog = (%d, equal %t, %q)", code, lockedOutput == first, stderr)
	}
}

func TestExperimentalAtlasSymbolsRejectsInvalidFiltersBudgetsAndOutput(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	for _, test := range []struct {
		name      string
		arguments []string
		contains  string
	}{
		{name: "missing kind", arguments: removeAtlasSymbolFlag(atlasSymbolsArguments("example.session.app"), "--kind"), contains: "at least one --kind"},
		{name: "duplicate kind", arguments: append(atlasSymbolsArguments("example.session.app"), "--kind", "function"), contains: "repeats declaration kind"},
		{name: "unknown kind", arguments: replaceAtlasPackArgument(atlasSymbolsArguments("example.session.app"), "--kind", "variable"), contains: "does not support declaration kind"},
		{name: "unknown module", arguments: append(atlasSymbolsArguments("example.session.app"), "--module", "missing"), contains: "module was not found"},
		{name: "visible cursor", arguments: append(atlasSymbolsArguments("example.session.app"), "--after", "fysym1-sha256-128-"+strings.Repeat("0", 32)), contains: "canonical full SymbolId"},
		{name: "zero symbols", arguments: replaceAtlasPackArgument(atlasSymbolsArguments("example.session.app"), "--max-symbols", "0"), contains: "canonical decimal"},
		{name: "tiny selection", arguments: replaceAtlasPackArgument(atlasSymbolsArguments("example.session.app"), "--max-selection-bytes", "1"), contains: "cannot contain"},
		{name: "tiny output", arguments: replaceAtlasPackArgument(atlasSymbolsArguments("example.session.app"), "--max-output-bytes", "1"), contains: "exceeds"},
	} {
		t.Run(test.name, func(t *testing.T) {
			code, stdout, stderr := runForTest(append([]string{"experimental", "atlas-symbols"}, test.arguments...), root)
			if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, test.contains) {
				t.Fatalf("result = (%d, %q, %q), want %q", code, stdout, stderr, test.contains)
			}
		})
	}

	var errorText bytes.Buffer
	code := Run(append([]string{"experimental", "atlas-symbols"}, atlasSymbolsArguments("example.session.app")...), root, failingWriter{}, &errorText)
	if code != ExitInternalFailure || !strings.Contains(errorText.String(), "write output failed") {
		t.Fatalf("output failure = (%d, %q)", code, errorText.String())
	}
}

func atlasSymbolsArguments(packageUID string) []string {
	return []string{
		"list", "--package-uid", packageUID, "--kind", "function",
		"--max-symbols", "1", "--max-selection-bytes", "1048576", "--max-output-bytes", "2097152",
	}
}

func removeAtlasSymbolFlag(arguments []string, flag string) []string {
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

type atlasSymbolsEnvelope struct {
	CatalogVersion string `json:"catalog_version"`
	Catalog        struct {
		Context struct {
			PackageUID string `json:"package_uid"`
		} `json:"context"`
		Symbols []struct {
			Symbol struct {
				Visible string `json:"Visible"`
				Full    string `json:"Full"`
			} `json:"symbol"`
			Name string `json:"name"`
		} `json:"symbols"`
		NextAfter string `json:"next_after"`
		Complete  bool   `json:"complete"`
	} `json:"catalog"`
}

func decodeAtlasSymbols(t *testing.T, output string) atlasSymbolsEnvelope {
	t.Helper()
	var result atlasSymbolsEnvelope
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	return result
}
