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

func TestCandidateGenericTransportPipeline(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("private compiler-host replay currently requires Windows AMD64; Linux deferred")
	}
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for the native compiler-host replay")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "module_generic_transport.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(data), "module tests::module_generic_transport;", "module main;", 1)
	source, observer := candidateGenericTransportSource()
	program = strings.ReplaceAll(program, "@test", "") + "\n@effects(console) fn i32 main(){extended_transport_keeps_operations_and_composite_signatures_closed();emit_transport(b" + strconv.Quote(source) + ");return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "project_compiler_core", "module_project_identified_ir", "module_public_type_text", "module_catalog", "module_unit_c_emit", "module_project_complete_c", "module_unit_body_ir_model", "module_unit_cleanup_mir_model", "module_project_artifact_encode", "module_project_artifact_decode", "candidate_artifact_wire", "generic_transport_scalars", "c_text", "syntax", "scanner")
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
		imagePath := filepath.Join(root, "generic-transport-"+profile.Name+".exe")
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

// Observe direct private C function transport without widening the Fairy extern-C
// scalar ABI or depending on candidate aggregate-by-value FFI support.
func candidateGenericTransportSource() (string, string) {
	var src, declarations, checks strings.Builder
	src.WriteString("module root;fn T id<T>(T value){var T copy=value;copy=value;return copy;}fn T relay<T>(T value){defer sink<T>(value);return id<T>(value);}fn void sink<T>(T value){discard value;}")
	for _, v := range []struct{ fy, c, value string }{
		{"i8", "int8_t", "INT8_MIN"}, {"i16", "int16_t", "INT16_MIN"}, {"isize", "intptr_t", "INT64_C(-2147483649)"},
		{"u16", "uint16_t", "UINT16_MAX"}, {"u64", "uint64_t", "UINT64_MAX"}, {"f32", "float", "1.25f"}, {"f64", "double", "-3.5"},
	} {
		fmt.Fprintf(&src, "pub fn %s probe_%s(%s value){return relay<%s>(value);}", v.fy, v.fy, v.fy, v.fy)
		fmt.Fprintf(&declarations, "%s fairyc_probe_%s(%s value);\n", v.c, v.fy, v.c)
		fmt.Fprintf(&checks, "if(fairyc_probe_%s(%s)!=%s)return 1;\n", v.fy, v.value, v.value)
	}
	src.WriteString("unsafe extern C fn i32 observe();fn i32 main(){return unsafe{observe()};}")
	return src.String(), declarations.String() + "int32_t observe(void){\n" + checks.String() + "return 42;}\n"
}
