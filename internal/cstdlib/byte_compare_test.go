package cstdlib

import (
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestByteMatchWindowsNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	implementation, err := WindowsMSVCSourceForSymbols([]string{"fy_std_bytes_match_at_v0"})
	if err != nil {
		t.Fatal(err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	const probe = `
#include <stddef.h>
#include <stdint.h>
int main(void) {
    const uint8_t input[] = {0, 1, 0, 255};
    const uint8_t same[] = {1, 0};
    const uint8_t different[] = {1, 1};
    if (fy_std_bytes_match_at_v0(input, 4, 1, same, 2) != UINT32_C(1)) return 1;
    if (fy_std_bytes_match_at_v0(input, 4, 1, different, 2) != UINT32_C(0)) return 2;
    if (fy_std_bytes_match_at_v0(input, 4, 1, different, 1) != UINT32_C(1)) return 3;
    if (fy_std_bytes_match_at_v0(input, 4, 4, NULL, 0) != UINT32_C(1)) return 4;
    if (fy_std_bytes_match_at_v0(input, 4, 5, NULL, 0) != UINT32_C(0)) return 5;
    if (fy_std_bytes_match_at_v0(input, 4, 3, same, 2) != UINT32_C(0)) return 6;
    if (fy_std_bytes_match_at_v0(NULL, 0, 0, NULL, 0) != UINT32_C(1)) return 7;
    if (fy_std_bytes_match_at_v0(NULL, 1, 0, same, 1) != UINT32_C(0)) return 8;
    if (fy_std_bytes_match_at_v0(input, 4, 0, NULL, 1) != UINT32_C(0)) return 9;
    if (input[2] != 0 || same[1] != 0 || different[1] != 1) return 10;
    return 0;
}
`
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, implementation+probe)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
			if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("byte comparison = %+v, %v", outcome, err)
			}
		})
	}
}
