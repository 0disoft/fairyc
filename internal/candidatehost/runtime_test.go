package candidatehost

import (
	"strings"
	"testing"
)

func TestPrivateWindowsHostSymbolSelection(t *testing.T) {
	if source, err := WindowsSource("ordinary header"); err != nil || source != "" {
		t.Fatalf("changed ordinary runtime: %v", err)
	}
	if _, err := WindowsSource("fairyc_private_windows_unknown_v1("); err == nil {
		t.Fatal("unknown host symbol admitted")
	}
	source, err := WindowsSource("extern int32_t fairyc_private_windows_create_v1 \n (")
	if err != nil || !strings.Contains(source, "return CreateProcessW(") {
		t.Fatalf("missing ABI adapter: %v", err)
	}
}

func TestPrivateWindowsRootFileSymbolSelection(t *testing.T) {
	ordinary, err := WindowsSource("fairyc_private_windows_current_path_v1(")
	if err != nil || ordinary != windowsProcess {
		t.Fatalf("unused rooted adapters changed ordinary transport: %v", err)
	}
	rooted, err := WindowsSource("fairyc_private_windows_directory_open_v1( fairyc_private_windows_root_file_close_v1(")
	if err != nil || rooted != windowsProcess+"\n"+windowsRootedFile {
		t.Fatalf("missing rooted file adapters: %v", err)
	}
	if _, err := WindowsSource("fairyc_private_windows_root_file_unknown_v1("); err == nil {
		t.Fatal("unknown rooted operation admitted")
	}
}
