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
