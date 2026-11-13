package cstdlibcontract

import (
	"github.com/0disoft/fairyc/internal/target"
	"slices"
	"testing"
)

func TestPortableDirectoryComponentIsOptIn(t *testing.T) {
	symbols := []string{"fy_std_windows_read_dir_open_v0", "fy_std_windows_read_dir_next_v0", "fy_std_windows_read_dir_close_v0", "fy_std_windows_read_dir_drop_v0"}
	for _, symbol := range symbols {
		if !AdmittedForTarget(symbol, target.X8664WindowsMSVC) || !AdmittedForTarget(symbol, target.X8664LinuxGNU) {
			t.Fatalf("directory symbol target boundary: %s", symbol)
		}
		if _, err := ComponentsForSymbols([]string{symbol}); err != nil {
			t.Fatal(err)
		}
	}
	components, err := WindowsMSVCComponentsForSymbols(symbols)
	if err != nil || !slices.Equal(components, []Component{ComponentWindowsDirectory}) {
		t.Fatalf("closure: %v, %v", components, err)
	}
	components, err = WindowsMSVCComponentsForSymbols([]string{"fy_std_file_open_utf8_v0"})
	if err != nil || slices.Contains(components, ComponentWindowsDirectory) {
		t.Fatalf("unrelated File selects directory: %v, %v", components, err)
	}
}

func TestPortableDirectoryPathComponents(t *testing.T) {
	for _, name := range []string{"fy_std_windows_directory_create_v0", "fy_std_windows_directory_ensure_v0", "fy_std_windows_directory_remove_v0", "fy_std_windows_directory_rename_v0", "fy_std_windows_dir_remove_all_v0"} {
		linux, err := ComponentsForSymbols([]string{name})
		if err != nil || !slices.Equal(linux, []Component{ComponentLinuxFile, ComponentLinuxDirectory, ComponentLinuxDirectoryPaths, ComponentTextView}) {
			t.Fatalf("Linux directory path %s: %v %v", name, linux, err)
		}
		windows, err := WindowsMSVCComponentsForSymbols([]string{name})
		if err != nil || !slices.Equal(windows, []Component{ComponentWindowsFile}) {
			t.Fatalf("Windows directory path %s: %v %v", name, windows, err)
		}
	}
}
