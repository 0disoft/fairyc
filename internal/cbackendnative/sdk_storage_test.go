package cbackendnative

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestSDKStorageProbe(t *testing.T) {
	// Linux lowering and exact dependency selection are checked on every host.
	source, support, err := sdkStorageProbe(t.TempDir(), cbackend.Stage0Profile())
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range []string{"fy_std_region_allocate_zeroed_v0", "fy_std_region_reset_v0",
		"fy_std_buffer_reserve_v0", "fy_std_string_append_storage_v0", "fy_std_cstring_from_text_storage_v0"} {
		if !strings.Contains(source, symbol) || !strings.Contains(support, symbol) {
			t.Errorf("SDK storage call/support missing %s", symbol)
		}
	}
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		var toolchains []cdriver.Toolchain
		for _, name := range []string{"clang", "gcc"} {
			tc, err := cdriver.Resolve(cbackend.Stage0Profile().Target, name)
			if err != nil {
				t.Fatal(err)
			}
			toolchains = append(toolchains, tc)
		}
		count, err := runStandardLibraryProbe(t.TempDir(), toolchains, "public SDK storage", "sdk-storage", source, support)
		if err != nil || count != 4 {
			t.Fatalf("Linux public SDK storage: executions=%d error=%v", count, err)
		}
		return
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Linux emission checked; native execution requires Linux or Windows amd64")
	}
	source, support, err = sdkStorageProbe(t.TempDir(), cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range nativeProfiles {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCTarget(tc, profile, source, support)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
			if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("SDK storage: %+v error=%v", outcome, err)
			}
		})
	}
}
