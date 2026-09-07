package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/symbolcatalog"
)

const atlasGenericRootSource = "module main; import answer as answer; fn T id<T>(T x) { return answer::value<T>(x); } fn T unused<T>(T x) { return x; } fn i32 main() { assert(id<i32>(42) == 42); assert(id<u32>(7) == 7); return 0; }"

func writeAtlasGenericWorkspace(t *testing.T) string {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, atlasGenericRootSource)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", "module answer; pub fn T value<T>(T x) { return x; }")
	return root
}

func TestExperimentalAtlasSymbolsV2DiscoversGenericDeclarations(t *testing.T) {
	root := writeAtlasGenericWorkspace(t)
	for _, name := range []string{"id", "unused"} {
		args := append(atlasSymbolsArguments("example.session.app"), "--name", name, "--catalog-version", "2")
		code, output, stderr := runForTest(append([]string{"experimental", "atlas-symbols"}, args...), root)
		if code != ExitSuccess || stderr != "" {
			t.Fatalf("generic discovery: (%d, %s, %s)", code, output, stderr)
		}
		var page struct {
			CatalogVersion string                       `json:"catalog_version"`
			Checksum       string                       `json:"checksum"`
			Catalog        atlasmap.SymbolCatalogResult `json:"catalog"`
		}
		if err := json.Unmarshal([]byte(output), &page); err != nil {
			t.Fatal(err)
		}
		if page.CatalogVersion != symbolcatalog.CatalogVersionV2 || page.Catalog.Total != 1 || len(page.Catalog.Symbols) != 1 || page.Catalog.Symbols[0].Name != name || page.Catalog.Context.GenericInterfaceHash == "" {
			t.Fatal("generic source identity missing or duplicated")
		}
		if _, err := symbolcatalog.ParseChecksumV2(page.Checksum); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(output, root) || strings.Contains(output, atlasGenericRootSource) || strings.HasSuffix(output, "\n") {
			t.Fatal("catalog leaked source text or newline")
		}
		code, locked, stderr := runForTest(append(append([]string{"experimental", "atlas-symbols"}, args...), "--locked"), root)
		if code != ExitSuccess || stderr != "" || locked != output {
			t.Fatal("locked catalog changed")
		}
		var errorText bytes.Buffer
		code = Run(append([]string{"experimental", "atlas-symbols"}, args...), root, failingWriter{}, &errorText)
		if code != ExitInternalFailure || !strings.Contains(errorText.String(), "write output failed") {
			t.Fatal("output failure swallowed")
		}
	}
}

func TestExperimentalAtlasSymbolsCatalogVersionPolicy(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	args := append([]string{"experimental", "atlas-symbols"}, atlasSymbolsArguments("example.session.app")...)
	code, legacy, stderr := runForTest(args, root)
	if code != ExitSuccess || stderr != "" {
		t.Fatal("legacy fixture failed")
	}
	code, explicit, stderr := runForTest(append(args, "--catalog-version", "1"), root)
	if code != ExitSuccess || stderr != "" || explicit != legacy {
		t.Fatal("default v1 changed")
	}
	for _, suffix := range [][]string{{"--catalog-version"}, {"--catalog-version", "0"}, {"--catalog-version", "02"}, {"--catalog-version", "2", "--catalog-version", "2"}} {
		code, stdout, stderr := runForTest(append(args, suffix...), root)
		if code != ExitInvalidInvocation || stdout != "" || stderr == "" {
			t.Fatal("invalid version accepted")
		}
	}
	root = writeAtlasGenericWorkspace(t)
	writeWorkspaceFile(t, root, "src/main.fy", "module main; import answer as answer; fn i32 main() { return answer::value<Unknown>(42); }")
	code, stdout, stderr := runForTest(append(args, "--catalog-version", "2"), root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "E4502") {
		t.Fatalf("source diagnostics: (%d, %s, %s)", code, stdout, stderr)
	}
}
