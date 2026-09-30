package cstdlibcontract

import (
	"github.com/0disoft/fairyc/internal/target"
	"slices"
	"testing"
)

func TestWindowsDirectoryComponentIsOptIn(t *testing.T) {
	symbols := []string{"fy_std_windows_read_dir_open_v0", "fy_std_windows_read_dir_next_v0", "fy_std_windows_read_dir_close_v0", "fy_std_windows_read_dir_drop_v0"}
	for _, symbol := range symbols {
		if !AdmittedForTarget(symbol, target.X8664WindowsMSVC) || AdmittedForTarget(symbol, target.X8664LinuxGNU) {
			t.Fatalf("directory symbol target boundary: %s", symbol)
		}
		if _, err := ComponentsForSymbols([]string{symbol}); err == nil {
			t.Fatal("Linux accepted directory symbol")
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
