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
	for _, name := range []string{"fairyc_private_windows_root_directory_open_v1", "fairyc_private_windows_root_directory_create_v1", "fairyc_private_windows_root_directory_edit_v1", "fairyc_private_windows_root_file_info_v1", "fairyc_private_windows_root_directory_query_v1", "fairyc_private_windows_root_file_publish_new_v1", "fairyc_private_windows_root_file_time_v1"} {
		source, err := WindowsSource(name + "(")
		if err != nil || source != rooted {
			t.Fatalf("missing inventory adapter for %s: %v", name, err)
		}
	}
}

func TestPrivateWindowsHTTPSSymbolSelection(t *testing.T) {
	for _, name := range []string{"open", "headers", "read", "close"} {
		selected, err := WindowsSource("fairyc_private_windows_https_" + name + "_v1(")
		if err != nil || selected != windowsProcess+"\n"+windowsHTTPS {
			t.Fatalf("HTTPS %s: %v", name, err)
		}
	}
	selected, err := WindowsSource("fairyc_private_windows_https_read_v1( fairyc_private_windows_directory_open_v1(")
	if err != nil || selected != windowsProcess+"\n"+windowsRootedFile+"\n"+windowsHTTPS {
		t.Fatal("combined runtime", err)
	}
	if _, err = WindowsSource("fairyc_private_windows_https_unknown_v1("); err == nil {
		t.Fatal("unknown HTTPS symbol accepted")
	}
}
