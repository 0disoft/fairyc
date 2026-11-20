package main

import (
	"os/exec"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestInteropGNUEmission(t *testing.T) {
	t.Chdir("../../..")
	if _, _, err := interopSources(cbackend.Stage0Profile()); err != nil {
		t.Fatal(err)
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
