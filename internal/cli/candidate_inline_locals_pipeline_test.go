package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
)

func TestCandidateInlineLocalsPipeline(t *testing.T) { runCandidateInlineLocalsPipeline(t, "locals") }
func TestCandidateInlineMutationPipeline(t *testing.T) {
	runCandidateInlineLocalsPipeline(t, "mutation")
}
func TestCandidateInlineConditionalPipeline(t *testing.T) {
	runCandidateInlineLocalsPipeline(t, "conditional")
}
func runCandidateInlineLocalsPipeline(t *testing.T, family string) {
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
	data = []byte(strings.NewReplacer(
		"return a*1000+b*100+c*10+d;", "let i32 left=a*1000+b*100;let i32 right=c*10+d;return left+right;",
		"return x+2;", "let i32 added=x+2;let doubled=added*2;return doubled/2;",
		"return -2147483648;", "let i32 minimum=-2147483648;return minimum;",
		"return a/b;", "let i32 unused=10/b;let i32 numerator=a;let i32 doubled=a*2;let i32 denominator=b;return numerator/denominator;",
	).Replace(string(data)))
	if family == "mutation" {
		data = []byte(strings.NewReplacer(
			"let i32 left=a*1000+b*100;let i32 right=c*10+d;return left+right;", "var i32 left=a*1000+b*100;let i32 right=c*10+d;left=left+right;return left;",
			"let i32 added=x+2;let doubled=added*2;return doubled/2;", "var i32 added=x;let before=added;added=before+2;let copied=added;added=copied*2;return added/2;",
			"let i32 minimum=-2147483648;return minimum;", "var i32 minimum=0;minimum=-2147483648;return minimum;",
			"let i32 unused=10/b;let i32 numerator=a;let i32 doubled=a*2;let i32 denominator=b;return numerator/denominator;", "var i32 numerator=a;let i32 unused=10/b;numerator=numerator;let i32 doubled=a*2;let i32 denominator=b;return numerator/denominator;",
		).Replace(string(data)))
	}
	if family == "conditional" {
		data = []byte(strings.NewReplacer(
			"let i32 added=x+2;", "let i32 added=if (x < 0) { 1/0 } else if (x == 0) { 2 } else { x+2 };",
			"let i32 unused=10/b;", "let i32 unused=if (b == 0) { 10/b } else { 0 };",
		).Replace(string(data)))
	}
	program := strings.Replace(string(data), "module tests::module_inline_pipeline;", "module main;", 1)
	// Emit three independently consumable target packets from one compiler image.
	// The target traps execute unused initializers before the returned expression.
	match := regexp.MustCompile(`parsed\(b("module root;[^"\n]+")\)`).FindStringSubmatch(program)
	if len(match) != 2 {
		t.Fatal("missing authored root source")
	}
	rootSource, err := strconv.Unquote(match[1])
	if err != nil {
		t.Fatal(err)
	}
	source := []string{rootSource, strings.Replace(rootSource, "return 23;", "return h::divide(2147483647,0);", 1), strings.Replace(rootSource, "return 23;", "return h::divide(2147483647,1);", 1)}
	selection := "parsed(if(scenario==0){b" + strconv.Quote(source[0]) + "}else if(scenario==1){b" + strconv.Quote(source[1]) + "}else{b" + strconv.Quote(source[2]) + "})"
	program = strings.Replace(program, match[0], selection, 1)
	program = strings.ReplaceAll(program, "compiled()", "compiled(scenario)")
	program = strings.Replace(program, "fn core::Complete compiled(scenario)", "fn core::Complete compiled(u32 scenario)", 1)
	program = strings.Replace(program, "fn void inline_roundtrip_and_partition()", "fn void inline_roundtrip_and_partition(u32 scenario)", 1)
	providerMatch := regexp.MustCompile(`parsed\(b("module helper;[^"\n]+")\)`).FindStringSubmatch(program)
	if len(providerMatch) != 2 {
		t.Fatal("missing authored provider")
	}
	provider, err := strconv.Unquote(providerMatch[1])
	if err != nil {
		t.Fatal(err)
	}
	program = strings.ReplaceAll(program, "@test ", "") + "\n@effects(console) fn i32 main(){inline_roundtrip_and_partition(0);inline_roundtrip_and_partition(1);inline_roundtrip_and_partition(2);return 0;}\n"
	program = strings.Replace(program, "module main;", "module main;import module_inline_ir as inline_proof;import module_unit_value_model as values;import module_unit_region_model as regions;", 1)
	program = strings.Replace(program, "if(!inline::zero(ir::inline_hash(ir::at(program,i)))){marked+=1;", "if(!inline::zero(ir::inline_hash(ir::at(program,i)))){if(frame.module_index==1 && frame.declaration==3){forged_inline_rejects(ir::at(program,i),ir::cleanup_ir(program),i);}marked+=1;", 1)
	program += inlineLocalsForgeryProbe
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "project_compiler_core", "module_project_identified_ir", "module_project_interfaces", "module_public_type_text", "module_catalog", "module_unit_c_emit", "module_project_complete_c", "module_unit_body_ir_model", "module_unit_cleanup_mir_model", "module_program_artifact_encode", "module_program_artifact_decode", "module_body_artifact_encode", "module_body_artifact_decode", "module_inline_identity", "candidate_artifact_wire", "syntax", "scanner", "module_project_artifact_encode", "module_project_artifact_decode", "module_program_artifact_wire")
	build, support := candidateParityBuildParts(t, root)
	support = "#include <stdio.h>\n" + strings.Replace(support, "    _Exit((int)(FY_RUNTIME_TRAP_EXIT_BASE + category));", `    fprintf(stderr, "producer trap %u", category);
    if (fy_trap_receipt_length >= 20) {
        uint64_t start = 0, end = 0;
        for (unsigned i = 0; i < 8; ++i) {
            start |= (uint64_t)fy_trap_receipt[4 + i] << (8 * i);
            end |= (uint64_t)fy_trap_receipt[12 + i] << (8 * i);
        }
        fprintf(stderr, " %.*s:%llu..%llu", (int)(fy_trap_receipt_length - 20), fy_trap_receipt + 20,
                (unsigned long long)start, (unsigned long long)end);
    }
    _Exit((int)(FY_RUNTIME_TRAP_EXIT_BASE + category));`, 1)
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
		packets := bytes.Split(out, []byte("FYCPARTS1\n"))
		if len(packets) != 4 || len(packets[0]) != 0 {
			t.Fatalf("expected three target packets, got %d", len(packets)-1)
		}
		for scenario := 0; scenario < 3; scenario++ {
			packet, err := candidatebuild.Decode(append([]byte("FYCPARTS1\n"), packets[scenario+1]...))
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
			targetRuntime := runtimeSource
			expectedExit := 23
			if scenario != 0 {
				expression := "10/b"
				category := 2
				if scenario == 2 {
					expression = "a*2"
					category = 1
				}
				start := strings.Index(provider, expression)
				if start < 0 {
					t.Fatal("missing independent trap expression")
				}
				// Private candidate categories: 2 division-by-zero, 1 integer-overflow.
				// The observer independently checks the provider byte range and runtime
				// category, without borrowing either compiler's reported source spans.
				targetRuntime = fmt.Sprintf(`#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
_Noreturn void fy_runtime_trap(uint32_t category){(void)category;_Exit(90);}
_Noreturn void fy_runtime_trap_at_v0(uint32_t category,const char *source,uint32_t length,uint64_t start,uint64_t end){if(source && length==12 && memcmp(source,"p2/helper.fy",12)==0 && category==%d && start==%d && end==%d)_Exit(42);fprintf(stderr,"trap observation: category=%%u source=%%.*s start=%%llu end=%%llu",category,(int)length,source,(unsigned long long)start,(unsigned long long)end);_Exit(91);}
`, category, start, start+len(expression))
				expectedExit = 42
			}
			for _, consumer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				artifact, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, consumer, packet.Header, packet.Literals, packet.Units, targetRuntime)
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
				if !ok || exit.ExitCode() != expectedExit || len(output) != 0 {
					t.Fatalf("generated %s/%s case %d: %v %q", profile.Name, consumer.Name, scenario, err, output)
				}
			}
		}
	}
}

