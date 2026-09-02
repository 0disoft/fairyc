package cli

import (
	"strings"
	"testing"
)

func TestStableExitCodeContract(t *testing.T) {
	got := []int{
		ExitSuccess,
		ExitInvalidInvocation,
		ExitSourceDiagnostics,
		ExitInternalFailure,
		ExitProgramTrap,
		ExitBuildFailure,
		ExitPackageFailure,
	}
	want := []int{0, 2, 3, 4, 5, 6, 7}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("exit contract[%d] = %d, want %d", index, got[index], want[index])
		}
	}
}

func TestStableHelpInventoryOwnsTopLevelCommands(t *testing.T) {
	for _, command := range []string{
		"fairyc new <package-name> --uid <package-uid>",
		"fairyc check [--locked]",
		"fairyc build [--release] [--locked]",
		"fairyc run [--release] [--locked] [-- <program-args...>]",
		"fairyc test [--json]",
		"fairyc explain <diagnostic-code>",
		"fairyc targets",
		"fairyc experimental parse <file.fy>",
	} {
		if !strings.Contains(usage, command) {
			t.Fatalf("help inventory is missing %q", command)
		}
	}
	if strings.Contains(usage, "fairyc emit c --target") {
		t.Fatal("removed stable emit command reappeared in help inventory")
	}
}

func TestStableConfigDiscoveryRemainsExactRootOnly(t *testing.T) {
	for _, forbidden := range []string{"parent directory", "environment override", "user config", "system config"} {
		if strings.Contains(strings.ToLower(usage), forbidden) {
			t.Fatalf("help unexpectedly advertises configuration discovery %q", forbidden)
		}
	}
}
