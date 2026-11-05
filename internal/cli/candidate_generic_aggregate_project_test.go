package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
)

func TestCandidateOrdinaryProjectTypes(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "module_project_types.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::module_project_types;", "module main;", 1)
	main = strings.ReplaceAll(main, "@test", "") + `
@effects(console) fn i32 main(){
 independently_lowered_units_share_project_types();
 reordered_wrappers_rebind_edges_and_preserve_array_lengths();
 project_type_and_unit_limits_and_signature_prefix_reject();
 return 0;
}`
	candidateAtlasProbeSize(t, "module_project_types", main, 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		code, out := candidateAtlasProcess(t, image, nil)
		if code != 0 || !strings.HasPrefix(out, "independent IR units 2 canonical types ") || strings.Count(out, "\n") != 1 {
			t.Fatalf("ordinary project types: %d %q", code, out)
		}
	}, "module_unit_body_ir_lower")
}

func TestCandidateOrdinaryProjectLink(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows project native replay; Linux execution is batched")
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "module_project_link.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::module_project_link;", "module main;", 1)
	main = strings.ReplaceAll(main, "@test", "") + `
@effects(console) fn i32 main(){
 project_link_rejects_missing_and_duplicate_ordinary_bodies();
 project_c_partition_keeps_generic_instance_namespaces();
 return 0;
}`
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	support, err := cruntime.ScalarSource(80)
	if err != nil {
		t.Fatal(err)
	}
	candidateAtlasProbeSize(t, "module_unit_c_emit", main, 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		code, out := candidateAtlasProcess(t, image, nil)
		if code != 0 {
			t.Fatalf("ordinary project link: %d %q", code, out)
		}
		packet, err := candidatebuild.Decode([]byte(out))
		if err != nil {
			t.Fatal(err)
		}
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			image, err := cdriver.BuildWindowsMSVCTargetParts(tc, profile, packet.Header, packet.Literals, packet.Units, support)
			if err != nil {
				t.Fatal(err)
			}
			candidateAggregateExecuteImage(t, image.Bytes, 0, "")
		}
	})
}
