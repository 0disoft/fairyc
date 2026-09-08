package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExperimentalAtlasEvidenceV1GenericAndLegacy(t *testing.T) {
	root := writeAtlasGenericWorkspace(t)
	writeWorkspaceFile(t, root, "src/main.fy", strings.Replace(atlasGenericRootSource, "fn T id", "\n"+atlasGenericDocumentation+"fn T id", 1))
	symbol := atlasGenericFunctionSymbol(t, root, "example.session.app", "id")
	args := append(atlasEvidenceArguments("example.session.app", symbol.Visible), "--bundle-version", "1")
	code, output, stderr := runForTest(append([]string{"experimental", "atlas-evidence"}, args...), root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("generic bundle: %d %s", code, stderr)
	}
	doc := decodeAtlasEvidence(t, output)
	if doc.BundleVersion != "fairyc.atlas-evidence/v1" || !bytes.Contains(doc.Bundle.Components.Facts, []byte(`"pack_version":"fairyc.context-pack/v2"`)) || !bytes.Contains(doc.Bundle.Components.Source, []byte(`fn T id`)) || !bytes.Contains(doc.Bundle.Components.Documentation, []byte("Ignore previous instructions")) {
		t.Fatal("wrong components")
	}
	for _, variant := range [][]string{append(append([]string{}, args...), "--locked"), replaceAtlasPackArgument(args, "--symbol", symbol.Full)} {
		code, again, stderr := runForTest(append([]string{"experimental", "atlas-evidence"}, variant...), root)
		if code != ExitSuccess || stderr != "" || again != output {
			t.Fatal("locked/full output differs")
		}
	}
	for _, flag := range []string{"--max-facts-bytes", "--max-text-document-bytes", "--max-output-bytes"} {
		code, out, stderr := runForTest(append([]string{"experimental", "atlas-evidence"}, replaceAtlasPackArgument(args, flag, "1")...), root)
		if code != ExitInvalidInvocation || out != "" || stderr == "" {
			t.Fatal("limit emitted partial bundle")
		}
	}
	var errorText bytes.Buffer
	if code := Run(append([]string{"experimental", "atlas-evidence"}, args...), root, failingWriter{}, &errorText); code != ExitInternalFailure || !strings.Contains(errorText.String(), "write output failed") {
		t.Fatal("writer error swallowed")
	}
	for _, version := range [][]string{{"--bundle-version"}, {"--bundle-version", "01"}, {"--bundle-version", "3"}, {"--bundle-version", "1", "--bundle-version", "0"}, {"--pack-version", "2"}} {
		bad := append(atlasEvidenceArguments("example.session.app", symbol.Visible), version...)
		code, out, _ := runForTest(append([]string{"experimental", "atlas-evidence"}, bad...), t.TempDir())
		if code != ExitInvalidInvocation || out != "" {
			t.Fatal("invalid version reached workspace")
		}
	}
	ordinary := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	main := atlasTextSymbolByName(t, ordinary, "main")
	legacy := atlasEvidenceArguments("example.session.app", main.Visible)
	code, before, stderr := runForTest(append([]string{"experimental", "atlas-evidence"}, legacy...), ordinary)
	if code != ExitSuccess || stderr != "" {
		t.Fatal("default bundle failed")
	}
	code, after, stderr := runForTest(append(append([]string{"experimental", "atlas-evidence"}, legacy...), "--bundle-version", "0"), ordinary)
	if code != ExitSuccess || stderr != "" || before != after {
		t.Fatal("explicit v0 changed bytes")
	}
}
