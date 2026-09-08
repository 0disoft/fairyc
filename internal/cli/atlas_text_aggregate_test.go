package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/attributedtext"
)

func TestExperimentalAtlasTextExtractsOriginalAggregateSourceAndDocs(t *testing.T) {
	root := writeAtlasAggregateWorkspace(t)
	source := strings.Replace(atlasAggregateSource, "resource Token", atlasGenericDocumentation+"resource Token", 1)
	writeWorkspaceFile(t, root, "src/main.fy", source)
	symbol := atlasAggregateSymbol(t, root, "example.session.app", "Token", "resource")
	for _, action := range []string{"source", "documentation"} {
		args := atlasTextArgumentsForAction(action, "example.session.app", symbol.Visible)
		code, output, stderr := runForTest(append([]string{"experimental", "atlas-text"}, args...), root)
		if code != ExitSuccess || stderr != "" {
			t.Fatalf("aggregate text: %d %s %s", code, output, stderr)
		}
		var document struct {
			TextVersion string `json:"text_version"`
			Checksum    string `json:"checksum"`
			Text        struct {
				Authority attributedtext.Authority `json:"authority"`
				Records   []attributedtext.View    `json:"records"`
			} `json:"text"`
		}
		if err := json.Unmarshal([]byte(output), &document); err != nil {
			t.Fatal(err)
		}
		if document.TextVersion != attributedtext.DocumentVersion || document.Text.Authority != attributedtext.AuthorityNone || len(document.Text.Records) != 1 {
			t.Fatal("attribution version or authority changed")
		}
		view := document.Text.Records[0]
		if view.SymbolVisible != symbol.Visible || view.Authority != attributedtext.AuthorityNone || view.Text != source[view.StartByte:view.EndByte] {
			t.Fatal("original span lost")
		}
		if action == "source" && !strings.HasPrefix(view.Text, "resource Token<T>") {
			t.Fatal("generated resource returned")
		}
		if action == "documentation" && view.Text != strings.TrimSuffix(atlasGenericDocumentation, "\n") {
			t.Fatal("documentation changed")
		}
		if _, err := attributedtext.ParseDocumentChecksum(document.Checksum); err != nil {
			t.Fatal(err)
		}
		code, full, stderr := runForTest(append(append([]string{"experimental", "atlas-text"}, replaceAtlasPackArgument(args, "--symbol", symbol.Full)...), "--locked"), root)
		if code != ExitSuccess || stderr != "" || full != output {
			t.Fatal("locked full-ID output differs")
		}
		code, tiny, stderr := runForTest(append([]string{"experimental", "atlas-text"}, replaceAtlasPackArgument(args, "--max-output-bytes", "1")...), root)
		if code != ExitInvalidInvocation || tiny != "" || stderr == "" {
			t.Fatal("overflow not atomic")
		}
	}
	unused := atlasAggregateSymbol(t, root, "example.session.app", "Unused", "struct")
	args := atlasTextArgumentsForAction("documentation", "example.session.app", unused.Visible)
	code, output, stderr := runForTest(append([]string{"experimental", "atlas-text"}, args...), root)
	if code != ExitSuccess || stderr != "" || !strings.Contains(output, `"code":"kind-unavailable"`) {
		t.Fatal("unused docs fabricated")
	}
}
