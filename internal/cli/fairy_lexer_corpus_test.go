package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

func TestWindowsFairyLexerCorpus(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	base := filepath.Join("..", "..", "spec", "fixtures")
	var paths []string
	err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("unexpected fixture symlink: %s", path)
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".fy") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(paths)
	if len(paths) == 0 {
		t.Fatal("empty source fixture corpus")
	}
	var stdin bytes.Buffer
	var expected strings.Builder
	var names []string
	var ends []int
	var inputEnds []int
	appendObservation := func(name string, input []byte, previous bool, limits source.Limits, custom bool) {
		if len(input) > 4096 || len(names) >= 10240 {
			t.Fatalf("candidate corpus bound exceeded by %s", name)
		}
		mode := byte(0)
		if previous {
			mode = 1
		}
		if custom {
			mode += 2
		}
		stdin.Write([]byte{mode, byte(len(input)), byte(len(input) >> 8)})
		if custom {
			for _, limit := range []int{limits.MaxBytes, limits.MaxTokens, limits.MaxDiagnostics, limits.MaxWorkUnits} {
				if limit < 0 || uint64(limit) > 0xffffffff {
					t.Fatalf("unrepresentable corpus budget: %d", limit)
				}
				var encoded [4]byte
				binary.LittleEndian.PutUint32(encoded[:], uint32(limit))
				stdin.Write(encoded[:])
			}
		}
		stdin.Write(input)
		file := source.NewFile("input.fy", input)
		budget := source.NewBudget(limits, len(input))
		var result lexer.Result
		if previous {
			result = lexer.LexPreviousRevision(file, budget)
		} else {
			result = lexer.Lex(file, budget)
		}
		for _, d := range result.Diagnostics {
			if d.Code == "E0010" && !custom {
				t.Fatalf("reference budget exhaustion in %s", name)
			}
		}
		for _, token := range result.Tokens {
			fmt.Fprintf(&expected, "%d %d %d %d %d ", token.Kind, token.Trivia, token.Span.StartByte, token.Span.EndByte, len(token.Cooked))
			for _, b := range []byte(token.Cooked) {
				fmt.Fprintf(&expected, "%d ", b)
			}
		}
		fmt.Fprintf(&expected, "%d ", len(result.Diagnostics))
		for _, d := range result.Diagnostics {
			for _, value := range []string{d.Code, string(d.Repair), d.Token} {
				fmt.Fprintf(&expected, "%d ", len(value))
				for _, b := range []byte(value) {
					fmt.Fprintf(&expected, "%d ", b)
				}
			}
			fmt.Fprintf(&expected, "%d %d ", d.Span.StartByte, d.Span.EndByte)
		}
		names = append(names, name)
		ends = append(ends, expected.Len())
		inputEnds = append(inputEnds, stdin.Len())
	}
	appendCase := func(name string, input []byte, previous bool) {
		appendObservation(name, input, previous, source.DefaultLimits(), false)
		for _, profile := range []string{"bytes-half", "one-token", "no-diagnostics", "work-half"} {
			limits := source.DefaultLimits()
			switch profile {
			case "bytes-half":
				limits.MaxBytes = len(input) / 2
			case "one-token":
				limits.MaxTokens = 1
			case "no-diagnostics":
				limits.MaxDiagnostics = 0
			case "work-half":
				limits.MaxWorkUnits = len(input) / 2
			}
			appendObservation(name+"/"+profile, input, previous, limits, true)
		}
	}
	for _, path := range paths {
		input, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		appendCase(path, input, false)
		appendCase(path+"/previous", input, true)
		midpoint := len(input) / 2
		inserted := append([]byte(nil), input[:midpoint]...)
		inserted = append(inserted, 0)
		inserted = append(inserted, input[midpoint:]...)
		appendCase(path+"/insert-NUL", inserted, false)
		if len(input) > 0 {
			appendCase(path+"/delete-first", input[1:], false)
			replaced := append([]byte(nil), input...)
			replaced[midpoint] = 255
			appendCase(path+"/replace-invalid", replaced, false)
		}
	}
	appendCase("previous-defer", []byte("defer deferx return true"), true)
	root := t.TempDir()
	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	for _, name := range []string{"fairy.toml", "src/main.fy", "src/scanner.fy", "src/numbers.fy", "src/diagnostics.fy", "src/utf8.fy", "src/strings.fy", "src/budget.fy"} {
		data, err := os.ReadFile(filepath.Join(example, name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, name, string(data))
	}
	for _, release := range []bool{false, true} {
		profile := cdriver.LowProfile
		if release {
			profile = cdriver.OptimizedProfile
		}
		var stderr strings.Builder
		compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: release}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &stderr)
		if code != ExitSuccess {
			t.Fatalf("corpus build release=%v code=%d stderr=%s", release, code, stderr.String())
		}
		begin, inputStart, outputStart := 0, 0, 0
		for begin < len(names) {
			end := begin
			for end < len(names) && end-begin < 2048 && ends[end]-outputStart <= 60000 {
				end++
			}
			if end == begin {
				t.Fatalf("single observation exceeds execution output bound: %s", names[begin])
			}
			inputEnd, outputEnd := inputEnds[end-1], ends[end-1]
			outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewReader(stdin.Bytes()[inputStart:inputEnd]))
			if err != nil {
				t.Fatalf("corpus release=%v starting %s: %v", release, names[begin], err)
			}
			programStderr, category, err := classifyStableRunOutcome(outcome)
			if err != nil || category != "" || programStderr != "" || outcome.ExitCode != 0 {
				t.Fatalf("corpus release=%v starting %s: exit=%d category=%s stderr=%s err=%v", release, names[begin], outcome.ExitCode, category, programStderr, err)
			}
			got, want := outcome.Stdout, expected.String()[outputStart:outputEnd]
			if got != want {
				i := 0
				for i < len(got) && i < len(want) && got[i] == want[i] {
					i++
				}
				caseIndex := begin
				for caseIndex < len(ends) && ends[caseIndex] <= outputStart+i {
					caseIndex++
				}
				name := "after final frame"
				if caseIndex < len(names) {
					name = names[caseIndex]
				}
				t.Fatalf("corpus release=%v %s byte=%d got=%q want=%q", release, name, i, got[i:min(i+200, len(got))], want[i:min(i+200, len(want))])
			}
			begin, inputStart, outputStart = end, inputEnd, outputEnd
		}
		badFrames := []struct {
			input []byte
			exit  int32
		}{{[]byte{4}, 8}, {[]byte{2}, 8}, {[]byte{0}, 8}, {[]byte{0, 1, 0}, 8}, {[]byte{0, 1, 16}, 4}, {[]byte{2, 1, 16}, 4}}
		for length := 0; length <= 16; length++ {
			// Include every truncated budget-header length and a complete header without its body.
			badFrames = append(badFrames, struct {
				input []byte
				exit  int32
			}{append([]byte{3, 1, 0}, make([]byte, length)...), 8})
		}
		for _, bad := range badFrames {
			outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewReader(bad.input))
			if err != nil || outcome.ExitCode != bad.exit || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("malformed frame %x: outcome=%+v err=%v", bad.input, outcome, err)
			}
		}
	}
	t.Logf("compared %d source files and %d framed cases at both profiles", len(paths), len(names))
}
