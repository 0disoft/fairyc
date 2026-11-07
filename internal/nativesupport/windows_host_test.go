package nativesupport

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/candidatehost"
	"github.com/0disoft/fairyc/internal/target"
)

func TestWindowsCandidateHostSelection(t *testing.T) {
	for _, test := range []struct {
		name    string
		symbols []string
	}{
		{"process", []string{"fairyc_private_windows_assign_v1"}},
		{"rooted", []string{"fairyc_private_windows_root_file_state_lock_v1"}},
		{"https", []string{"fairyc_private_windows_https_read_v1"}},
		{"console", []string{"fairyc_private_windows_cancel_console_bind_v1", "fairyc_private_windows_cancel_console_close_v1"}},
		{"combined", []string{"fairyc_private_windows_assign_v1", "fairyc_private_windows_root_file_read_v1", "fairyc_private_windows_https_open_v1", "fairyc_private_windows_cancel_local_v1", "fairyc_private_windows_assign_v1"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			host, err := candidatehost.WindowsSource(strings.Join(test.symbols, "(\n") + "(")
			if err != nil || host == "" {
				t.Fatalf("existing closed host selection: %v", err)
			}
			for _, storage := range []bool{false, true} {
				for _, traps := range []bool{false, true} {
					standard := []string{"fy_std_system_heap_allocate_v0"}
					if storage {
						standard = append(standard, "fy_private_vector_initialize_v0", "fy_private_vector_prepare_push_v0")
					}
					options := Options{SDKStorage: storage, TrapReceipt: traps}
					base, err := Source(target.X8664WindowsMSVC, standard, options)
					if err != nil {
						t.Fatal(err)
					}
					// Interleave private host/vector/stdlib spellings. Spare capacity
					// detects accidental mutation of the caller's backing array.
					symbols := append([]string{test.symbols[0]}, standard...)
					symbols = append(symbols, test.symbols[1:]...)
					backing := append(symbols, "caller-owned-spare")
					symbols = backing[:len(backing)-1]
					before := append([]string(nil), backing...)
					options.CandidateHost = true
					got, err := Source(target.X8664WindowsMSVC, symbols, options)
					if err != nil || got != base+"\n"+host {
						t.Fatalf("host must follow unchanged stdlib/vector/trap support (storage=%v traps=%v): %v", storage, traps, err)
					}
					if !reflect.DeepEqual(backing, before) {
						t.Fatal("mutated caller symbol storage")
					}
				}
			}
		})
	}
}

func TestWindowsCandidateHostOptInAndOrdinarySupport(t *testing.T) {
	for _, symbols := range [][]string{nil, {"fy_std_system_heap_allocate_v0"}, {"fy_private_vector_initialize_v0"}} {
		for _, traps := range []bool{false, true} {
			options := Options{SDKStorage: true, TrapReceipt: traps}
			ordinary, err := Source(target.X8664WindowsMSVC, symbols, options)
			if err != nil {
				t.Fatal(err)
			}
			options.CandidateHost = true
			selected, err := Source(target.X8664WindowsMSVC, symbols, options)
			if err != nil || ordinary != selected {
				t.Fatalf("host opt-in changed an ordinary support unit: %v", err)
			}
		}
	}
	for _, options := range []Options{{}, {SDKStorage: true}, {TrapReceipt: true}, {SDKStorage: true, TrapReceipt: true}} {
		if source, err := Source(target.X8664WindowsMSVC, []string{"fairyc_private_windows_assign_v1"}, options); err == nil || source != "" {
			t.Fatalf("host adapter linked without explicit CandidateHost: %v", err)
		}
	}
	if source, err := Source(target.X8664WindowsMSVC, []string{"fairyc_private_windows_assign_v1", "fy_private_vector_initialize_v0"}, Options{CandidateHost: true}); err == nil || source != "" {
		t.Fatal("host opt-in must not implicitly enable private vector storage")
	}
}

func TestWindowsCandidateHostRejectsUnclosedSymbols(t *testing.T) {
	for _, symbol := range []string{
		"fairyc_private_windows_", "fairyc_private_windows",
		"fairyc_private_windows_unknown_v1", "fairyc_private_windows_cancel_reset_v1",
		"fairyc_private_windows_assign_v2", "fairyc_private_windows_assign_v1_extra",
		"fairyc_private_windows_root_file_unknown_v1", "fairyc_private_windows_https_unknown_v1",
		"fairyc_private_windows_assign_v1(", "fairyc_private_windows_assign_v1()",
		"fairyc_private_windows_assign_v1 ", " fairyc_private_windows_assign_v1",
		"fairyc_private_windows_assign_v1\n", "fairyc_private_windows_assign_v1\x00",
		"fairyc_private_windows_assign-v1", "fairyc_private_windows_assign_v1;",
		"fairyc_private_windows_assign_v1/* comment */", "fairyc_private_windows_assign_v1\t(",
		"fairyc_private_windows_한글_v1", "fairyc_private_windows_assign_v1\nfairyc_private_windows_unknown_v1",
		"fairyc_private_linux_root_file_read_v1", "not_a_closed_standard_symbol",
		"fy_private_vector_unknown_v0",
	} {
		t.Run(symbol, func(t *testing.T) {
			// A valid host before or after the invalid spelling cannot hide it.
			for _, symbols := range [][]string{{symbol}, {"fairyc_private_windows_assign_v1", symbol}, {symbol, "fairyc_private_windows_assign_v1"}} {
				for _, storage := range []bool{false, true} {
					source, err := Source(target.X8664WindowsMSVC, symbols, Options{CandidateHost: true, SDKStorage: storage})
					if err == nil || source != "" {
						t.Fatalf("unclosed spelling admitted (storage=%v): source bytes=%d err=%v", storage, len(source), err)
					}
				}
			}
		})
	}
}
