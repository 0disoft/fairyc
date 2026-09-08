package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/symbolcatalog"
	"github.com/0disoft/fairyc/internal/workspacecallcatalog"
)

const atlasAggregateSource = `module main; import answer as answer;
struct Cell<T> { T value; }
struct Unused<T> { T value; }
resource Token<T> { Cell<T> payload; cleanup { let T observed = answer::value<T>(this.payload.value); } }
fn Token<T> make<T>(T value) { return Token<T> { .payload = Cell<T> { .value = value } }; }
fn i32 main() { let Token<i32> a = make<i32>(42); let Token<u32> b = make<u32>(7); return a.payload.value; }`

func writeAtlasAggregateWorkspace(t *testing.T) string {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, atlasAggregateSource)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", "module answer; pub fn T value<T>(T x) { return x; }")
	return root
}

func atlasAggregateSymbol(t *testing.T, root, uid, name, kind string) atlasSymbolReference {
	t.Helper()
	args := append(replaceAtlasPackArgument(atlasSymbolsArguments(uid), "--kind", kind), "--name", name, "--catalog-version", "3")
	code, output, stderr := runForTest(append([]string{"experimental", "atlas-symbols"}, args...), root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("aggregate discovery: %d %s %s", code, output, stderr)
	}
	var page struct {
		CatalogVersion string                       `json:"catalog_version"`
		Checksum       string                       `json:"checksum"`
		Catalog        atlasmap.SymbolCatalogResult `json:"catalog"`
	}
	if err := json.Unmarshal([]byte(output), &page); err != nil {
		t.Fatal(err)
	}
	if page.CatalogVersion != symbolcatalog.CatalogVersionV3 || page.Catalog.Context.LogicalSchema != atlasmap.LogicalSchemaV3 || page.Catalog.Total != 1 || len(page.Catalog.Symbols) != 1 || page.Catalog.Symbols[0].Name != name || string(page.Catalog.Symbols[0].Kind) != kind {
		t.Fatal("aggregate discovery lost original kind")
	}
	if _, err := symbolcatalog.ParseChecksumV3(page.Checksum); err != nil {
		t.Fatal(err)
	}
	code, locked, stderr := runForTest(append(append([]string{"experimental", "atlas-symbols"}, args...), "--locked"), root)
	if code != ExitSuccess || stderr != "" || locked != output {
		t.Fatal("locked symbols differ")
	}
	return atlasSymbolReference{Visible: page.Catalog.Symbols[0].Symbol.Visible, Full: page.Catalog.Symbols[0].Symbol.Full}
}

