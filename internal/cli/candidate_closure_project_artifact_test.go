package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func candidateClosureProjectArtifactWorkspace(t *testing.T) (string, []closureProjectCCase) {
	t.Helper()
	cases := candidateClosureProjectCControls(t)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_project_artifact.fy"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_project_artifact.fy", raw), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("project C fixture syntax: %v %+v", err, parsed.Tree.Diagnostics)
	}
	main := strings.Replace(string(raw), "module tests::closure_project_artifact;", "module main;", 1)
	main += fmt.Sprintf("\n@effects(console) fn i32 main(){framing();forged();assert(io::write(\"FYPROJECTC1\\n%d\\n\").is_ok);\n", len(cases))
	for _, tc := range cases {
		texts := [3]string{}
		copy(texts[:], tc.files)
		main += fmt.Sprintf("produce(b%s,b%s,b%s,b%s,%d,%t);\n", strconv.Quote(tc.name), strconv.Quote(texts[0]), strconv.Quote(texts[1]), strconv.Quote(texts[2]), tc.made, tc.sameDeclaration)
	}
	main += "return 0;}\n"
	root := t.TempDir()
	copyLexerExample(t, root)
	writeTestSource(t, root, "src/main.fy", main)
	copyCandidateParityModules(t, root, "module_project_artifact_encode", "module_project_artifact_decode", "module_project_artifact_current", "module_project_artifact_rebase", "project_compiler_core", "module_project_identified_ir", "module_unit_c_emit", "module_unit_c_names", "module_unit_body_ir_closures", "sha256_stream", "syntax", "scanner")
	return root, cases
}

func TestCandidateClosureProjectArtifactSource(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root, cases := candidateClosureProjectArtifactWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("closure project source-bound envelope source: %d %s %s", code, out, diagnostic)
	}
	t.Logf("%d original-source project controls; source checked, not executed", len(cases))
}

func TestCandidateClosureProjectArtifactNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 closure envelope and generated consumers; Linux remains batched")
	}
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	root, cases := candidateClosureProjectArtifactWorkspace(t)
	candidateClosureProjectCNativeFixture(t, root, cases)
}
