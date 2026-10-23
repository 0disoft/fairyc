package workspacecompiler

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/nativesupport"
)

func TestRegionParametersNative(t *testing.T) {
	// Cover all three allocation forms and early Result propagation. Every
	// helper uses the caller's Region; only the caller creates and resets it.
	input := strings.Replace(regionParameterSource, "assert(both(scratch, scratch) == 42);", `
        assert(both(scratch, scratch) == 42);
        assert(switch (copied(scratch)) {
            case .ok(value): { assert(value.len == 2 && value[0] == 79); value[1] = 75; value[1] == 75 }
            case .err(_): { false }
        });
        assert(switch (zeroed(scratch)) {
            case .ok(value): { value.len == 2 && value[0] == 0 && value[1] == 0 }
            case .err(_): { false }
        });
        assert(switch (failed_forward(scratch)) {
            case .ok(_): { false }
            case .err(error): { alloc::error_status(error) == 7 }
        });
        assert(read(scratch, 42) == 42);`, 1) + `
fn Result<ref mut u8[], alloc::AllocError> copied(Region input) { return input.copy_bytes(b"OK"); }
fn Result<ref mut u8[], alloc::AllocError> zeroed(Region input) { return input.alloc_zeroed_bytes(2); }
fn Result<ref i32, alloc::AllocError> failed(Region input) { return .err(alloc::AllocError { .status = 7 }); }
fn Result<ref i32, alloc::AllocError> failed_forward(Region input) {
    let ref i32 value = failed(input)?;
    return .ok(value);
}
`
	root := writeCacheWorkspace(t, regionProviderSource)
	writeFile(t, filepath.Join(root, "src", "main.fy"), regionProviderCaller(input))
	workspace := loadCacheWorkspace(t, root)
	checked, err := Check(workspace)
	if err != nil || !checked.Verified {
		t.Fatalf("check: diagnostics=%+v unsupported=%+v error=%v", checked.Semantic.Diagnostics, checked.Semantic.Unsupported, err)
	}
	owner, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		t.Run(string(profile.Target), func(t *testing.T) {
			unit, err := cbackend.Emit(checked.MIR, hir.QualifiedModulePath(owner, "main")+"::main", profile)
			if err != nil {
				t.Fatal(err)
			}
			support, err := nativesupport.Source(profile.Target, unit.ForeignSymbols, nativesupport.Options{})
			if err != nil {
				t.Fatal(err)
			}
			// Existing allocator instrumentation checks final cleanup without
			// replacing any generated Region operation or adding product hooks.
			plain, err := cstdlib.OwnedTextSource()
			if err != nil {
				t.Fatal(err)
			}
			probe, err := cstdlib.OwnedTextProbeImplementationSource()
			if err != nil || !strings.HasSuffix(probe, plain) || probe == plain {
				t.Fatalf("allocator adapter: %v", err)
			}
			support = strings.TrimSuffix(probe, plain) + support
			wrapped := "#define main region_program_main\n" + unit.Source + `
#undef main
size_t fy_std_system_heap_probe_live_allocations_v0(void);
int main(void) {
    int result = region_program_main();
    if (result != 42 || fy_std_system_heap_probe_live_allocations_v0() != 0) return 1;
    return 42;
}
`
			if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" && profile.Target == cbackend.Stage0WindowsMSVCProfile().Target {
				tc, err := cdriver.ResolveWindowsHostMSVC()
				if err != nil {
					t.Fatal(err)
				}
				for _, optimization := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
					t.Run(optimization.Name, func(t *testing.T) {
						artifact, err := cdriver.BuildWindowsMSVCTarget(tc, optimization, wrapped, support)
						if err != nil {
							t.Fatal(err)
						}
						outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
						if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
							t.Fatalf("Region parameter native: %+v %v", outcome, err)
						}
					})
				}
			} else if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" && profile.Target == cbackend.Stage0Profile().Target {
				for _, compiler := range []string{"clang", "gcc"} {
					tc, err := cdriver.Resolve(profile.Target, compiler)
					if err != nil {
						t.Fatal(err)
					}
					for _, optimization := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
						t.Run(compiler+"/"+optimization.Name, func(t *testing.T) {
							artifact, err := cdriver.Build(tc, optimization, wrapped, support)
							if err != nil {
								t.Fatal(err)
							}
							outcome, err := cdriver.ExecuteLinuxTarget(artifact)
							if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
								t.Fatalf("Region parameter native: %+v %v", outcome, err)
							}
						})
					}
				}
			} else {
				t.Log("emission checked; native execution requires the matching host")
			}
		})
	}
}
