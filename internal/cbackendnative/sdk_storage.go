package cbackendnative

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// sdkStorageProbe loads the real bundled SDK through the workspace compiler.
// Its C wrapper observes final cleanup; it does not replace SDK operations.
func sdkStorageProbe(root string, profile cbackend.Profile) (string, string, error) {
	manifest := fmt.Sprintf(`schema = "fairyc.manifest/v1"
[package]
uid = "test.native.sdk-storage"
name = "sdk-storage"
version = "1.0.0"
language = %q
source = "src"
`, sdkstdlib.LanguageRevision)
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o700); err != nil {
		return "", "", err
	}
	for name, contents := range map[string]string{"fairy.toml": manifest, "src/main.fy": sdkStorageSource} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(contents), 0o600); err != nil {
			return "", "", err
		}
	}
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		return "", "", err
	}
	checked, err := workspacecompiler.Check(workspace)
	if err != nil {
		return "", "", err
	}
	if !checked.Verified || len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 {
		return "", "", fmt.Errorf("SDK storage source: diagnostics=%+v unsupported=%+v", checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	owner, err := workspace.RootPackageID()
	if err != nil {
		return "", "", err
	}
	unit, err := cbackend.Emit(checked.MIR, hir.QualifiedModulePath(owner, "main")+"::main", profile)
	if err != nil {
		return "", "", err
	}
	var symbols []string
	for _, symbol := range unit.ForeignSymbols {
		if symbol != "fairyc_sdk_probe_fail_after" {
			symbols = append(symbols, symbol)
		}
	}
	support, err := nativesupport.Source(profile.Target, symbols, nativesupport.Options{})
	if err != nil {
		return "", "", err
	}
	// Extract the existing probe-only allocator adapter, then prepend it to the
	// normal target-selected dependency closure. Product linkage stays untouched.
	plain, err := cstdlib.OwnedTextSource()
	if err != nil {
		return "", "", err
	}
	probe, err := cstdlib.OwnedTextProbeImplementationSource()
	if err != nil {
		return "", "", err
	}
	if !strings.HasSuffix(probe, plain) || probe == plain {
		return "", "", fmt.Errorf("SDK storage probe allocator boundary changed")
	}
	support = strings.TrimSuffix(probe, plain) + support
	wrapped := "#define main fy_sdk_storage_main\n" + unit.Source + `
#undef main
void fy_std_system_heap_probe_fail_after_allocations_v0(size_t count);
size_t fy_std_system_heap_probe_live_allocations_v0(void);
void fairyc_sdk_probe_fail_after(uintptr_t count) {
    fy_std_system_heap_probe_fail_after_allocations_v0((size_t)count);
}
int main(void) {
    int result = fy_sdk_storage_main();
    if (result != 42) return 10;
    if (fy_std_system_heap_probe_live_allocations_v0() != 0) return 11;
    return 0;
}
`
	return wrapped, support, nil
}

func runSDKStorageProbe(root string, toolchains []cdriver.Toolchain) (int, error) {
	source, support, err := sdkStorageProbe(filepath.Join(root, "sdk-storage"), cbackend.Stage0Profile())
	if err != nil {
		return 0, fmt.Errorf("prepare public SDK storage probe: %w", err)
	}
	return runStandardLibraryProbe(root, toolchains, "public SDK storage", "sdk-storage-probe", source, support)
}
