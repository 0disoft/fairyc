package cli

import (
	"context"
	"fmt"
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

func TestCandidateStaticOperationPipeline(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("private compiler-host replay currently requires Windows AMD64; Linux deferred")
	}
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for the native compiler-host replay")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "static_operation_pipeline.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(data), "module tests::static_operation_pipeline;", "module main;", 1)
	source, observer := candidateStaticOperationSource()
	oracle := candidateStaticOperationOracle(t, root)
	program = strings.Replace(program, "import project_compiler_core as core;", "import project_compiler_core as core;\nimport static_operation_plan_fixture as oracle;", 1)
	program = strings.ReplaceAll(program, "@test", "") + "\n@effects(console,filesystem) fn i32 main(){oracle::verify();emit_operations(b" + strconv.Quote(source) + ");return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "project_compiler_core", "module_project_identified_ir", "module_public_type_text", "module_catalog", "module_unit_c_emit", "module_project_complete_c", "module_unit_body_ir_model", "module_unit_cleanup_mir_model", "module_project_artifact_encode", "module_project_artifact_decode", "candidate_artifact_wire", "static_operation_declarations", "c_text", "syntax", "scanner")
	fixture, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "static_operation_plan_fixture.fy"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/static_operation_plan_fixture.fy", strings.Replace(string(fixture), "module tests::static_operation_plan_fixture;", "module static_operation_plan_fixture;", 1))
	copyCandidateParityModules(t, root, "generic_source_plan", "generic_source_plan_input", "generic_concrete_types", "generic_package_context", "generic_source_requests", "generic_source_dependencies", "generic_source_closure", "generic_source_dependency_model", "generic_source_dependency_json", "generic_source_catalog", "generic_package_bindings", "module_public_templates", "module_generic_interface")
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
		out, err := exec.CommandContext(ctx, imagePath, oracle).Output()
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
		for _, name := range []string{"int8_t", "int16_t", "intptr_t", "uint16_t", "uint64_t", "float", "double"} {
			if !strings.Contains(packet.Header, name+" fy_fn_") {
				t.Fatalf("missing transport C signature %s", name)
			}
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

func candidateStaticOperationSource() (string, string) {
	var src, declarations, checks strings.Builder
	src.WriteString("module root;@interface fn T transform<T>(T value){}@implements(transform,i32) fn i32 twice(i32 x){return x*2;}@implements(transform,i64) fn i64 triple(i64 x){return x*3;}fn T algorithm<T>(T x){return transform<T>(x);}@interface fn T seed<T>(){}@implements(seed,i32) fn i32 seed32(){return 42;}@interface fn T choose<T>(T a,T b,T c,T d){}@implements(choose,i32) fn i32 choose32(i32 a,i32 b,i32 c,i32 d){return b;}@interface fn T transport<T>(T x){}")
	for _, v := range []struct{ fy, c, value string }{
		{"i32", "int32_t", "INT32_MIN"}, {"i64", "int64_t", "INT64_MIN"}, {"u8", "uint8_t", "UINT8_MAX"}, {"u32", "uint32_t", "UINT32_MAX"}, {"usize", "size_t", "SIZE_MAX"}, {"bool", "bool", "true"},
		{"i8", "int8_t", "INT8_MIN"}, {"i16", "int16_t", "INT16_MIN"}, {"isize", "intptr_t", "INT64_C(-2147483649)"}, {"u16", "uint16_t", "UINT16_MAX"}, {"u64", "uint64_t", "UINT64_MAX"}, {"f32", "float", "1.25f"}, {"f64", "double", "-3.5"},
	} {
		fmt.Fprintf(&src, "@implements(transport,%s) fn %s keep_%s(%s x){return x;}pub fn %s probe_%s(%s x){return transport<%s>(x);}", v.fy, v.fy, v.fy, v.fy, v.fy, v.fy, v.fy, v.fy)
		fmt.Fprintf(&declarations, "%s fairyc_probe_%s(%s value);\n", v.c, v.fy, v.c)
		fmt.Fprintf(&checks, "if(fairyc_probe_%s(%s)!=%s)return 1;\n", v.fy, v.value, v.value)
	}
	src.WriteString("unsafe extern C fn i32 observe();fn i32 main(){assert(algorithm<i64>(14)==42);assert(seed<i32>()==42);assert(choose<i32>(1,42,3,4)==42);assert(algorithm<i32>(21)==42);return unsafe{observe()};}")
	return src.String(), "#include <stdbool.h>\n" + declarations.String() + "int32_t observe(void){\n" + checks.String() + "return 42;}\n"
}
