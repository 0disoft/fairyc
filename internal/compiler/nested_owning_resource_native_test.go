package compiler_test

import (
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/semantic"
)

const nestedOwningNativeSource = `module main;
unsafe extern C fn i32 owned_record(i32 value);
unsafe extern C fn i32 owned_mutate(ptr mut void value);
unsafe extern C fn i32 owned_finished();
resource Leaf{i32 value;cleanup{assert(unsafe{owned_record(this.value)}==1);}}
resource Empty{cleanup{assert(unsafe{owned_record(7)}==1);}}
resource Silent{cleanup{}}
resource Box{Leaf first;Empty middle;Leaf last;cleanup{assert(unsafe{owned_record(10)}==1);assert(unsafe{owned_mutate(ref mut this as ptr mut void)}==1);}}
fn Box make(){return Box{.first=Leaf{.value=1},.middle=Empty{},.last=Leaf{.value=2}};}
fn Option<Result<Box,Empty>> wrapped(){return .some(.ok(make()));}
fn Option<Result<Box,Empty>> forward(Option<Result<Box,Empty>> value){return value;}
fn Empty zero(Empty value){return value;}
fn i32 exercise(){
 {let Option<Result<Box,Empty>> absent=.none;}
 {let Option<Option<Box>> absent=.some(.none);}
 {let Option<Result<Box,Empty>> value=forward(wrapped());let Option<Result<Box,Empty>> moved=value;}
 {let Option<Result<Box,Empty>> value=.some(.err(Empty{}));}
 {let Option<Option<Box>> value=.some(.some(make()));}
 {let Result<Option<Box>,Result<Box,Empty>> value=.ok(.some(make()));discard value;}
 {let Result<Option<Box>,Result<Box,Empty>> value=.err(.ok(make()));discard value;}
 {let Result<Option<Box>,Result<Box,Empty>> value=.err(.err(Empty{}));discard value;}
 {let Option<Result<Box,Empty>> value=wrapped();let Result<Box,Empty> extracted=switch(value){case .some(inner):{inner}case .none:{assert(false);}};discard extracted;}
 {var Option<Option<Box>> value=.some(.some(make()));value=.none;}
 {let Empty value=zero(Empty{});let Empty moved=value;let Silent noop=Silent{};}
 return 41;
}
fn i32 main(){return exercise()+unsafe{owned_finished()};}
`

const nestedOwningNativeC = `
#include <string.h>
static const int32_t owned_expected[]={10,2,7,99,7,10,2,7,99,10,2,7,99,10,2,7,99,7,10,2,7,99,10,2,7,99,7};
static size_t owned_position=0;
int32_t owned_record(int32_t value){if(owned_position>=sizeof owned_expected/sizeof owned_expected[0]||value!=owned_expected[owned_position])return 0;owned_position++;return 1;}
int32_t owned_mutate(void *value){int32_t replacement=99;memcpy(value,&replacement,sizeof replacement);return 1;}
int32_t owned_finished(void){return owned_position==sizeof owned_expected/sizeof owned_expected[0]?1:0;}
`

func TestNestedOwningResourceWindows(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires native Windows AMD64 MSVC")
	}
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", nestedOwningNativeSource)})
	if err != nil || !checked.Verified {
		t.Fatalf("source: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	unit, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []cdriver.ConsumerMode{cdriver.C11ConsumerMode, cdriver.C17ConsumerMode} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(string(mode)+"/"+profile.Name, func(t *testing.T) {
				a, err := cdriver.BuildWindowsMSVCABIProbeWithConsumerMode(tc, profile, mode, unit.Source, scalar+"\n"+nestedOwningNativeC)
				if err != nil {
					t.Fatal(err)
				}
				if len(a.Objects) != 2 {
					t.Fatal("independent generated/C observer objects missing")
				}
				if _, err := pecoff.InspectExecutable(a.Bytes); err != nil {
					t.Fatal(err)
				}
				if _, err := pecoff.InspectAMD64Unwind(a.Bytes); err != nil {
					t.Fatal(err)
				}
				out, err := cdriver.ExecuteWindowsMSVCABIProbe(a)
				if err != nil || out.ExitCode != 42 || out.Stdout != "" || out.Stderr != "" {
					t.Fatalf("cleanup: %+v %v", out, err)
				}
				t.Logf("MSVC %s: exact 27 cleanup events, inactive payloads skipped, parent mutation reaches child; exit42 empty streams", tc.Version)
			})
		}
	}
}

func TestNestedOwningResourceLinux(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("deferred to batched native Linux validation")
	}
	runNestedResourceCleanupNative(t, nestedOwningNativeSource, nestedOwningNativeC)
}
