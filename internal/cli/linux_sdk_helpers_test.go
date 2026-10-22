package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
)

// The callers already checked independent oracle assertions through the
// interpreter and MIR codec. Execute that same program under both Linux compilers.
func runLinuxSDKHelperExecution(t *testing.T, unit cbackend.Unit) {
	t.Helper()
	support, err := nativesupport.Source(target.X8664LinuxGNU, unit.ForeignSymbols, nativesupport.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				toolchain, err := cdriver.Resolve(target.X8664LinuxGNU, compiler)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(toolchain, profile, unit.Source, support)
				if err != nil {
					t.Fatal(err)
				}
				root := t.TempDir()
				image := filepath.Join(root, "sdk-helper")
				if err := os.WriteFile(image, artifact.Bytes, 0o700); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, image)
				command.Dir = root
				if output, err := command.CombinedOutput(); err != nil || len(output) != 0 {
					t.Fatalf("Linux SDK helper: %v output=%q", err, output)
				}
			})
		}
	}
}
