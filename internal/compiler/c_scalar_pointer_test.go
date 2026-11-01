package compiler_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func cScalarPointerSource(typ string) string {
	return fmt.Sprintf(`module main;
@repr(C) struct Pointers{ptr %[1]s shared;ptr mut %[1]s writable;ptr mut ptr mut %[1]s slot;}
unsafe extern C fn %[1]s produce();
unsafe extern C fn ptr mut %[1]s address();
unsafe extern C fn ptr %[1]s shared_address();
unsafe extern C fn i32 fill(ptr mut ptr mut %[1]s output);
unsafe extern C fn i32 observe(ptr %[1]s shared,ptr mut %[1]s writable,Pointers packet);
fn ptr mut %[1]s identity(ptr mut %[1]s p){return p;}
fn ptr mut ptr mut %[1]s slot_identity(ptr mut ptr mut %[1]s p){return p;}
fn i32 shared_borrow(ref %[1]s value){
 let ptr %[1]s p=unsafe{value as ptr %[1]s};
 let Pointers packet=Pointers{.shared=p,.writable=null,.slot=null};
 return unsafe{observe(p,null,packet)};
}
fn i32 borrowed(ref mut %[1]s value,ref mut ptr mut %[1]s output){
 let ptr %[1]s shared=unsafe{value as ptr %[1]s};
 let ptr mut %[1]s writable=unsafe{value as ptr mut %[1]s};
 let ptr mut ptr mut %[1]s slot=unsafe{output as ptr mut ptr mut %[1]s};
 let Pointers packet=Pointers{.shared=shared,.writable=writable,.slot=slot};
 return unsafe{observe(shared,writable,packet)};
}
fn i32 main(){
 var %[1]s value=unsafe{produce()};
 var ptr mut %[1]s output=null;
 let ptr mut %[1]s p=identity(unsafe{address()});
 let ptr %[1]s q=unsafe{shared_address()};
 let i32 status=unsafe{fill(slot_identity(ref mut output as ptr mut ptr mut %[1]s))};
 let i32 seen=borrowed(ref mut value,ref mut output);
 let i32 shared_seen=shared_borrow(ref value);
 let ptr %[1]s direct=unsafe{ref value as ptr %[1]s};
 let ptr mut %[1]s direct_mut=unsafe{ref mut value as ptr mut %[1]s};
 let Pointers packet=Pointers{.shared=direct,.writable=direct_mut,.slot=null};
 return if(p==output&&q!=null&&status==42&&seen==42&&shared_seen==42){unsafe{observe(direct,direct_mut,packet)}}else{1};
}`, typ)
}

