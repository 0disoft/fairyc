package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scip-code/scip/bindings/go/scip"
	"google.golang.org/protobuf/proto"
)

func TestExperimentalSCIPIndexesLockedWorkspace(t *testing.T) {
	const line = "fn i32 main() { let i32 num = 1; /* 🚀 */ return answer::value() + num; }"
	const src = "module main;\r\nimport answer as answer;\r\n" + line + "\r\n"
	root := writeWorkspaceSessionFixture(t, src)
	args := []string{"experimental", "scip"}
	if code, output, _ := runForTest(args, root); code != ExitPackageFailure || output != "" {
		t.Fatalf("missing lock yielded %d, %d bytes", code, len(output))
	}
	if code, _, diagnostics := runForTest([]string{"check"}, root); code != ExitSuccess {
		t.Fatalf("check: %d %s", code, diagnostics)
	}
	lockPath := filepath.Join(root, "fairy.lock")
	lock, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	code, output, diagnostics := runForTest(args, root)
	if code != ExitSuccess || diagnostics != "" {
		t.Fatalf("scip: %d %s", code, diagnostics)
	}
	var index scip.Index
	if err := proto.Unmarshal([]byte(output), &index); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(index.GetMetadata().GetProjectRoot(), "file://") || len(index.Documents) != 2 ||
		index.Documents[0].RelativePath != "libs/answer/src/answer.fy" || index.Documents[1].RelativePath != "src/main.fy" {
		t.Fatalf("unexpected index: %v", &index)
	}
	library, main := index.Documents[0], index.Documents[1]
	if len(library.Symbols) != 1 || main.PositionEncoding != scip.PositionEncoding_UTF8CodeUnitOffsetFromLineStart {
		t.Fatalf("unexpected declarations or encoding: %v", &index)
	}
	value := library.Symbols[0].Symbol
	parsed, err := scip.ParseSymbol(value)
	if err != nil || parsed.Package.Name != "example.session.answer" || parsed.Descriptors[len(parsed.Descriptors)-1].Name != "value" {
		t.Fatalf("unexpected value symbol: %s (%v)", value, err)
	}
	start := strings.Index(line, "answer::value")
	var foundCall, foundLocalDefinition, foundLocalUse bool
	for _, occurrence := range main.Occurrences {
		if occurrence.Symbol == value && occurrence.SymbolRoles == 0 {
			span := occurrence.GetSingleLineRange()
			foundCall = span != nil && span.Line == 2 && span.StartCharacter == int32(start) &&
				span.EndCharacter == int32(start+len("answer::value"))
		}
		if strings.HasPrefix(occurrence.Symbol, "local ") {
			foundLocalDefinition = foundLocalDefinition || occurrence.SymbolRoles == int32(scip.SymbolRole_Definition)
			foundLocalUse = foundLocalUse || occurrence.SymbolRoles == 0
		}
	}
	if !foundCall || !foundLocalDefinition || !foundLocalUse {
		t.Fatalf("missing call or local facts: %v", main)
	}
	if code, again, _ := runForTest(args, root); code != ExitSuccess || again != output {
		t.Fatal("index changed between identical invocations")
	}
	writeWorkspaceFile(t, root, "src/main.fy", src+"\r\n")
	if code, output, _ := runForTest(args, root); code != ExitPackageFailure || output != "" {
		t.Fatalf("stale lock yielded %d, %d bytes", code, len(output))
	}
	if current, err := os.ReadFile(lockPath); err != nil || string(current) != string(lock) {
		t.Fatalf("read-only index changed lock: %v", err)
	}
}
