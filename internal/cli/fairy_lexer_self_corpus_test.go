package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

func streamObservation(t *testing.T, input []byte, previous bool) []uint64 {
	t.Helper()
	file := source.NewFile("input.fy", input)
	budget := source.NewBudget(source.DefaultLimits(), len(input))
	result := lexer.Lex(file, budget)
	if previous {
		result = lexer.LexPreviousRevision(file, source.NewBudget(source.DefaultLimits(), len(input)))
	}
	values := []uint64{uint64(len(result.Tokens))}
	text := func(value string) {
		values = append(values, uint64(len(value)))
		for _, b := range []byte(value) {
			values = append(values, uint64(b))
		}
	}
	for _, token := range result.Tokens {
		values = append(values, uint64(token.Kind), uint64(token.Trivia), uint64(token.Span.StartByte), uint64(token.Span.EndByte))
		text(token.Cooked)
	}
	values = append(values, uint64(len(result.Diagnostics)))
	for _, d := range result.Diagnostics {
		code, err := strconv.ParseUint(strings.TrimPrefix(d.Code, "E"), 10, 32)
		if err != nil {
			t.Fatal(err)
		}
		repair := uint64(0)
		if string(d.Repair) == "exit" {
			repair = 1
		}
		values = append(values, code, uint64(d.Span.StartByte), uint64(d.Span.EndByte), repair)
		text(d.Token)
	}
	return values
}

func streamPageFrame(input []byte, previous bool, start uint32) []byte {
	frame := make([]byte, 9, 9+len(input))
	if previous {
		frame[0] = 1
	}
	binary.LittleEndian.PutUint32(frame[1:5], uint32(len(input)))
	binary.LittleEndian.PutUint32(frame[5:9], start)
	return append(frame, input...)
}

func TestWindowsFairyLexerStreamSelfCorpus(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	entries, err := os.ReadDir(filepath.Join(example, "src"))
	if err != nil {
		t.Fatal(err)
	}
	type inputCase struct {
		name string
		data []byte
	}
	var inputs []inputCase
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			t.Fatal("unexpected source symlink")
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".fy") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(example, "src", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if len(data) > 32768 {
			t.Fatalf("source exceeds candidate admission: %s", entry.Name())
		}
		inputs = append(inputs, inputCase{entry.Name(), data})
	}
	sourceCount := len(inputs)
	if sourceCount == 0 {
		t.Fatal("empty own-source corpus")
	}
	inputs = append(inputs, inputCase{"identifier-32767", bytes.Repeat([]byte("a"), 32767)}, inputCase{"identifier-32768", bytes.Repeat([]byte("a"), 32768)}, inputCase{"long-cooked", []byte(`"` + strings.Repeat("a", 10000) + `"`)}, inputCase{"unterminated", []byte("/*" + strings.Repeat("a", 20000))})
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	pages, err := os.ReadFile(filepath.Join(example, "src", "pages.fy"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/pages.fy", string(pages))
	writeTestSource(t, root, "src/main.fy", "module main; import pages as pages; @effects(console) fn i32 main() { return pages::run(); }")
	pageCount := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		var stderr strings.Builder
		compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &stderr)
		if code != ExitSuccess {
			t.Fatalf("compile: %d %s", code, stderr.String())
		}
		for _, input := range inputs {
			for _, previous := range []bool{false, true} {
				values := streamObservation(t, input.data, previous)
				for start := 0; start < len(values); start += 4096 {
					var expected strings.Builder
					for _, value := range values[start:min(start+4096, len(values))] {
						fmt.Fprintf(&expected, "%d ", value)
					}
					if expected.Len() >= 60000 {
						t.Fatal("page exceeds bounded transport")
					}
					outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewReader(streamPageFrame(input.data, previous, uint32(start))))
					if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" || outcome.Stdout != expected.String() {
						t.Fatalf("%s previous=%v profile=%v start=%d: exit=%d err=%v stderr=%s output lengths %d/%d", input.name, previous, profile, start, outcome.ExitCode, err, outcome.Stderr, len(outcome.Stdout), expected.Len())
					}
					pageCount++
				}
			}
		}
		valid := streamPageFrame([]byte("a"), false, 0)
		for length := 0; length < len(valid); length++ {
			outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewReader(valid[:length]))
			if err != nil || outcome.ExitCode != 8 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("truncated frame %d: %+v %v", length, outcome, err)
			}
		}
		oversized := streamPageFrame(nil, false, 0)
		binary.LittleEndian.PutUint32(oversized[1:5], 32769)
		for _, bad := range []struct {
			data []byte
			exit int32
		}{{oversized, 4}, {append(valid, 0), 8}} {
			outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewReader(bad.data))
			if err != nil || outcome.ExitCode != bad.exit || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("bad frame: %+v %v", outcome, err)
			}
		}
	}
	t.Logf("compared all %d own source files plus 4 boundary inputs in both revisions/profiles across %d bounded pages", sourceCount, pageCount)
}
