package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

func tokenStatsOutput(tokens, ids, numbers int, sum string, first, firstIDs, firstNumbers int) string {
	return fmt.Sprintf("tokens\t%d\nidentifiers\t%d\nnumbers\t%d\nsum\t%s\nfirst_tokens\t%d\nfirst_identifiers\t%d\nfirst_numbers\t%d\n", tokens, ids, numbers, sum, first, firstIDs, firstNumbers)
}

func TestStructTokenAnalyzerNative(t *testing.T) {
	if runtime.GOARCH != "amd64" || runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		t.Skip("requires Windows MSVC or Linux GCC/Clang amd64")
	}
	root := t.TempDir()
	manifest, err := os.ReadFile(filepath.Join("..", "..", "examples", "struct_vec_view", "fairy.toml"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "fairy.toml", string(manifest))
	for _, name := range []string{"main.fy", "tokens.fy"} {
		src, err := os.ReadFile(filepath.Join("..", "..", "examples", "struct_vec_view", "src", name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, filepath.Join("src", name), string(src))
	}
	evidenceRoot := filepath.Join("..", "..", ".cache", "struct-token-native", runtime.GOOS)
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		args   []string
		exit   int
		output string
	}{
		{"default", nil, 0, tokenStatsOutput(10, 3, 2, "4000000007", 6, 2, 1)},
		{"empty", []string{""}, 0, tokenStatsOutput(0, 0, 0, "0", 0, 0, 0)},
		{"single-statement", []string{"x=7;"}, 0, tokenStatsOutput(4, 1, 1, "7", 4, 1, 1)},
		{"no-semicolon", []string{"alpha+2"}, 0, tokenStatsOutput(3, 1, 1, "2", 3, 1, 1)},
		{"i64-max", []string{"9223372036854775807;"}, 0, tokenStatsOutput(2, 0, 1, "9223372036854775807", 2, 0, 1)},
		{"invalid-byte", []string{"@"}, 2, ""},
		{"non-ascii", []string{"한글"}, 2, ""},
		{"number-overflow", []string{"9223372036854775808"}, 2, ""},
		{"sum-overflow", []string{"9223372036854775807 1"}, 2, ""},
		{"token-limit", []string{strings.Repeat("+", 129)}, 2, ""},
		{"argument-limit", []string{strings.Repeat("x", 4097)}, 2, ""},
		{"usage", []string{"x", "y"}, 2, ""},
	}
	var rows []map[string]any
	for _, release := range []bool{false, true} {
		args := []string{"build"}
		profile := "dev"
		if release {
			args = append(args, "--release", "--locked")
			profile = "release"
		}
		code, out, diag := runForTest(args, root)
		if code != 0 {
			t.Fatalf("build %s: %d %s %s", profile, code, out, diag)
		}
		name := "struct-vec-view"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		image := filepath.Join(root, "build", name)
		data, err := os.ReadFile(image)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:])
		if err := os.WriteFile(filepath.Join(evidenceRoot, profile+filepath.Ext(image)), data, 0700); err != nil {
			t.Fatal(err)
		}
		for _, item := range cases {
			t.Run(profile+"/"+item.name, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, image, item.args...)
				cmd.Dir = root
				var stdout, stderr strings.Builder
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				err := cmd.Run()
				exit := 0
				if err != nil {
					if e, ok := err.(*exec.ExitError); ok {
						exit = e.ExitCode()
					} else {
						t.Fatal(err)
					}
				}
				wantDiag := ""
				if item.exit == 2 {
					wantDiag = "invalid ASCII input, integer overflow, token limit or allocation failure\n"
					if item.name == "usage" {
						wantDiag = "usage: struct-vec-view [ASCII-source]\n"
					}
				}
				if exit != item.exit || stdout.String() != item.output || stderr.String() != wantDiag {
					t.Fatalf("outcome: %d %q %q", exit, stdout.String(), stderr.String())
				}
				rows = append(rows, map[string]any{"case": item.name, "profile": profile, "argv": append([]string{image}, item.args...), "cwd": root, "stdout": stdout.String(), "stderr": stderr.String(), "exit": exit, "imageSHA256": hash})
			})
		}
	}
	if !t.Failed() {
		data, err := json.MarshalIndent(map[string]any{"schema": "fairyc.struct-token-native/v1", "sdk": sdkstdlib.Version, "cases": rows}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(evidenceRoot, "receipt.json"), append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
