package cli

import (
	"github.com/0disoft/fairyc/internal/cdriver"
	"testing"
	"time"
)

// Dimension probe only: this does not claim authored CLI execution acceptance.
// It identifies which existing private-host transport bound prevents that replay.
func TestCandidateInlineLocalsCLIHostDimensions(t *testing.T) {
	profiles := copiedClosureCLIProfiles(t)
	root := t.TempDir()
	copyLexerFileCLIExample(t, root)
	build, support := candidateParityBuildPartsWithLimit(t, root, map[string]bool{"CloseHandle": true, "CreateFileW": true, "GetFileInformationByHandleEx": true, "GetFinalPathNameByHandleW": true, "GetLastError": true}, 256<<10)
	literals := "#include \"fy_project.h\"\n"
	total := len(build.Header) + len(literals) + len(support)
	maxUnit := 0
	for _, unit := range build.Sources {
		total += len(unit)
		if len(unit) > maxUnit {
			maxUnit = len(unit)
		}
	}
	t.Logf("full authored CLI dimensions: header=%d literals=%d runtime=%d units=%d maxUnit=%d total=%d partLimit=%d", len(build.Header), len(literals), len(support), len(build.Sources), maxUnit, total, 256<<10)
	invalid := false
	for _, bound := range []struct {
		name     string
		got, max int
	}{
		{"header", len(build.Header), 2 << 20}, {"literals", len(literals), 2 << 20}, {"runtime", len(support), 8 << 20}, {"units", len(build.Sources), 4096},
	} {
		if bound.got > bound.max {
			invalid = true
			t.Logf("validator rejection: %s=%d exceeds %d by %d", bound.name, bound.got, bound.max, bound.got-bound.max)
		}
	}
	if !invalid {
		t.Log("top-level transport dimensions fit; no compiler execution attempted by this measurement probe")
		return
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	_, err = cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profiles[0], build.Header, literals, build.Sources, support, time.Minute, 16<<20)
	if err == nil || err.Error() != "invalid bounded project C inputs" {
		t.Fatalf("dimension admission: %v", err)
	}
	t.Log("reproduced strict transport rejection before C compilation; ordinary product limits preserved")
}
