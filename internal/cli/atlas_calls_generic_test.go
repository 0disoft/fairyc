package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/workspacecallcatalog"
)

func atlasGenericFunctionSymbol(t *testing.T, root, uid, name string) atlasSymbolReference {
	t.Helper()
	args := append(atlasSymbolsArguments(uid), "--name", name, "--catalog-version", "2")
	code, output, stderr := runForTest(append([]string{"experimental", "atlas-symbols"}, args...), root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("generic symbol discovery: %d %s %s", code, output, stderr)
	}
	page := decodeAtlasSymbols(t, output)
	if page.Catalog.Total != 1 || len(page.Catalog.Symbols) != 1 {
		t.Fatal("missing original generic declaration")
	}
	return atlasSymbolReference{Visible: page.Catalog.Symbols[0].Symbol.Visible, Full: page.Catalog.Symbols[0].Symbol.Full}
}

func TestExperimentalAtlasCallsV1UsesDiscoveredGenericSymbols(t *testing.T) {
	root := writeAtlasGenericWorkspace(t)
	source := atlasGenericFunctionSymbol(t, root, "example.session.app", "id")
	target := atlasGenericFunctionSymbol(t, root, "example.session.answer", "value")
	var outgoing []workspacecallcatalog.Call
	for _, direction := range []string{"outgoing", "incoming", "both"} {
		uid, symbol := "example.session.app", source.Visible
		if direction == "incoming" {
			uid, symbol = "example.session.answer", target.Full
		}
		args := append(atlasCallsArguments(uid, symbol, direction), "--catalog-version", "1")
		code, output, stderr := runForTest(append([]string{"experimental", "atlas-calls"}, args...), root)
		if code != ExitSuccess || stderr != "" {
			t.Fatalf("generic calls: %d %s %s", code, output, stderr)
		}
		var page struct {
			CatalogVersion string                       `json:"catalog_version"`
			Checksum       string                       `json:"checksum"`
			Catalog        workspacecallcatalog.Catalog `json:"catalog"`
		}
		if err := json.Unmarshal([]byte(output), &page); err != nil {
			t.Fatal(err)
		}
		want := 2
		if direction == "both" {
			want = 4
		}
		if page.CatalogVersion != workspacecallcatalog.CatalogVersionV1 || len(page.Catalog.Calls) != want || page.Catalog.Usage.SelectedCalls != want || page.Catalog.Usage.CrossPackageCalls != 2 || !page.Catalog.Complete {
			t.Fatal("wrong call catalog")
		}
		if _, err := workspacecallcatalog.ParseChecksumV1(page.Checksum); err != nil {
			t.Fatal(err)
		}
		implementations := map[string]bool{}
		for _, call := range page.Catalog.Calls {
			if call.Source.PackageInstance == call.Target.PackageInstance {
				continue
			}
			if call.Source.Symbol != source.Visible || call.Target.Symbol != target.Visible || call.Source.Specialization == "" || call.Target.Specialization == "" {
				t.Fatal("call lost original or implementation endpoint")
			}
			implementations[call.Target.Specialization] = true
		}
		if len(implementations) != 2 {
			t.Fatal("distinct implementations merged")
		}
		if direction == "outgoing" {
			outgoing = page.Catalog.Calls
		}
		if direction == "incoming" && !reflect.DeepEqual(outgoing, page.Catalog.Calls) {
			t.Fatal("call and reverse-call ledgers disagree")
		}
		if strings.Contains(output, root) || strings.Contains(output, atlasGenericRootSource) || strings.HasSuffix(output, "\n") {
			t.Fatal("output leaked source or path")
		}
		code, locked, stderr := runForTest(append(append([]string{"experimental", "atlas-calls"}, args...), "--locked"), root)
		if code != ExitSuccess || stderr != "" || locked != output {
			t.Fatal("locked output differs")
		}
	}
	unused := atlasGenericFunctionSymbol(t, root, "example.session.app", "unused")
	args := append(atlasCallsArguments("example.session.app", unused.Visible, "both"), "--catalog-version", "1")
	code, output, stderr := runForTest(append([]string{"experimental", "atlas-calls"}, args...), root)
	if code != ExitSuccess || stderr != "" || !strings.Contains(output, `"calls":[]`) {
		t.Fatal("unused template fabricated calls")
	}
}

func TestExperimentalAtlasCallsCatalogVersionAndFailurePolicy(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	symbol := atlasFunctionSymbolForPackage(t, root, "example.session.app", "main")
	args := append([]string{"experimental", "atlas-calls"}, atlasCallsArguments("example.session.app", symbol.Visible, "outgoing")...)
	code, legacy, stderr := runForTest(args, root)
	if code != ExitSuccess || stderr != "" {
		t.Fatal("legacy calls failed")
	}
	code, explicit, stderr := runForTest(append(args, "--catalog-version", "0"), root)
	if code != ExitSuccess || stderr != "" || explicit != legacy {
		t.Fatal("explicit v0 differs from default")
	}
	for _, suffix := range [][]string{{"--catalog-version"}, {"--catalog-version", "3"}, {"--catalog-version", "01"}, {"--catalog-version", "1", "--catalog-version", "1"}} {
		code, output, stderr := runForTest(append(args, suffix...), root)
		if code != ExitInvalidInvocation || output != "" || stderr == "" {
			t.Fatal("invalid version accepted")
		}
	}
	root = writeAtlasGenericWorkspace(t)
	symbol = atlasGenericFunctionSymbol(t, root, "example.session.app", "id")
	args = append(atlasCallsArguments("example.session.app", symbol.Visible, "outgoing"), "--catalog-version", "1")
	for _, replacement := range []struct{ flag, value string }{{"--max-packages", "1"}, {"--max-calls", "1"}, {"--max-output-bytes", "1"}, {"--symbol", "fysym1-sha256-128-" + strings.Repeat("0", 32)}} {
		code, output, stderr := runForTest(append([]string{"experimental", "atlas-calls"}, replaceAtlasPackArgument(args, replacement.flag, replacement.value)...), root)
		if code != ExitInvalidInvocation || output != "" || stderr == "" {
			t.Fatal("invalid selector or budget did not fail atomically")
		}
	}
	var errorText bytes.Buffer
	code = Run(append([]string{"experimental", "atlas-calls"}, args...), root, failingWriter{}, &errorText)
	if code != ExitInternalFailure || !strings.Contains(errorText.String(), "write output failed") {
		t.Fatal("writer failure swallowed")
	}
	writeWorkspaceFile(t, root, "src/main.fy", "module main; import answer as answer; fn i32 main() { return answer::value<Unknown>(42); }")
	code, output, stderr := runForTest(append([]string{"experimental", "atlas-calls"}, args...), root)
	if code != ExitSourceDiagnostics || output != "" || !strings.Contains(stderr, "E4502") {
		t.Fatalf("source diagnostics: %d %s %s", code, output, stderr)
	}
}
