package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCandidateInlinePacketParity(t *testing.T) {
	if (runtime.GOOS != "windows" && runtime.GOOS != "linux") || runtime.GOARCH != "amd64" {
		t.Skip("requires native Windows/Linux AMD64")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "native_packet_decode")
	fixture := filepath.Join(root, "packets.bin")
	command := exec.CommandContext(t.Context(), "go", "test", "-mod=readonly", "./internal/candidatebuild", "-run", "^TestPacketDecodeOracle$", "-count=1")
	command.Dir = filepath.Join("..", "..")
	command.Env = append(os.Environ(), "FAIRYC_PACKET_DECODE_FIXTURE="+fixture)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("Go packet oracle: %v %s", err, out)
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "native_packet_decode.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(data), "module tests::native_packet_decode;", "module main;", 1)
	program = strings.ReplaceAll(program, "@test ", "") + "\n@effects(filesystem) fn i32 main(){matches_go_admission_and_part_lengths();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	for _, args := range [][]string{{"run", "--", fixture}, {"run", "--release", "--locked", "--", fixture}} {
		code, out, diagnostics := runForTest(args, root)
		if code != ExitSuccess || out != "" || diagnostics != "" {
			t.Fatalf("%v: %d %q %s", args, code, out, diagnostics)
		}
	}
}
