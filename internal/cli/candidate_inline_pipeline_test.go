package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
)

func TestCandidateInlinePipeline(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("private compiler-host replay currently requires Windows AMD64; Linux deferred")
	}
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for the native compiler-host replay")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "module_inline_pipeline.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(data), "module tests::module_inline_pipeline;", "module main;", 1)
	program = strings.ReplaceAll(program, "@test ", "") + "\n@effects(console) fn i32 main(){inline_roundtrip_and_partition();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "project_compiler_core", "module_project_identified_ir", "module_project_interfaces", "module_public_type_text", "module_catalog", "module_unit_c_emit", "module_project_complete_c", "module_unit_body_ir_model", "module_unit_cleanup_mir_model", "module_program_artifact_encode", "module_program_artifact_decode", "module_body_artifact_encode", "module_body_artifact_decode", "module_inline_identity", "candidate_artifact_wire", "syntax", "scanner", "module_project_artifact_encode", "module_project_artifact_decode", "module_program_artifact_wire")
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	runtimeSource, err := cruntime.ScalarSource(80)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Logf("private %s compiler: %d parts", profile.Name, len(build.Sources))
		image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		imagePath := filepath.Join(root, "inline-"+profile.Name+".exe")
		if err := os.WriteFile(imagePath, image.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
		out, err := exec.CommandContext(ctx, imagePath).Output()
		cancel()
		if err != nil {
			t.Fatalf("candidate %s: %v stderr=%s", profile.Name, err, func() []byte {
				if e, ok := err.(*exec.ExitError); ok {
					return e.Stderr
				}
				return nil
			}())
		}
		packet, err := candidatebuild.Decode(out)
		if err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(packet.Header, "static inline int32_t fy_fn_"); n != 8 {
			t.Fatalf("expected four inline prototypes and definitions, got %d", n)
		}
		for _, unit := range packet.Units {
			if strings.Contains(unit, "static inline ") {
				t.Fatal("inline body escaped the shared header")
			}
		}
		for _, consumer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			artifact, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, consumer, packet.Header, packet.Literals, packet.Units, runtimeSource)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "generated-"+profile.Name+"-"+consumer.Name+".exe")
			if err := os.WriteFile(path, artifact.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			output, err := exec.CommandContext(ctx, path).CombinedOutput()
			cancel()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 23 || len(output) != 0 {
				t.Fatalf("generated %s/%s: %v %q", profile.Name, consumer.Name, err, output)
			}
		}
	}
}