// Clone detached components deliberately without Program publication; each forged
// body must fail the inline marker verifier itself, before outer source admission.
const inlineLocalsForgeryProbe = `
fn void forged_inline_rejects(ref ir::Body b,ref clean::Program p,usize frame){
    var usize mode=0;while(mode<5){
        var v::Vec<ir::Node> nodes=v::empty<ir::Node>();var usize i=0;
        while(i<ir::node_count(b)){var n=ir::node_at(b,i);if(mode==2 && n.value.kind==2 && n.value.binding==1){n.value.binding=3;}if(mode==4 && n.value.kind==2 && n.value.binding==3){n.value.binding=2;}assert(v::push<ir::Node>(ref mut nodes,n).is_ok);i+=1;}
        var v::Vec<ir::Statement> statements=v::empty<ir::Statement>();i=0;
        while(i<ir::statement_count(b)){var s=ir::statement_at(b,i);if(mode==3 && s.value.kind==2){s.value.kind=4;}assert(v::push<ir::Statement>(ref mut statements,s).is_ok);i+=1;}
        var v::Vec<values::Local> locals=v::empty<values::Local>();i=1;
        while(i<=ir::local_count(b)){var l=ir::local_at(b,i);if(i==2){if(mode==0){l.kind=if(l.kind==2){3}else{2};}if(mode==1){l.initializer=ir::local_at(b,3).initializer;}}assert(v::push<values::Local>(ref mut locals,l).is_ok);i+=1;}
        let text=switch(bytes::copy(ir::text_view(b))){case .ok(p):{p}case .err(_):{assert(false);return;}};
        let forged=ir::mark_inline(ir::assemble_body_regions(ir::body(b),ir::source_bytes(b),ir::start(b),ir::end(b),nodes,statements,locals,v::empty<values::Invocation>(),v::empty<values::Argument>(),v::empty<regions::Scope>(),v::empty<regions::Placement>(),v::empty<regions::Exit>(),text),ir::inline_hash(b));
        assert(inline_proof::check(ref forged,p,frame)!=0);mode+=1;
    }
}
`
