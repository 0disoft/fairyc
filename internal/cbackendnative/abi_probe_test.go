package cbackendnative

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cabi"
	"github.com/0disoft/fairyc/internal/cabifixture"
	"github.com/0disoft/fairyc/internal/target"
)

func TestGenerateABIProbesFromRepositoryFixtures(t *testing.T) {
	t.Parallel()
	manifest := filepath.Join("..", "..", "spec", "fixtures", "c-abi", "manifest-v0.json")
	targetID, cases, err := cabifixture.NativeProbeCases(manifest)
	if err != nil {
		t.Fatal(err)
	}
	probes, err := generateABIProbes(targetID, cases)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := generateABIProbes(targetID, cases)
	if err != nil || !reflect.DeepEqual(repeated, probes) {
		t.Fatalf("repeated ABI probe drifted: %v", err)
	}
	if targetID != target.X8664LinuxGNU || len(probes) != 14 || probes[0].ID != "pass.aggregate-rollback" || probes[len(probes)-1].ID != "pass.void-return" {
		t.Fatalf("ABI probes = %q, %+v", targetID, probes)
	}
	for _, probe := range probes {
		for role, source := range map[string]string{"caller": probe.CallerSource, "callee": probe.CalleeSource} {
			if source == "" || strings.Contains(source, "\r") || strings.Contains(source, probe.ID) || strings.Contains(source, filepath.Dir(manifest)) {
				t.Errorf("%s %s source is empty, non-LF, or leaks fixture identity", probe.ID, role)
			}
			for _, required := range []string{"fy_abi_target", "fy_abi_failure", "fy_abi_callback"} {
				if !strings.Contains(source, required) {
					t.Errorf("%s %s source is missing %q", probe.ID, role, required)
				}
			}
			if strings.Contains(source, "typedef struct") && !strings.Contains(source, "struct fy_layout_tag_") {
				t.Errorf("%s %s source has an anonymous cross-translation-unit struct", probe.ID, role)
			}
		}
		if !strings.Contains(probe.CallerSource, "int main(void)") || strings.Contains(probe.CalleeSource, "int main(void)") || probe.CallerSource == probe.CalleeSource {
			t.Errorf("%s caller/callee ownership drifted", probe.ID)
		}
	}
	cases[0].Plan.Arguments[0].Classification.Classes[0] = cabi.ClassMemory
	if _, err := generateABIProbes(targetID, cases); err == nil || !strings.Contains(err.Error(), "drifted from its fixture plan") {
		t.Fatalf("generateABIProbes() accepted plan drift: %v", err)
	}
}
