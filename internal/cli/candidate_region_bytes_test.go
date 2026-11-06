package cli

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
)

func candidateRegionBytesMain() string {
	main := strings.SplitN(candidateAtlasTypesMain, " if(process::arg_count()==2)", 2)[0]
	main = strings.Replace(main, "import project_atlas_types as types;", "import project_atlas_types as types;import module_unit_c_emit as emit;import module_unit_interpreter as interpreter;", 1)
	return main + `
 var usize entry=1;while(entry<=ir::count(p)){let f=clean::frame_at(clean,entry);if(input::origin(ref source,f.module_index).owner==0 && (f.kind==1 || f.kind==3) && bytes::equal(syntax::declaration_name(input::source(ref source,f.module_index),f.declaration),b"main")){break;}entry+=1;}
 if(entry>ir::count(p)){return 7;}
 let plan=switch(interpreter::prepare(p)){case .ok(p):{p}case .err(e):{discard io::write_u32(e);return 7;}};
 let arguments=v::empty<usize>();var heap=v::empty<usize>();var usize steps=1000000;
 let done=interpreter::execute(compiler::checked_program(ref complete),ref plan,entry,ref arguments,ref mut heap,ref mut steps,0);
 if(done.error!=0 || done.trap!=0 || done.bits!=0){discard io::write("interpreter/");discard io::write_u32(done.error);discard io::write("/");discard io::write_u32(done.trap);return 7;}
 var text=bytes::empty();let rendered=emit::render(p,entry,ref mut text);if(rendered.code!=0){discard io::write("emit/");discard io::write_u32(rendered.code);return 7;}
 return if(io::write_bytes(bytes::view(ref text)).is_ok){0}else{4};
}`
}

func TestCandidateRegionBytesExecution(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows Region bytes native replay; Linux execution is batched")
	}
	const prefix = `module main;import std::alloc as alloc;fn i32 main(){var alloc::Arena arena=switch(alloc::create()){case .ok(a):{a}case .err(_):{return 1;}};region r in ref mut arena{`
	const suffix = `}return 0;}`
	fixtures := []struct{ name, body string }{
		{"zero-length", `let ref mut u8[] data=switch(r.alloc_zeroed_bytes(0)){case .ok(v):{v}case .err(_):{return 2;}};assert(data.len==0);`},
		{"zeroed-mutation", `let ref mut u8[] data=switch(r.alloc_zeroed_bytes(3)){case .ok(v):{v}case .err(_):{return 2;}};assert(data.len==3 && data[0]==0 && data[2]==0);data[2]=255;assert(data[2]==255 && data[0]==0);`},
		{"empty-copy", `let ref mut u8[] data=switch(r.copy_bytes(b"")){case .ok(v):{v}case .err(_):{return 2;}};assert(data.len==0);`},
		{"binary-copy", `let ref mut u8[] data=switch(r.copy_bytes(b"A\0\xff")){case .ok(v):{v}case .err(_):{return 2;}};assert(data.len==3 && data[0]==65 && data[1]==0 && data[2]==255);data[0]=66;assert(data[0]==66);`},
		{"copy-isolation", `var u8[3] source=[1,2,3];let ref mut u8[] data=switch(r.copy_bytes(ref source[..])){case .ok(v):{v}case .err(_):{return 2;}};source[0]=9;assert(data[0]==1);data[2]=8;assert(source[2]==3);`},
		{"mutable-copy", `let ref mut u8[] source=switch(r.alloc_zeroed_bytes(1)){case .ok(v):{v}case .err(_):{return 2;}};source[0]=42;let ref mut u8[] data=switch(r.copy_bytes(source)){case .ok(v):{v}case .err(_):{return 2;}};source[0]=7;assert(data.len==1 && data[0]==42);`},
		{"region-parameter", `assert(exercise(r)==0);`},
		{"same-spelled-methods", ``},
	}
	var roots []string
	for _, f := range fixtures {
		source := prefix + f.body + suffix
		if f.name == "region-parameter" {
			source += `fn i32 exercise(Region r){let ref mut u8[] data=switch(r.alloc_zeroed_bytes(1)){case .ok(v):{v}case .err(_):{return 2;}};assert(data[0]==0);data[0]=42;return 0;}`
		}
		if f.name == "same-spelled-methods" {
			source = `module main;struct Box{i32 value;}fn i32 Box::copy_bytes(ref Box this){return this.value;}fn i32 Box::alloc_zeroed_bytes(ref Box this,usize count){return this.value;}fn i32 main(){let box=Box{.value=42};let ref Box view=ref box;assert(view.copy_bytes()==42 && view.alloc_zeroed_bytes(0)==42);return 0;}`
		}
		root := writeWorkspaceSessionFixture(t, source)
		var diagnostics bytes.Buffer
		if _, code := prepareAtlasWorkspace(root, false, &diagnostics); code != ExitSuccess {
			t.Fatalf("Go source %s: %d %s", f.name, code, &diagnostics)
		}
		roots = append(roots, root)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	support, err := nativesupport.Source(target.X8664WindowsMSVC, []string{"fy_std_arena_initialize_v0", "fy_std_arena_drop_v0", "fy_std_region_begin_v0", "fy_std_region_reset_v0", "fy_std_region_allocate_zeroed_v0", "fy_std_region_copy_bytes_v0"}, nativesupport.Options{})
	if err != nil {
		t.Fatal(err)
	}
	candidateAtlasProbeSize(t, "project_atlas_source", candidateRegionBytesMain(), 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for i, root := range roots {
			t.Run(fixtures[i].name, func(t *testing.T) {
				code, output, stderr := fetchCLIProcess(t, image, root, nil, "")
				if code != 0 || stderr != "" || !strings.Contains(output, "fy_candidate_entry") {
					t.Fatalf("Fairy interpreter/C emission: %d %.1000s %q", code, output, stderr)
				}
				for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
					native, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, profile, "/* single-buffer transport */\n", "typedef int fy_empty_literals;\n", []string{output}, support)
					if err != nil {
						t.Fatal(err)
					}
					candidateAggregateExecuteImage(t, native.Bytes, 0, "")
				}
			})
			if t.Failed() {
				return
			}
		}
	}, "project_compiler_core", "module_unit_c_emit", "module_unit_interpreter")
}
