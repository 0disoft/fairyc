package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestInteropGNUEmission(t *testing.T) {
	t.Chdir("../../..")
	if _, err := emitInterop(filepath.Join("examples", "c_interop_callback"), cbackend.Stage0Profile(), false); err != nil {
		t.Fatal(err)
	}
}

func TestInteropCPP(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires native Windows amd64 with MSVC")
	}
	t.Chdir("../../..")
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			args := []string{"--cpp", "--profile", profile.Name}
			if err := run(append(args, "--check-header")); err != nil {
				t.Fatal(err)
			}
			if err := run(args); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInteropGNU(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires native Linux amd64")
	}
	t.Chdir("../../..")
	for _, cc := range []string{"gcc", "clang"} {
		t.Run(cc, func(t *testing.T) {
			if _, err := exec.LookPath(cc); err != nil {
				t.Skipf("compiler unavailable: %v", err)
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					if err := run([]string{"--cc", cc, "--profile", profile.Name}); err != nil {
						t.Fatal(err)
					}
				})
			}
		})
	}
}

func TestInteropHeaderCheck(t *testing.T) {
	if runtime.GOARCH != "amd64" || (runtime.GOOS != "windows" && runtime.GOOS != "linux") {
		t.Skip("requires Windows or Linux amd64")
	}
	example := filepath.Join("..", "..", "..", "examples", "c_interop_callback")
	main, err := os.ReadFile(filepath.Join(example, "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	header, err := os.ReadFile(filepath.Join(example, "legacy.h"))
	if err != nil {
		t.Fatal(err)
	}
	const prototype = "typedef int32_t (*legacy_predicate)(int32_t value, void *data);"
	if !strings.Contains(string(header), prototype) {
		t.Fatal("callback fixture prototype changed")
	}
	cases := []struct {
		name, header, symbol string
	}{
		{name: "match", header: string(header)},
		{name: "callback-argument-width", header: strings.Replace(string(header), prototype, "typedef int32_t (*legacy_predicate)(int64_t value, void *data);", 1), symbol: "legacy_filter"},
		{name: "callback-return-type", header: strings.Replace(string(header), prototype, "typedef int64_t (*legacy_predicate)(int32_t value, void *data);", 1), symbol: "legacy_filter"},
		{name: "missing-declaration", header: strings.Replace(string(header), "int32_t legacy_note_visit(void *data);", "", 1), symbol: "legacy_note_visit"},
	}
	compilers := []string{"gcc", "clang"}
	if runtime.GOOS == "windows" {
		compilers = []string{"msvc"}
	}
	for _, cc := range compilers {
		t.Run(cc, func(t *testing.T) {
			ccArgs := []string{}
			if cc != "msvc" {
				if _, err := exec.LookPath(cc); err != nil {
					t.Skipf("compiler unavailable: %v", err)
				}
				ccArgs = []string{"--cc", cc}
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				for _, tc := range cases {
					t.Run(profile.Name+"/"+tc.name, func(t *testing.T) {
						root := t.TempDir()
						dir := filepath.Join(root, "examples", "c_interop_callback")
						if err := os.MkdirAll(dir, 0o700); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(filepath.Join(dir, "main.fy"), main, 0o600); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(filepath.Join(dir, "legacy.h"), []byte(tc.header), 0o600); err != nil {
							t.Fatal(err)
						}
						// There is no legacy.c in this fixture. Matching declarations
						// can succeed only without compiling the companion or linking.
						t.Chdir(root)
						err := run(append([]string{"--check-header", "--profile", profile.Name}, ccArgs...))
						if tc.symbol == "" {
							if err != nil {
								t.Fatal(err)
							}
							return
						}
						diagnostic := tc.symbol
						if cc == "msvc" && tc.symbol == "legacy_filter" {
							// MSVC reports the incompatible formal parameter without naming the function.
							diagnostic = "C4028"
						}
						if err == nil || !strings.Contains(err.Error(), "failed during compilation") || !strings.Contains(err.Error(), diagnostic) {
							t.Fatalf("wanted compiler rejection %q, got %v", diagnostic, err)
						}
					})
				}
			}
		})
	}
}
