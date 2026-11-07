package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
)

func TestCandidateCopiedClosurePipeline(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("private compiler-host replay currently requires Windows AMD64; Linux deferred")
	}
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for the native compiler-host replay")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "copied_closure_pipeline.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(data), "module tests::copied_closure_pipeline;", "module main;", 1)
	source := strings.Replace(candidateCopiedClosureProgram(), "module root;", "module root;unsafe extern C fn i32 ordered(i32 index);unsafe extern C fn i32 finished();fn T keep<T>(T value){let i32 base=40;let add=fn [i32 base] i32(i32 x){return base+x;};assert(add(2)==42);return value;}", 1)
	source = strings.Replace(source, "return add(2);", "let i32 base=39;let sum=fn [i32 base] i32(i32 a,i32 b){return base+a+b;};assert(sum(unsafe{ordered(1)},unsafe{ordered(2)})==42);assert(unsafe{finished()}==2);assert(keep<i64>(42)==42);return keep<i32>(add(2));", 1)
	view, err := os.ReadFile(filepath.Join("testdata", "copied_closure_for_in", "view.fy"))
	if err != nil {
		t.Fatal(err)
	}
	viewFunction := strings.Replace(strings.TrimPrefix(string(view), "module main;"), "fn i32 main()", "fn i32 for_in()", 1)
	source = strings.Replace(source, "module root;", "module root;"+viewFunction, 1)
	source = strings.Replace(source, "return keep<i32>(add(2));", "assert(for_in()==42);return keep<i32>(add(2));", 1)
	observer := "static int32_t position=0;int32_t ordered(int32_t index){return ++position==index?index:-1000;}int32_t finished(void){return position;}"

	program = strings.ReplaceAll(program, "@test", "") + "\n@effects(console,filesystem) fn i32 main(){emit_closures(b" + strconv.Quote(source) + ");return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "project_compiler_core", "module_project_identified_ir", "module_public_type_text", "module_catalog", "module_unit_c_emit", "module_project_complete_c", "module_unit_body_ir_model", "module_unit_cleanup_mir_model", "module_project_artifact_encode", "module_project_artifact_decode", "candidate_artifact_wire", "static_operation_declarations", "c_text", "syntax", "scanner")
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
		imagePath := filepath.Join(root, "static-operation-"+profile.Name+".exe")
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
		for _, consumer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			artifact, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, consumer, packet.Header, packet.Literals, packet.Units, runtimeSource+"\n"+observer)
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
			if !ok || exit.ExitCode() != 42 || len(output) != 0 {
				t.Fatalf("generated %s/%s: %v %q", profile.Name, consumer.Name, err, output)
			}
		}
	}
}
