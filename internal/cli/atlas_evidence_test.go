package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestExperimentalAtlasEvidenceShowComposesExactFactsSourceAndDocumentation(t *testing.T) {
	const source = "module main;\n/// " + atlasTextPromptLikeComment + "\nfn i32 main() { return helper(); }\nfn i32 helper() { return 42; }\n"
	root := writeWorkspaceSessionFixture(t, source)
	main := atlasTextSymbolByName(t, root, "main")
	arguments := atlasEvidenceArguments("example.session.app", main.Visible)

	code, first, stderr := runForTest(append([]string{"experimental", "atlas-evidence"}, arguments...), root)
	if code != ExitSuccess || stderr != "" || !json.Valid([]byte(first)) || strings.HasSuffix(first, "\n") {
		t.Fatalf("first evidence bundle = (%d, %q, %q)", code, first, stderr)
	}
	bundle := decodeAtlasEvidence(t, first)
	if bundle.BundleVersion != "fairyc.atlas-evidence/v0" || bundle.Encoding != "fairyc.atlas-evidence-json/v0" ||
		bundle.Bundle.Authority.Bundle != "none" || bundle.Bundle.Authority.Facts != "compiler-fact" ||
		bundle.Bundle.Authority.Source != "untrusted-text" || bundle.Bundle.Authority.Documentation != "untrusted-text" ||
		bundle.Bundle.Subject.SymbolVisible != main.Visible || bundle.Bundle.Subject.SymbolFull != main.Full ||
		bundle.Bundle.Subject.PackageUID != "example.session.app" {
		t.Fatalf("evidence bundle = %+v", bundle)
	}
	if !json.Valid(bundle.Bundle.Components.Facts) || !json.Valid(bundle.Bundle.Components.Source) ||
		!json.Valid(bundle.Bundle.Components.Documentation) ||
		!bytes.Contains(bundle.Bundle.Components.Facts, []byte(`"pack_version":"fairyc.context-pack/v1"`)) ||
		!bytes.Contains(bundle.Bundle.Components.Facts, []byte(`"documentation":[]`)) ||
		!bytes.Contains(bundle.Bundle.Components.Facts, []byte(`"source_excerpts":[]`)) {
		t.Fatalf("facts component = %s", bundle.Bundle.Components.Facts)
	}
	var sourceDocument, documentationDocument atlasTextEnvelope
	if err := json.Unmarshal(bundle.Bundle.Components.Source, &sourceDocument); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(bundle.Bundle.Components.Documentation, &documentationDocument); err != nil {
		t.Fatal(err)
	}
	if sourceDocument.Text.Authority != "untrusted-text" || len(sourceDocument.Text.Records) != 1 ||
		!strings.Contains(sourceDocument.Text.Records[0].Text, "fn i32 main()") ||
		strings.Contains(sourceDocument.Text.Records[0].Text, atlasTextPromptLikeComment) {
		t.Fatalf("source component = %+v", sourceDocument)
	}
	if documentationDocument.Text.Authority != "untrusted-text" || len(documentationDocument.Text.Records) != 1 ||
		documentationDocument.Text.Records[0].Text != "/// "+atlasTextPromptLikeComment ||
		strings.Contains(documentationDocument.Text.Records[0].Text, "fn i32 main()") {
		t.Fatalf("documentation component = %+v", documentationDocument)
	}
	if strings.Contains(first, root) {
		t.Fatal("evidence bundle leaked an absolute workspace path")
	}

	fullArguments := atlasEvidenceArguments("example.session.app", main.Full)
	code, full, stderr := runForTest(append([]string{"experimental", "atlas-evidence"}, fullArguments...), root)
	if code != ExitSuccess || stderr != "" || full != first {
		t.Fatalf("full SymbolId bundle = (%d, equal %t, %q)", code, full == first, stderr)
	}
	lockedArguments := append(atlasEvidenceArguments("example.session.app", main.Visible), "--locked")
	code, locked, stderr := runForTest(append([]string{"experimental", "atlas-evidence"}, lockedArguments...), root)
	if code != ExitSuccess || stderr != "" || locked != first {
		t.Fatalf("locked evidence bundle = (%d, equal %t, %q)", code, locked == first, stderr)
	}

	helper := atlasTextSymbolByName(t, root, "helper")
	code, undocumented, stderr := runForTest(
		append([]string{"experimental", "atlas-evidence"}, atlasEvidenceArguments("example.session.app", helper.Visible)...), root,
	)
	if code != ExitSuccess || stderr != "" || !strings.Contains(undocumented, `"code":"kind-unavailable"`) {
		t.Fatalf("undocumented evidence bundle = (%d, %q, %q)", code, undocumented, stderr)
	}

	limitedArguments := replaceAtlasPackArgument(arguments, "--max-fragment-bytes", "1")
	limitedArguments = replaceAtlasPackArgument(limitedArguments, "--max-text-bytes", "1")
	code, limited, stderr := runForTest(append([]string{"experimental", "atlas-evidence"}, limitedArguments...), root)
	if code != ExitSuccess || stderr != "" || strings.Contains(limited, "fn i32 main()") ||
		strings.Contains(limited, atlasTextPromptLikeComment) ||
		strings.Count(limited, `"code":"fragment-byte-limit"`) != 2 {
		t.Fatalf("limited evidence bundle = (%d, %q, %q)", code, limited, stderr)
	}
}

