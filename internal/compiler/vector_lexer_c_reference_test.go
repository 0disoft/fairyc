package compiler_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

// This C source is independently authored and rejects floats and strings
// until their cooked values can match the full scanner.
func TestPrivateVectorSDKCandidateIndependentCScannerCore(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	code, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "scanner_reference.c"))
	if err != nil {
		t.Fatal(err)
	}
	inputs := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"identifier and keyword", []byte("module fairy; true false null defer")},
		{"BOM CRLF and nested comment", append([]byte{0xef, 0xbb, 0xbf}, []byte("module main;\r\n/// doc\r\n/* outer /* inner */ done */\r\n")...)},
		{"symbols and whitespace", []byte("a <<= b::c -> d..e += f\t\n")},
		{"unclosed comment", []byte("let x; /* still open")},
		{"invalid bytes", []byte{'m', ' ', 0xff, ' ', 0xe3, 0x95, 0x9c, ' ', '#', 0}},
		{"lone CR", []byte("abc \rnext")},
		{"real scanner sample", []byte("let n=12345; /*open")},
		{"integer spellings", []byte("0 000_42 0xDEAD_BEEF 0b1010_0011 0xffffffffffffffffffffffffffffffff")},
		{"invalid integers", []byte("0x 0b 1__2 12abc 0xG 0b102")},
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, string(code))
			if err != nil {
				t.Fatal(err)
			}
			for _, item := range inputs {
				t.Run(item.name, func(t *testing.T) {
					outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(artifact, scannerCInput(item.data))
					if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("C scanner execution: %+v %v", outcome, err)
					}
					want := scannerGoObservation(item.data)
					if outcome.Stdout != want {
						t.Fatalf("C scanner differs from Go lexer\n got: %q\nwant: %q", outcome.Stdout, want)
					}
				})
			}
			for _, unsupported := range []string{"42.5", "1e10", `"text"`, `b"bytes"`, `c"text"`} {
				outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(artifact, scannerCInput([]byte(unsupported)))
				if err != nil || outcome.ExitCode != 7 || outcome.Stdout != "" || outcome.Stderr != "" {
					t.Fatalf("unsupported %q: %+v %v", unsupported, outcome, err)
				}
			}
		})
	}
}

func scannerCInput(data []byte) *bytes.Reader {
	encoded := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint32(encoded, uint32(len(data)))
	copy(encoded[4:], data)
	return bytes.NewReader(encoded)
}

func scannerGoObservation(data []byte) string {
	result := lexer.Lex(source.NewFile("input.fy", data), source.NewBudget(source.DefaultLimits(), len(data)))
	var output strings.Builder
	for _, token := range result.Tokens {
		cooked := hex.EncodeToString([]byte(token.Cooked))
		if cooked == "" {
			cooked = "-"
		}
		fmt.Fprintf(&output, "T %d %d %d %d %s\n", token.Kind, token.Trivia,
			token.Span.StartByte, token.Span.EndByte, cooked)
	}
	for _, diagnostic := range result.Diagnostics {
		text := hex.EncodeToString([]byte(diagnostic.Token))
		if text == "" {
			text = "-"
		}
		fmt.Fprintf(&output, "D %s %d %d %s %s\n", diagnostic.Code,
			diagnostic.Span.StartByte, diagnostic.Span.EndByte, diagnostic.Repair, text)
	}
	return output.String()
}
