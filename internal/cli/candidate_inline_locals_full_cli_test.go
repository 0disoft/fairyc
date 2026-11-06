package cli

import "testing"

// Replay the authored main and persistent cache with real immutable-local bodies.
// The shared runner retains its opt-in/host gates and all packet/source checks.
func TestCandidateInlineLocalsFullCLI(t *testing.T) {
	// This acceptance owns both host profiles, even if a developer's shell has
	// selected one profile for a different inline investigation.
	t.Setenv("FAIRYC_INLINE_CLI_PROFILE", "")
	runCandidateInlineTrapCLI(t, inlineLocalsTraps)
}
