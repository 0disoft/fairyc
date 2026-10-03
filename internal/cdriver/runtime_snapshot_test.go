package cdriver

import (
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
)

func TestLinuxRuntimeSnapshotFlags(t *testing.T) {
	for _, profile := range []Profile{LowProfile, OptimizedProfile} {
		for _, name := range []string{"gcc", "clang"} {
			base := compilerArguments(profile, nil)
			original := slices.Clone(base)
			snapshot := linuxRuntimeSnapshotFlags(name, base)
			if !slices.Equal(base, original) || !slices.Equal(snapshot[:len(base)], original) {
				t.Fatal("snapshot policy changed strict source flags")
			}
			wantExtra := 2
			if !slices.Equal(snapshot[len(base):len(base)+2], []string{"-x", "cpp-output"}) {
				t.Fatal("snapshot input must be explicitly preprocessed C")
			}
			if name == "clang" {
				wantExtra++
				if snapshot[len(base)+2] != "-Wno-error=gnu-line-marker" {
					t.Fatal("marker policy must preserve the diagnostic")
				}
			}
			if len(snapshot) != len(base)+wantExtra {
				t.Fatalf("unexpected snapshot policy: %v", snapshot)
			}
			before := runtimeObjectIdentity{Flags: base}
			after := runtimeObjectIdentity{Flags: snapshot}
			if before.key() == after.key() {
				t.Fatal("changed snapshot policy reused the old cache key")
			}
		}
	}
}

// The positive native cache test already exercises libc markers, cold/warm
// execution and header invalidation. These failures guard the narrow exception:
// it must not admit project warnings or other GNU language extensions.
func TestLinuxRuntimeSnapshotDiagnostics(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires Linux amd64 GCC/Clang")
	}
	for _, name := range []string{"gcc", "clang"} {
		t.Run(name, func(t *testing.T) {
			toolchain, err := Resolve(target.X8664LinuxGNU, name)
			if err != nil {
				t.Skipf("native compiler unavailable: %v", err)
			}
			for _, profile := range []Profile{LowProfile, OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					for _, test := range []struct{ name, unit, support, phase string }{
						{"runtime-warning", "int answer(void); int main(void) { return answer(); }\n", "#include <stdio.h>\nstatic int unused(void) { return 1; }\nint answer(void) { return 7; }\n", "compile runtime:"},
						{"runtime-extension", "int answer(void); int main(void) { return answer(); }\n", "#include <stdio.h>\nint answer(void) { return ({ 7; }); }\n", "compile runtime:"},
						{"unit-warning", "static int unused(void) { return 1; }\nint answer(void); int main(void) { return answer(); }\n", "#include <stdio.h>\nint answer(void) { return 7; }\n", "link runtime:"},
					} {
						t.Run(test.name, func(t *testing.T) {
							root := t.TempDir()
							open := func() (*packagecache.DiskStore, error) { return packagecache.NewDiskStore(root, runtimeObjectLimits()) }
							artifact, err := buildLinuxRuntimeCached(toolchain, profile, test.unit, test.support, open)
							if err == nil || !strings.Contains(err.Error(), test.phase) || len(artifact.Bytes) != 0 {
								t.Fatalf("strict diagnostic lost: artifact=%d bytes, error=%v", len(artifact.Bytes), err)
							}
							// A failed compile/link must not publish a reusable object.
							records, readErr := filepath.Glob(filepath.Join(root, "v1", "*", "*.json"))
							if readErr != nil || len(records) != 0 {
								t.Fatalf("failed build published runtime records: %v, %v", records, readErr)
							}
						})
					}
				})
			}
		})
	}
}
