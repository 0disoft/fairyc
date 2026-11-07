package candidatehost

import (
	"fmt"
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

func TestPrivateWindowsCancelSymbolSelection(t *testing.T) {
	for _, operation := range []string{"create", "open", "signal", "local", "console_bind", "console_close"} {
		name := "fairyc_private_windows_cancel_" + operation + "_v1"
		for _, gap := range []string{"", " ", "\t", "\r\n"} {
			t.Run(operation+fmt.Sprintf("/gap-%q", gap), func(t *testing.T) {
				source, err := WindowsSource("extern void *" + name + gap + "(void);")
				if err != nil || source != windowsProcess {
					t.Fatalf("cancel adapter %s: error=%v, selected process=%v", name, err, source == windowsProcess)
				}
			})
		}
		// Cancellation composes with existing optional hosts without selecting
		// either unrelated component when used alone.
		source, err := WindowsSource(name + "( fairyc_private_windows_root_file_close_v1( fairyc_private_windows_https_read_v1(")
		if err != nil || source != windowsProcess+"\n"+windowsRootedFile+"\n"+windowsHTTPS {
			t.Fatalf("combined cancel adapter %s: %v", name, err)
		}
	}
	for _, name := range []string{
		"fairyc_private_windows_cancel_v1", "fairyc_private_windows_cancel_reset_v1",
		"fairyc_private_windows_cancel_create_v2", "fairyc_private_windows_cancel_open_v1_extra",
		"fairyc_private_windows_cancel_signal_v10",
		"fairyc_private_windows_cancel_local_v2", "fairyc_private_windows_cancel_console_bind_v1_extra",
		"fairyc_private_windows_cancel_console_close_v10",
	} {
		t.Run(name, func(t *testing.T) {
			source, err := WindowsSource("fairyc_private_windows_cancel_create_v1( " + name + "(")
			if err == nil || source != "" || !strings.Contains(err.Error(), name) {
				t.Fatalf("unknown cancellation operation did not fail closed: source bytes=%d error=%v", len(source), err)
			}
		})
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
