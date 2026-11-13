package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
)

// Each copied Result has its own explicit handling. Both outcomes of ? execute
// a foreign counter once, observed independently by the C caller of each probe.
const candidateResultTransportFunctions = `
unsafe extern C fn i32 result_tick();
fn Result<T,u32> result_copy<T>(Result<T,u32> value){let Result<T,u32> copied=value;discard value;var Result<T,u32> replaced=.err(9);discard replaced;replaced=copied;discard copied;return replaced;}
fn Result<T,E> result_bind<T,E>(Result<T,E> value){return value;}
fn Result<T,u32> result_attempt<T>(T value,bool fail){discard unsafe{result_tick()};if(fail){return .err(77);}return .ok(value);}
fn Result<T,u32> result_once<T>(T value,bool fail){let T actual=result_attempt<T>(value,fail)?;return .ok(actual);}
fn T result_relay<T>(T value){
    let Result<T,u32> original=.ok(value);let Result<T,u32> copied=result_copy<T>(original);discard original;
    let Result<T,u32> bound=result_bind<T,u32>(copied);discard copied;
    let Result<T,u32> successful=result_once<T>(value,false);let T actual=switch(successful){case .ok(x):{x}case .err(_):{assert(false);value}};
    let Result<T,u32> failed=result_once<T>(value,true);assert(switch(failed){case .ok(_):{false}case .err(error):{error==77}});
    return switch(bound){case .ok(x):{x}case .err(_):{assert(false);actual}};
}
`

func candidateCompositeResultSource() (string, string) {
	var raw, declarations, checks strings.Builder
	raw.WriteString("module root;" + candidateResultTransportFunctions)
	for _, value := range []struct{ fy, c, seed string }{
		{"i8", "int8_t", "INT8_MIN"}, {"i16", "int16_t", "INT16_MIN"}, {"isize", "intptr_t", "(-INTPTR_MAX-1)"},
		{"u16", "uint16_t", "UINT16_MAX"}, {"u64", "uint64_t", "UINT64_MAX"}, {"f32", "float", "1.25f"}, {"f64", "double", "-3.5"},
	} {
		fmt.Fprintf(&raw, "fn Result<%[1]s,u32> written_%[1]s(Result<%[1]s,u32> value){return value;}pub fn %[1]s probe_%[1]s(%[1]s value){let Result<%[1]s,u32> original=.ok(result_relay<%[1]s>(value));let Result<%[1]s,u32> copied=written_%[1]s(original);discard original;return switch(copied){case .ok(x):{x}case .err(_):{assert(false);value}};}", value.fy)
		fmt.Fprintf(&declarations, "%s fairyc_probe_%s(%s value);\n", value.c, value.fy, value.c)
		fmt.Fprintf(&checks, "{%s expected=%s;int32_t before=result_ticks;%s actual=fairyc_probe_%s(expected);if(memcmp(&actual,&expected,sizeof expected)!=0||result_ticks!=before+2)return 1;}\n", value.c, value.seed, value.c, value.fy)
	}
	raw.WriteString("unsafe extern C fn i32 observe();fn i32 main(){return unsafe{observe()};}")
	observer := "#include <stdint.h>\n#include <string.h>\nstatic int32_t result_ticks;int32_t result_tick(void){return ++result_ticks;}\n" + declarations.String() + "int32_t observe(void){\n" + checks.String() + `
float floats[]={0.0f,-0.0f,1.25f};double doubles[]={0.0,-0.0,-3.5};
for(size_t i=0;i<3;i++){
 int32_t before=result_ticks;float small=fairyc_probe_f32(floats[i]);
 if(memcmp(&small,&floats[i],sizeof small)!=0||result_ticks!=before+2)return 2;
 before=result_ticks;double wide=fairyc_probe_f64(doubles[i]);
 if(memcmp(&wide,&doubles[i],sizeof wide)!=0||result_ticks!=before+2)return 3;
}
return result_ticks==26?42:4;}
`
	return raw.String(), observer
}