func TestCScalarPointerTransport(t *testing.T) {
	cNames := []string{"_Bool", "char", "double", "float", "int", "long", "long long", "signed char", "short", "size_t", "ssize_t", "unsigned char", "unsigned int", "unsigned long", "unsigned long long", "unsigned short"}
	for i, typ := range hir.CScalarTypes() {
		t.Run(typ.String(), func(t *testing.T) {
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", cScalarPointerSource(typ.String()))})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v diag=%+v unsupported=%+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			}
			data, err := mircodec.Encode(checked.MIR, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			program, err := mircodec.Decode(data, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			if failures := mir.Verify(program); len(failures) != 0 {
				t.Fatal(failures)
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				unit, err := cbackend.Emit(program, "main::main", profile)
				if typ == hir.TypeCSSizeT && profile.Target == cbackend.Stage0WindowsMSVCProfile().Target {
					if err == nil {
						t.Fatal("Windows ssize_t pointer admitted")
					}
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				for _, fragment := range []string{"const " + cNames[i] + " * shared;", cNames[i] + " * writable;", cNames[i] + " * * slot;", "fill(" + cNames[i] + " * * fy_a_0)", "sizeof(" + cNames[i] + ") =="} {
					if !strings.Contains(unit.Source, fragment) {
						t.Fatalf("missing exact pointer/provider fact %q", fragment)
					}
				}
			}
		})
	}
}

func TestCScalarPointerClosedShapes(t *testing.T) {
	for _, typ := range []string{"ptr ptr c::int", "ptr ptr mut c::int", "ptr mut ptr c::int", "ptr mut ptr mut ptr mut c::int", "ptr mut ptr mut i32", "ptr mut ptr mut Record", "ptr bool", "ptr i8", "ptr f64", "ptr c::int[2]"} {
		t.Run(typ, func(t *testing.T) {
			source := fmt.Sprintf("module main; @repr(C) struct Record{i32 v;} unsafe extern C fn i32 consume(%s value); fn i32 main(){return unsafe{consume(null)};}", typ)
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
			if err != nil {
				t.Fatal(err)
			}
			if checked.Verified {
				for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
					if _, err := cbackend.Emit(checked.MIR, "main::main", profile); err == nil {
						t.Fatal("unadmitted pointer shape emitted")
					}
				}
			}
		})
	}
}

func TestCScalarPointerUnsafeAndTypeBoundaries(t *testing.T) {
	for name, body := range map[string]string{
		"unsafe-required": `fn ptr c::int bad(ref c::int v){return v as ptr c::int;}`,
		"strengthening":   `fn ptr mut c::int bad(ref c::int v){return unsafe{v as ptr mut c::int};}`,
		"retype":          `fn ptr c::long bad(ref c::int v){return unsafe{v as ptr c::long};}`,
		"integer-retype":  `fn ptr i32 bad(ref c::int v){return unsafe{v as ptr i32};}`,
		"raw-cast":        `fn ptr c::long bad(ptr c::int v){return unsafe{v as ptr c::long};}`,
		"int-cast":        `fn ptr c::int bad(usize v){return unsafe{v as ptr c::int};}`,
		"deref":           `fn c::int bad(ptr c::int v){return unsafe{*v};}`,
		"arithmetic":      `fn ptr c::int bad(ptr c::int v){return unsafe{v+1};}`,
		"slot-identity":   `fn ptr mut ptr mut c::int bad(ref mut ptr mut c::long v){return unsafe{v as ptr mut ptr mut c::int};}`,
		"const-slot":      `fn ptr mut ptr mut c::int bad(ref ptr mut c::int v){return unsafe{v as ptr mut ptr mut c::int};}`,
	} {
		t.Run(name, func(t *testing.T) {
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;"+body+"fn i32 main(){return 0;}")})
			if err != nil {
				t.Fatal(err)
			}
			if checked.Verified {
				t.Fatal("unsafe/type boundary accepted")
			}
		})
	}
}

func TestCScalarPointerLocalOnlyProvider(t *testing.T) {
	source := `module main; fn i32 main(){let ptr c::ssize_t p=null;return if(p==null){0}else{1};}`
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v", err, checked)
	}
	unit, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(unit.Source, "#include <sys/types.h>") || !strings.Contains(unit.Source, "sizeof(ssize_t)") {
		t.Fatal("pointer-only local lost provider header/assertions")
	}
	if _, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
		t.Fatal("local-only POSIX pointer admitted on Windows")
	}
}

func TestCScalarPointerMIRRejectsForgedCasts(t *testing.T) {
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", cScalarPointerSource("c::int"))})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v", err, checked)
	}
	data, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []mir.OperationKind{mir.OperationRawAddress, mir.OperationRawBorrowPointer} {
		for _, mutation := range []string{"provenance", "pointee", "strengthening"} {
			t.Run(string(kind)+"/"+mutation, func(t *testing.T) {
				program, err := mircodec.Decode(data, 1<<20)
				if err != nil {
					t.Fatal(err)
				}
				mutated := false
				for fi := range program.Functions {
					for bi := range program.Functions[fi].Blocks {
						for oi := range program.Functions[fi].Blocks[bi].Operations {
							op := &program.Functions[fi].Blocks[bi].Operations[oi]
							if mutated || op.Kind != kind {
								continue
							}
							leaf, _, ok := hir.RawPointerElement(op.Type)
							if !ok || leaf != hir.TypeCInt {
								continue
							}
							mutated = true
							switch mutation {
							case "provenance":
								program.Functions[fi].UnsafeOperations = nil
							case "pointee":
								op.Type = hir.RawPointer(hir.TypeCLong, true)
							case "strengthening":
								if kind == mir.OperationRawAddress {
									op.Operator = "&"
									op.Type = hir.RawPointer(hir.TypeCInt, true)
								} else {
									op.OwnerType = hir.SharedBorrow(hir.TypeCInt)
									op.Type = hir.RawPointer(hir.TypeCInt, true)
								}
							}
						}
					}
				}
				if !mutated || len(mir.Verify(program)) == 0 {
					t.Fatal("forged C scalar pointer cast accepted")
				}
				if _, err := cbackend.Emit(program, "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
					t.Fatal("forged C scalar pointer cast emitted")
				}
			})
		}
	}
}
