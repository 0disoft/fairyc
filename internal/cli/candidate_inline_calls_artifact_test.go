package cli

import (
	"bytes"
	"context"
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

// The Go frontend builds the compiler image only. The image's authored Fairy
// parser/checker/IR codecs/Complete restore/C emitter handle every target below.
func TestCandidateInlineCallsArtifact(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows private compiler-host replay; Linux remains batched")
	}
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for authored Fairy artifact replay")
	}
	previous := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	root := t.TempDir()
	copyLexerExample(t, root)
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "module_inline_pipeline.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(data), "module tests::module_inline_pipeline;", "module main;", 1)
	program = strings.Replace(program, "return 23;", "assert(h::rival(0)==9);return 23;", 1)
	provider := regexp.MustCompile(`parsed\(b("module helper;[^"\n]+")\)`)
	match := provider.FindStringSubmatch(program)
	if len(match) != 2 {
		t.Fatal("missing authored provider")
	}
	source, err := strconv.Unquote(match[1])
	if err != nil {
		t.Fatal(err)
	}
	source = strings.Replace(source, "return x+2;", "let i32 base=relay(x);let i32 result=step(base);return finish(result,zero());", 1)
	source = strings.Replace(source, "return a/b;", "let i32 numerator=step(a)-1;return quotient(numerator,b);", 1)
	source += "@inline pub fn i32 step(i32 x){let i32 amount=1;return x+amount;}@inline pub fn i32 zero(){return 0;}@inline pub fn i32 finish(i32 a,i32 b){return a+b;}@inline pub fn i32 quotient(i32 a,i32 b){return a/b;}@inline pub fn i32 relay(i32 x){return step(x);}@inline pub fn i32 rival(i32 x){return x+9;}@inline pub fn i32 unused(i32 x){return step(x);}"
	program = strings.Replace(program, match[0], "parsed(b"+strconv.Quote(source)+")", 1)
	first, last := strings.Index(program, "fn wire::Payload alter_program("), strings.Index(program, "@effects(console) fn void member(")
	if first < 0 || last < first {
		t.Fatal("missing artifact fixture owners")
	}
	program = program[:first] + inlineCallsArtifactProbes + "\n" + program[last:]
	program = strings.Replace(program, "marked==4", "marked==11", 1)
	program = strings.Replace(program, `assert(io::write("FYCPARTS1\n").is_ok);`, `emitted_membership(ref bundle);assert(io::write("FYCPARTS1\n").is_ok);`, 1)
	if !strings.Contains(program, "emitted_membership(ref bundle);") {
		t.Fatal("missing emitted symbol membership probe")
	}
	program = strings.ReplaceAll(program, "@test ", "") + "\n@effects(console) fn i32 main(){inline_roundtrip_and_partition();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "project_compiler_core", "module_project_identified_ir", "module_project_interfaces", "module_public_type_text", "module_catalog", "module_unit_c_emit", "module_project_complete_c", "module_unit_body_ir_model", "module_unit_cleanup_mir_model", "module_program_artifact_encode", "module_program_artifact_decode", "module_body_artifact_encode", "module_body_artifact_decode", "module_inline_identity", "candidate_artifact_wire", "syntax", "scanner", "module_project_artifact_encode", "module_project_artifact_decode", "module_program_artifact_wire")
	build, support := candidateParityBuildParts(t, root)
	// Keep a failed authored assertion attributable without changing its exit,
	// location or admission policy. Successful images emit no diagnostics.
	trap := "_Noreturn void fy_runtime_trap_at_v0(uint32_t category, const char *source, uint32_t source_length, uint64_t start_byte, uint64_t end_byte) {"
	if !strings.Contains(support, trap) {
		t.Fatal("missing private compiler runtime trap observer boundary")
	}
	support = "#include <stdio.h>\n" + strings.Replace(support, trap, trap+`fprintf(stderr,"authored trap %u %.*s %llu..%llu\n",category,(int)source_length,source,(unsigned long long)start_byte,(unsigned long long)end_byte);`, 1)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	runtimeSource, err := cruntime.ScalarSource(80)
	if err != nil {
		t.Fatal(err)
	}
	for _, producer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Logf("private %s Fairy compiler: %d parts", producer.Name, len(build.Sources))
		image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, producer, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		imagePath := filepath.Join(root, "inline-calls-"+producer.Name+".exe")
		if err := os.WriteFile(imagePath, image.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
		var out, diagnostics bytes.Buffer
		command := exec.CommandContext(ctx, imagePath)
		command.Stdout, command.Stderr = &out, &diagnostics
		err = command.Run()
		cancel()
		if err != nil || diagnostics.Len() != 0 {
			if at := regexp.MustCompile(`main\.fy (\d+)\.\.(\d+)`).FindStringSubmatch(diagnostics.String()); len(at) == 3 {
				start, _ := strconv.Atoi(at[1])
				end, _ := strconv.Atoi(at[2])
				if start >= 0 && start <= end && end <= len(program) {
					t.Logf("authored failed expression: %s", program[start:end])
				}
			}
			t.Fatalf("authored Fairy %s: %v stderr=%s", producer.Name, err, &diagnostics)
		}
		packet, err := candidatebuild.Decode(out.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		// All eleven bodies round-trip; only ten are reachable from main.
		if n := strings.Count(packet.Header, "static inline int32_t fy_fn_"); n != 20 {
			t.Fatalf("ten reachable inline prototypes/definitions: got %d", n)
		}
		for _, unit := range packet.Units {
			if strings.Contains(unit, "static inline ") {
				t.Fatal("inline definition escaped the shared header")
			}
		}
		for _, consumer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			artifact, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, consumer, packet.Header, packet.Literals, packet.Units, runtimeSource)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "target-"+producer.Name+"-"+consumer.Name+".exe")
			if err := os.WriteFile(path, artifact.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			output, err := exec.CommandContext(ctx, path).CombinedOutput()
			cancel()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 23 || len(output) != 0 {
				t.Fatalf("native %s/%s: %v %q", producer.Name, consumer.Name, err, output)
			}
			t.Logf("authored Fairy %s -> native %s: artifact probes passed, exit=23", producer.Name, consumer.Name)
		}
	}
}