func TestExperimentalAtlasAggregateSymbolsFeedCleanupCallCLI(t *testing.T) {
	root := writeAtlasAggregateWorkspace(t)
	source := atlasAggregateSymbol(t, root, "example.session.app", "Token", "resource")
	atlasAggregateSymbol(t, root, "example.session.app", "Cell", "struct")
	unused := atlasAggregateSymbol(t, root, "example.session.app", "Unused", "struct")
	target := atlasAggregateSymbol(t, root, "example.session.answer", "value", "function")
	lockPath := filepath.Join(root, "fairy.lock")
	lockBefore, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	var outgoing []workspacecallcatalog.Call
	for _, direction := range []string{"outgoing", "incoming", "both"} {
		uid, symbol := "example.session.app", source.Visible
		if direction == "incoming" {
			uid, symbol = "example.session.answer", target.Full
		}
		args := append(atlasCallsArguments(uid, symbol, direction), "--catalog-version", "2", "--locked")
		code, output, stderr := runForTest(append([]string{"experimental", "atlas-calls"}, args...), root)
		if code != ExitSuccess || stderr != "" {
			t.Fatalf("aggregate calls: %d %s %s", code, output, stderr)
		}
		var page struct {
			CatalogVersion string                       `json:"catalog_version"`
			Checksum       string                       `json:"checksum"`
			Catalog        workspacecallcatalog.Catalog `json:"catalog"`
		}
		if err := json.Unmarshal([]byte(output), &page); err != nil {
			t.Fatal(err)
		}
		if page.CatalogVersion != workspacecallcatalog.CatalogVersionV2 || len(page.Catalog.Calls) != 2 || page.Catalog.Usage.CrossPackageCalls != 2 {
			t.Fatal("wrong cleanup catalog")
		}
		if _, err := workspacecallcatalog.ParseChecksumV2(page.Checksum); err != nil {
			t.Fatal(err)
		}
		ids := map[string]bool{}
		for _, call := range page.Catalog.Calls {
			if call.SourceBody != "cleanup" || call.Source.Symbol != source.Visible || call.Target.Symbol != target.Visible || call.Source.Specialization == "" || call.Target.Specialization == "" {
				t.Fatal("cleanup endpoints lost")
			}
			ids[call.Source.Specialization] = true
		}
		if len(ids) != 2 {
			t.Fatal("specializations merged")
		}
		if direction == "outgoing" {
			outgoing = page.Catalog.Calls
		} else if !reflect.DeepEqual(outgoing, page.Catalog.Calls) {
			t.Fatal("directions disagree")
		}
		if strings.Contains(output, root) || strings.Contains(output, atlasAggregateSource) || strings.HasSuffix(output, "\n") {
			t.Fatal("source/path/newline leak")
		}
	}
	args := append(atlasCallsArguments("example.session.app", unused.Visible, "both"), "--catalog-version", "2", "--locked")
	code, output, stderr := runForTest(append([]string{"experimental", "atlas-calls"}, args...), root)
	if code != ExitSuccess || stderr != "" || !strings.Contains(output, `"calls":[]`) {
		t.Fatal("unused template fabricated calls")
	}
	lockAfter, err := os.ReadFile(lockPath)
	if err != nil || !bytes.Equal(lockBefore, lockAfter) {
		t.Fatal("locked catalog changed lock bytes")
	}
	for _, path := range []string{".cache", "artifacts"} {
		if _, err := os.Stat(filepath.Join(root, path)); !os.IsNotExist(err) {
			t.Fatalf("unexpected output directory %s: %v", path, err)
		}
	}
	code, _, _ = runForTest([]string{"check", "--locked"}, root)
	if code != ExitSourceDiagnostics {
		t.Fatal("experimental catalog promoted stable source admission")
	}
}

func TestExperimentalAtlasAggregateCatalogFailuresStayAtomic(t *testing.T) {
	root := writeAtlasAggregateWorkspace(t)
	source := atlasAggregateSymbol(t, root, "example.session.app", "Token", "resource")
	args := append(atlasCallsArguments("example.session.app", source.Visible, "outgoing"), "--catalog-version", "2")
	for _, replacement := range []struct{ flag, value string }{{"--max-packages", "1"}, {"--max-calls", "1"}, {"--max-output-bytes", "1"}, {"--catalog-version", "02"}, {"--symbol", "fysym1-sha256-128-" + strings.Repeat("0", 32)}} {
		code, output, stderr := runForTest(append([]string{"experimental", "atlas-calls"}, replaceAtlasPackArgument(args, replacement.flag, replacement.value)...), root)
		if code != ExitInvalidInvocation || output != "" || stderr == "" {
			t.Fatal("invalid request leaked partial output")
		}
	}
	var errorText bytes.Buffer
	if code := Run(append([]string{"experimental", "atlas-calls"}, args...), root, failingWriter{}, &errorText); code != ExitInternalFailure || !strings.Contains(errorText.String(), "write output failed") {
		t.Fatal("writer failure swallowed")
	}
	for _, version := range []string{"0", "1"} {
		code, output, _ := runForTest(append([]string{"experimental", "atlas-calls"}, replaceAtlasPackArgument(args, "--catalog-version", version)...), root)
		if code != ExitSourceDiagnostics || output != "" {
			t.Fatal("legacy calls admitted aggregates")
		}
	}
	writeWorkspaceFile(t, root, "src/main.fy", strings.Replace(atlasAggregateSource, "make<i32>(42)", "make<Unknown>(42)", 1))
	code, output, stderr := runForTest(append([]string{"experimental", "atlas-calls"}, args...), root)
	if code != ExitSourceDiagnostics || output != "" || stderr == "" {
		t.Fatalf("invalid source: %d %s %s", code, output, stderr)
	}
}
