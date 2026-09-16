package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/cdriver"
)

// Observe the stable command path, not the private aggregate adapter. No timing threshold.
func TestStableGenericWorkspaceGrowthMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows amd64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	type sample struct {
		Repetition      int               `json:"repetition"`
		Types           int               `json:"types"`
		Mode            string            `json:"mode"`
		Release         bool              `json:"release"`
		SourceHashes    map[string]string `json:"source_hashes"`
		CheckNS         int64             `json:"check_ns"`
		BuildNS         int64             `json:"build_ns"`
		ExecutableBytes int               `json:"executable_bytes"`
		ExitCode        int               `json:"exit_code"`
	}
	var samples []sample
	for repetition := 0; repetition < 2; repetition++ {
		counts, modes := []int{1, 3, 5}, []string{"generic", "manual"}
		if repetition == 1 {
			counts, modes = []int{5, 3, 1}, []string{"manual", "generic"}
		}
		for _, count := range counts {
			for _, mode := range modes {
				for _, release := range []bool{false, true} {
					root := t.TempDir()
					writeNativeWorkspace(t, root, true)
					provider, main := stableGenericGrowthSources(count, mode == "generic")
					writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", provider)
					writeWorkspaceFile(t, root, "src/main.fy", main)
					hashes := map[string]string{}
					for _, name := range []string{"fairy.toml", "libs/answer/fairy.toml", "src/main.fy", "libs/answer/src/answer.fy"} {
						b, err := os.ReadFile(filepath.Join(root, name))
						if err != nil {
							t.Fatal(err)
						}
						h := sha256.Sum256(b)
						hashes[name] = hex.EncodeToString(h[:])
					}
					start := time.Now()
					code, _, stderr := runForTest([]string{"check"}, root)
					checkNS := time.Since(start).Nanoseconds()
					if code != 0 || stderr != "" {
						t.Fatalf("check %s/%d: %d %s", mode, count, code, stderr)
					}
					args := []string{"build", "--locked"}
					if release {
						args = append(args, "--release")
					}
					start = time.Now()
					code, _, stderr = runForTest(args, root)
					buildNS := time.Since(start).Nanoseconds()
					if code != 0 || stderr != "" {
						t.Fatalf("build: %d %s", code, stderr)
					}
					binary, err := os.ReadFile(filepath.Join(root, "build", "app.exe"))
					if err != nil {
						t.Fatal(err)
					}
					args[0] = "run"
					code, stdout, stderr := runForTest(args, root)
					if code != 42 || stdout != "" || stderr != "" {
						t.Fatalf("run: %d %q %q", code, stdout, stderr)
					}
					samples = append(samples, sample{repetition, count, mode, release, hashes, checkNS, buildNS, len(binary), code})
				}
			}
		}
	}
	report := struct {
		Schema  string   `json:"schema"`
		Build   string   `json:"test_executable_identity"`
		Go      string   `json:"go_version"`
		MSVC    string   `json:"msvc_version"`
		Cache   string   `json:"cache"`
		Samples []sample `json:"samples"`
	}{"fairyc.stable-generic-workspace-growth/v0", identity.String(), runtime.Version(), toolchain.Version, "fresh workspace per sample; in-process CLI; ambient OS/toolchain caches; run rebuilds", samples}
	b, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))
}

func stableGenericGrowthSources(count int, generic bool) (string, string) {
	types, values := []string{"bool", "u8", "i32", "u32", "usize"}, []string{"true", "42", "42", "42", "42"}
	var provider, main strings.Builder
	provider.WriteString("module answer;")
	main.WriteString("module main; import answer as answer; fn i32 main(){")
	if generic {
		provider.WriteString("pub fn T value<T>(T input){return input;}")
	}
	for i := 0; i < count; i++ {
		call := fmt.Sprintf("answer::value<%s>(%s)", types[i], values[i])
		if !generic {
			fmt.Fprintf(&provider, "pub fn %s value%d(%s input){return input;}", types[i], i, types[i])
			call = fmt.Sprintf("answer::value%d(%s)", i, values[i])
		}
		for repeat := 0; repeat < 2; repeat++ {
			fmt.Fprintf(&main, "assert(%s == %s);", call, values[i])
		}
	}
	main.WriteString("return 42;}")
	return provider.String(), main.String()
}
