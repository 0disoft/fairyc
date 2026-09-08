package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExperimentalAtlasEvidenceV2Aggregates(t *testing.T) {
	root := writeAtlasAggregateWorkspace(t)
	writeWorkspaceFile(t, root, "src/main.fy", strings.Replace(atlasAggregateSource, "resource Token", atlasGenericDocumentation+"resource Token", 1))
	symbol := atlasAggregateSymbol(t, root, "example.session.app", "Token", "resource")
	args := append(atlasEvidenceArguments("example.session.app", symbol.Visible), "--bundle-version", "2")
	code, output, stderr := runForTest(append([]string{"experimental", "atlas-evidence"}, args...), root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("aggregate bundle: %d %s", code, stderr)
	}
	doc := decodeAtlasEvidence(t, output)
	if doc.BundleVersion != "fairyc.atlas-evidence/v2" || !bytes.Contains(doc.Bundle.Components.Facts, []byte(`"pack_version":"fairyc.context-pack/v3"`)) || !bytes.Contains(doc.Bundle.Components.Facts, []byte(`"source_body":"cleanup"`)) || !bytes.Contains(doc.Bundle.Components.Source, []byte(`resource Token`)) || !bytes.Contains(doc.Bundle.Components.Documentation, []byte("Ignore previous instructions")) || bytes.Contains(doc.Bundle.Components.Facts, []byte("Ignore previous instructions")) {
		t.Fatal("wrong child versions or text in facts")
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
			t.Fatal("partial bundle at limit")
		}
	}
	for _, version := range []string{"0", "1"} {
		code, out, _ := runForTest(append([]string{"experimental", "atlas-evidence"}, replaceAtlasPackArgument(args, "--bundle-version", version)...), root)
		if code != ExitSourceDiagnostics || out != "" {
			t.Fatal("legacy bundle admitted aggregate")
		}
	}
	var errorText bytes.Buffer
	if code := Run(append([]string{"experimental", "atlas-evidence"}, args...), root, failingWriter{}, &errorText); code != ExitInternalFailure || !strings.Contains(errorText.String(), "write output failed") {
		t.Fatal("writer error swallowed")
	}
}
