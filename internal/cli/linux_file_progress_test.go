package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
)

func linuxFileProgressUnit(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.linux-file-progress\"\nname = \"file-progress\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
	writeTestSource(t, root, "src/main.fy", linuxFileProgressSource)
	var diagnostics strings.Builder
	lowered, code := lowerWorkspaceExecutable(root, false, workspaceCompilePolicy{stableDiagnostics: true}, "Linux File progress", &diagnostics)
	if code != ExitSuccess {
		t.Fatalf("lower: %d %s", code, diagnostics.String())
	}
	unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, cbackend.Stage0Profile())
	if err != nil {
		t.Fatal(err)
	}
	var symbols []string
	for _, symbol := range unit.ForeignSymbols {
		if symbol != "fairyc_file_progress_begin" && symbol != "fairyc_file_progress_end" {
			symbols = append(symbols, symbol)
		}
	}
	support, err := linuxVectorSupportSource(symbols)
	if err != nil {
		t.Fatal(err)
	}
	// Intercept only OS calls in the actual target-selected support implementation.
	hooked := strings.NewReplacer("read(", "fy_file_progress_read(", "write(", "fy_file_progress_write(",
		"fsync(", "fy_file_progress_sync(", "close(", "fy_file_progress_close(").Replace(support)
	return unit.Source, linuxFileProgressHooks + hooked
}

func TestLinuxFileProgressEmission(t *testing.T) {
	source, support := linuxFileProgressUnit(t)
	for _, symbol := range []string{"fy_std_file_write_v0", "fy_std_file_write_all_v0", "fy_std_file_read_v0",
		"fy_std_file_read_exact_v0", "fy_std_file_sync_v0", "fy_std_file_close_v0"} {
		if !strings.Contains(source, symbol) || !strings.Contains(support, symbol) {
			t.Errorf("missing SDK call/support: %s", symbol)
		}
	}
}

func TestLinuxFileProgressNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires Linux GNU amd64, GCC and Clang; emission has a separate host-independent test")
	}
	source, support := linuxFileProgressUnit(t)
	for _, cc := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(cc+"/"+profile.Name, func(t *testing.T) {
				tc, err := cdriver.Resolve(target.X8664LinuxGNU, cc)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(tc, profile, source, support)
				if err != nil {
					t.Fatal(err)
				}
				root := t.TempDir()
				image := filepath.Join(root, "file-progress")
				if err := os.WriteFile(image, artifact.Bytes, 0o700); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, image)
				cmd.Dir = root
				if out, err := cmd.CombinedOutput(); err != nil || len(out) != 0 {
					t.Fatalf("File progress: %v %q", err, out)
				}
			})
		}
	}
}