func TestExperimentalAtlasEvidenceShowRejectsInvalidPolicyAndSeparatesFailures(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	main := atlasTextSymbolByName(t, root, "main")
	arguments := atlasEvidenceArguments("example.session.app", main.Visible)
	for _, test := range []struct {
		name      string
		arguments []string
		contains  string
	}{
		{name: "missing custom limit", arguments: removeAtlasTextFlag(arguments, "--max-facts-bytes"), contains: "requires --max-facts-bytes"},
		{name: "missing base value before custom limit", arguments: append([]string{"show", "--package-uid"}, arguments[3:]...), contains: "requires a value for --package-uid"},
		{name: "duplicate custom limit", arguments: append(arguments, "--max-facts-bytes", "1"), contains: "more than once"},
		{name: "fragment guard", arguments: replaceAtlasPackArgument(arguments, "--max-fragment-bytes", "8388609"), contains: "1 through 8388608"},
		{name: "fragment exceeds text", arguments: replaceAtlasPackArgument(arguments, "--max-text-bytes", "1"), contains: "no greater than"},
		{name: "tiny facts", arguments: replaceAtlasPackArgument(arguments, "--max-facts-bytes", "1"), contains: "Context Pack exceeds"},
		{name: "tiny text document", arguments: replaceAtlasPackArgument(arguments, "--max-text-document-bytes", "1"), contains: "attributed-text document exceeds"},
		{name: "tiny final output", arguments: replaceAtlasPackArgument(arguments, "--max-output-bytes", "1"), contains: "Evidence Bundle exceeds"},
		{name: "unknown package", arguments: atlasEvidenceArguments("example.missing", main.Visible), contains: "no package UID"},
		{name: "unknown symbol", arguments: atlasEvidenceArguments("example.session.app", "fysym1-sha256-256-"+strings.Repeat("0", 64)), contains: "not found"},
		{name: "unknown flag", arguments: append(arguments, "--pretty"), contains: "does not accept"},
	} {
		t.Run(test.name, func(t *testing.T) {
			code, stdout, stderr := runForTest(append([]string{"experimental", "atlas-evidence"}, test.arguments...), root)
			if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, test.contains) {
				t.Fatalf("result = (%d, %q, %q), want %q", code, stdout, stderr, test.contains)
			}
		})
	}
	for _, test := range []struct {
		name      string
		arguments []string
		contains  string
	}{
		{name: "missing action", arguments: []string{"experimental", "atlas-evidence"}, contains: `requires "show"`},
		{name: "unknown action", arguments: []string{"experimental", "atlas-evidence", "around"}, contains: "does not support action"},
	} {
		t.Run(test.name, func(t *testing.T) {
			code, stdout, stderr := runForTest(test.arguments, root)
			if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, test.contains) {
				t.Fatalf("result = (%d, %q, %q), want %q", code, stdout, stderr, test.contains)
			}
		})
	}

	var errorText bytes.Buffer
	code := Run(append([]string{"experimental", "atlas-evidence"}, arguments...), root, failingWriter{}, &errorText)
	if code != ExitInternalFailure || !strings.Contains(errorText.String(), "write output failed") {
		t.Fatalf("output failure = (%d, %q)", code, errorText.String())
	}
}

func atlasEvidenceArguments(packageUID, symbol string) []string {
	return []string{
		"show", "--package-uid", packageUID, "--symbol", symbol,
		"--direction", "outgoing", "--relation", "calls",
		"--max-records", "64", "--max-relations", "64", "--max-selection-bytes", "1048576",
		"--max-fragment-bytes", "1048576", "--max-text-bytes", "1048576",
		"--max-facts-bytes", "2097152", "--max-text-document-bytes", "2097152",
		"--max-output-bytes", "8388608",
	}
}

type atlasEvidenceEnvelope struct {
	BundleVersion string `json:"bundle_version"`
	Encoding      string `json:"encoding"`
	Bundle        struct {
		Authority struct {
			Bundle        string `json:"bundle"`
			Facts         string `json:"facts"`
			Source        string `json:"source"`
			Documentation string `json:"documentation"`
		} `json:"authority"`
		Subject struct {
			SymbolVisible string `json:"symbol_visible"`
			SymbolFull    string `json:"symbol_full"`
			PackageUID    string `json:"package_uid"`
		} `json:"subject"`
		Components struct {
			Facts         json.RawMessage `json:"facts"`
			Source        json.RawMessage `json:"source"`
			Documentation json.RawMessage `json:"documentation"`
		} `json:"components"`
	} `json:"bundle"`
}

func decodeAtlasEvidence(t *testing.T, output string) atlasEvidenceEnvelope {
	t.Helper()
	var result atlasEvidenceEnvelope
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	return result
}