// Mutate actual encoded Fairy artifacts. The table widths below are the frozen
// FYBODY00 layout, independently of either frontend's computed target identity.
const inlineCallsArtifactProbes = `
fn usize occurrences(ref u8[] raw,ref u8[] needle){var usize count=0;var usize i=0;while(i<=raw.len && needle.len<=raw.len-i){if(bytes::equal(raw[i..i+needle.len],needle)){count+=1;}i+=1;}return count;}
fn void emitted_membership(ref c::Bundle bundle){
    let p=checked::ir_program(core::checked_program(complete_c::complete(bundle)));var usize marked=0;var usize included=0;var usize omitted=0;var usize i=1;
    while(i<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),i);if(!inline::zero(ir::inline_hash(ir::at(p,i)))){assert(f.kind==3 && f.module_index==1 && f.declaration>=2 && f.declaration<=13);marked+=1;var bytes::Buffer symbol=bytes::empty();assert(bytes::append(ref mut symbol,b"fy_fn_").is_ok && bytes::append_usize(ref mut symbol,i).is_ok && bytes::append_byte(ref mut symbol,40).is_ok);
        let used=f.declaration!=13;var bytes::Buffer definition=bytes::empty();assert(bytes::append(ref mut definition,b"static inline int32_t ").is_ok && bytes::append(ref mut definition,bytes::view(ref symbol)).is_ok);assert(occurrences(c::project_header(bundle),bytes::view(ref definition))==if(used){2}else{0});var usize unit=0;while(unit<c::project_unit_count(bundle)){assert(occurrences(c::project_source(bundle,unit),bytes::view(ref symbol))==0 || used);unit+=1;}if(used){included+=1;}else{omitted+=1;assert(ir::call_count(ir::at(p,i))==1 && occurrences(c::project_header(bundle),bytes::view(ref symbol))==0);var usize j=0;while(j<ir::callable_count(p)){let signature=ir::callable_at(p,j);assert(signature.module_index!=1 || signature.declaration!=13);j+=1;}}
    }i+=1;}assert(marked==11 && included==10 && omitted==1);
}
fn u8 patch_octet(usize value){let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
fn void patch_number(ref mut bytes::Buffer out,usize at,usize value){var usize rest=value;var usize i=0;while(i<8){bytes::view_mut(out)[at+7-i]=patch_octet(rest%256);rest/=256;i+=1;}}
fn bytes::Buffer forge_body(ref u8[] raw,ref ir::Body b,usize mode){
    var out=switch(bytes::copy(if(mode==0){raw[..raw.len-64]}else{raw})){case .ok(p):{p}case .err(_):{assert(false);return bytes::empty();}};if(mode==0){return out;}
    let calls=56+ir::node_count(b)*344+8+ir::statement_count(b)*96+8+ir::local_count(b)*144+8;
    assert(ir::call_count(b)==1);let call=ir::call_at(b,0);let expression=ir::node_at(b,call.expression-1).value;let path=ir::node(b,expression.left);assert(path!=0);
    if(mode==1 || mode==2 || mode==3){patch_number(ref mut out,calls+32,if(mode==1){6}else if(mode==2){11}else{12});}
    if(mode==4){let arguments=calls+88+8;let text=arguments+ir::argument_count(b)*96+32;let n=ir::node_at(b,path-1);bytes::view_mut(ref mut out)[text+n.text_start]=88;}
    if(mode==5){let arguments=calls+88+8;patch_number(ref mut out,arguments+24,expression.source_index);}
    if(mode==6){patch_number(ref mut out,calls+24,0);}
    if(mode==7){patch_number(ref mut out,calls+40,1);}
    if(mode==8){patch_number(ref mut out,calls+80,0);}
    return out;
}
fn wire::Payload replace_body(ref u8[] raw,usize target,ref u8[] replacement){
    var usize work=0;var cursor=switch(framing::cursor(raw,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return replace_body(raw,target,replacement);}};var framing::Status status=framing::Status{.code=0};var out=framing::writer(ref mut status,ref mut work);var usize i=0;
    while(i<4){let magic=if(i==0){b"FYCLEAN0"}else if(i==1){b"FYSDK000"}else if(i==2){b"FYMETA00"}else{b"FYSTATE0"};let part=framing::member_from(raw,ref mut cursor,8388608,magic,ref mut work);framing::member(ref mut out,raw[part.start..part.end],ref mut status,ref mut work);i+=1;}let count=framing::bodies_from(raw,ref mut cursor,ref mut work);framing::number(ref mut out,count,ref mut status,ref mut work);i=1;
    while(i<=count){let part=framing::member_from(raw,ref mut cursor,16777216,b"FYBODY00",ref mut work);framing::member(ref mut out,if(i==target){replacement}else{raw[part.start..part.end]},ref mut status,ref mut work);i+=1;}assert(framing::end(raw,ref mut cursor,ref mut work)==0);return switch(framing::finish(out,status)){case .ok(p):{p}case .err(_):{assert(false);replace_body(raw,target,replacement)}};
}
fn wire::Payload replace_project_program(ref u8[] raw,ref u8[] replacement){
    var usize work=0;var cursor=switch(framing::cursor_marker(raw,b"FYPRJ000",ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return replace_project_program(raw,replacement);}};discard framing::member_from(raw,ref mut cursor,framing::byte_limit(),b"FYPROG00",ref mut work);var framing::Status status=framing::Status{.code=0};var out=framing::writer_marker(b"FYPRJ000",ref mut status,ref mut work);framing::member(ref mut out,replacement,ref mut status,ref mut work);
    let stamps=framing::member_from(raw,ref mut cursor,8388608,b"FYPROV00",ref mut work);framing::member(ref mut out,raw[stamps.start..stamps.end],ref mut status,ref mut work);let ids=framing::member_from(raw,ref mut cursor,8388608,b"FYIDS000",ref mut work);framing::member(ref mut out,raw[ids.start..ids.end],ref mut status,ref mut work);assert(framing::end(raw,ref mut cursor,ref mut work)==0);return switch(framing::finish(out,status)){case .ok(p):{p}case .err(_):{assert(false);replace_project_program(raw,replacement)}};
}
@effects(console) fn void restore_checked(ref core::Complete current,ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<catalog::Binding> imports,ref coordinates::Coordinates table){
    var usize work=0;let encoded=switch(project_encode::encode(core::checked_program(current),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};let raw=wire::content(ref encoded);work=0;let saved=switch(project_decode::decode(raw,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize source_work=0;var usize artifact_work=0;var usize link_work=0;
    let restored=switch(core::restore_complete_sources(saved,sources,owners,imports,table,ref mut source_work,ref mut artifact_work,ref mut link_work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(inline::equal(interfaces::inline_hash(interfaces::at(core::public_interfaces(current),1)),interfaces::inline_hash(interfaces::at(core::public_interfaces(ref restored),1))));
    let p=checked::ir_program(core::checked_program(current));var usize relay=0;var usize step=0;var usize i=1;while(i<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),i);if(f.module_index==1 && f.declaration==11){relay=i;}if(f.module_index==1 && f.declaration==7){step=i;}i+=1;}assert(relay!=0 && step!=0);
    work=0;let encoded_program=switch(encode::encode(p,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var usize mode=0;while(mode<9){let frame=if(mode==0){step}else{relay};work=0;let packet=switch(body_encode::encode(p,frame,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};let forged=forge_body(wire::content(ref packet),ir::at(p,frame),mode);work=0;
        let body_ok=switch(body_decode::decode(bytes::view(ref forged),ir::cleanup_ir(p),frame,ref mut work)){case .ok(_):{true}case .err(_):{false}};assert(body_ok==(mode<=3 || mode==6));
        let program=replace_body(wire::content(ref encoded_program),frame,bytes::view(ref forged));work=0;let program_ok=switch(decode::decode(wire::content(ref program),ref mut work)){case .ok(_):{true}case .err(code):{if(mode==3){discard io::write_stderr_u32(code);}false}};if(program_ok!=(mode==3)){discard io::write_stderr_usize(mode);}assert(program_ok==(mode==3));
        let project=replace_project_program(raw,wire::content(ref program));work=0;discard switch(project_decode::decode(wire::content(ref project),ref mut work)){case .ok(saved):{source_work=0;artifact_work=0;link_work=0;assert(switch(core::restore_complete_sources(saved,sources,owners,imports,table,ref mut source_work,ref mut artifact_work,ref mut link_work)){case .ok(_):{false}case .err(e):{e.phase==7}});0}case .err(_):{assert(mode!=3);0}};mode+=1;
    }
}
`