func TestCandidateCompositeResultTransportPipeline(t *testing.T) {
	runCandidateCompositeResultTransportPipeline(t, false)
}

func TestCandidateCompositeResultSignedErrorPipeline(t *testing.T) {
	runCandidateCompositeResultTransportPipeline(t, true)
}

func runCandidateCompositeResultTransportPipeline(t *testing.T, signed bool) {
	candidateResultNativeGate(t)
	raw, observer := candidateCompositeResultSource()
	if signed {
		raw = strings.NewReplacer(",u32>", ",i32>", "return .err(77)", "return .err(-77)", "error==77", "error==-77").Replace(raw)
	}
	runCandidateTaggedTransportPipeline(t, raw, observer)
}

func runCandidateTaggedTransportPipeline(t *testing.T, raw, observer string) {
	candidateResultNativeGate(t)
	if checked := candidateResultGoCheck(t, raw); !checked.Verified {
		t.Fatalf("independent Go admission of exact Fairy input: %+v", checked.Semantic)
	}
	cache, err := filepath.Abs(filepath.Join("..", "..", ".cache"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cache, 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(cache, "flat-result-pipeline-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("retained failed frozen Result source/images: %s", root)
		} else if err := os.RemoveAll(root); err != nil {
			t.Error(err)
		}
	})
	copyLexerExample(t, root)
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "static_operation_pipeline.fy"))
	if err != nil {
		t.Fatal(err)
	}
	// This existing helper builds Complete, encodes/decodes nested body/program/
	// project artifacts, restores against the actual source, then partitions C.
	// Its bridge lookup uses written probe names, avoiding declaration-ID guesses.
	program := strings.Replace(string(data), "module tests::static_operation_pipeline;", "module main;", 1)
	program += "\n@effects(console) fn i32 main(){emit_operations(b" + strconv.Quote(raw) + ");return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "project_compiler_core", "module_project_identified_ir", "module_public_type_text", "module_catalog", "module_unit_c_emit", "module_project_complete_c", "module_unit_body_ir_model", "module_unit_cleanup_mir_model", "module_project_artifact_encode", "module_project_artifact_decode", "candidate_artifact_wire", "static_operation_declarations", "c_text", "syntax", "scanner")
	// The source closure is copied exactly once and remains frozen for both
	// producers and all four independently observed C consumer products.
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	runtimeSource, err := cruntime.ScalarSource(80)
	if err != nil {
		t.Fatal(err)
	}
	for _, producer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Logf("frozen tagged-value compiler %s: %d C parts", producer.Name, len(build.Sources))
		image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, producer, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		imagePath := filepath.Join(root, "result-"+producer.Name+".exe")
		if err := os.WriteFile(imagePath, image.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
		command := exec.CommandContext(ctx, imagePath)
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err = command.Run()
		cancel()
		if err != nil || stderr.Len() != 0 {
			t.Fatalf("Result %s producer: %v stderr=%s", producer.Name, err, &stderr)
		}
		packet, err := candidatebuild.Decode(stdout.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"int8_t", "int16_t", "intptr_t", "uint16_t", "uint64_t", "float", "double"} {
			if !strings.Contains(packet.Header, name+" fy_fn_") {
				t.Fatalf("missing actual Result transport signature %s", name)
			}
		}
		for _, consumer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			artifact, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, consumer, packet.Header, packet.Literals, packet.Units, runtimeSource+"\n"+observer)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "result-generated-"+producer.Name+"-"+consumer.Name+".exe")
			if err := os.WriteFile(path, artifact.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			output, err := exec.CommandContext(ctx, path).CombinedOutput()
			cancel()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 42 || len(output) != 0 {
				t.Fatalf("Result %s/%s: %v %q", producer.Name, consumer.Name, err, output)
			}
			t.Logf("tagged values %s/%s: exit 42, empty output; independent C observer passed", producer.Name, consumer.Name)
		}
	}
}
