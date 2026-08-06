package cli

import "testing"

func TestStableNativeTestTimeoutOptionDomain(t *testing.T) {
	t.Parallel()
	options, err := parseStableTestOptions([]string{"--native", "--timeout-seconds", "3", "--", "--timeout-seconds", "7"})
	if err != nil || options.timeoutSeconds != 3 || !options.timeoutExplicit || len(options.programArgs) != 2 || options.programArgs[0] != "--timeout-seconds" {
		t.Fatalf("timeout options = %+v, %v", options, err)
	}
	defaults, err := parseStableTestOptions([]string{"--native"})
	if err != nil || defaults.timeoutSeconds != 60 || defaults.timeoutExplicit {
		t.Fatalf("default timeout options = %+v, %v", defaults, err)
	}
	for _, arguments := range [][]string{
		{"--timeout-seconds", "60"}, {"--list", "--timeout-seconds", "1"},
		{"--native", "--timeout-seconds"}, {"--native", "--timeout-seconds", "0"},
		{"--native", "--timeout-seconds", "01"}, {"--native", "--timeout-seconds", "61"},
		{"--native", "--timeout-seconds", "x"},
		{"--native", "--timeout-seconds", "1", "--timeout-seconds", "2"},
	} {
		if _, err := parseStableTestOptions(arguments); err == nil {
			t.Fatalf("invalid timeout options succeeded: %q", arguments)
		}
	}
}
