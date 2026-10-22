package compiler_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestCEnumFieldsRoundTrip(t *testing.T) {
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", `module main;
@repr(c::uchar) enum Tiny { Off; On; }
@repr(c::uint) enum State { Ready; Busy; Done; }
@repr(C) struct Packet { u8 lead; Tiny tiny; State state; u8 tail; }
@repr(C) struct Envelope { Packet packet; }
fn i32 score(State s) { return switch(s) { case .Ready:{0} case .Busy:{1} case .Done:{42} }; }
fn i32 main() {
    var Packet p=Packet{.lead=7,.tiny=.On,.state=.Busy,.tail=9};
    let Packet copy=p;
    p.state=.Done;
    let Envelope e=Envelope{.packet=p};
    assert(score(copy.state)==1 && score(e.packet.state)==42 && e.packet.lead==7 && e.packet.tail==9);
    return switch(e.packet.tiny) { case .Off:{0} case .On:{42} };
}`)})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	encoded, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	program, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	result, err := interpreter.Execute(program, "main::main")
	if err != nil || result.Kind != interpreter.OutcomeReturn || result.Value != 42 {
		t.Fatalf("execute: %+v %v", result, err)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.Emit(program, "main::main", profile)
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range []string{"typedef unsigned char fy_enum_", "typedef unsigned int fy_enum_", "fairyc C enum", "state) == UINT64_C(4)"} {
			if !strings.Contains(unit.Source, fragment) {
				t.Fatalf("missing %q", fragment)
			}
		}
	}
}

func TestCEnumIntegerBaseEmission(t *testing.T) {
	count := 0
	for _, base := range hir.CScalarTypes() {
		if hir.CEnumVariantCapacity(strings.ReplaceAll(base.String(), "::", ".")) == 0 {
			continue
		}
		count++
		t.Run(base.String(), func(t *testing.T) {
			source := fmt.Sprintf(`module main; @repr(%s) enum State { Zero; One; }
@repr(C) struct Packet { u8 lead; State state; }
fn i32 main(){let Packet p=Packet{.lead=7,.state=.One};return switch(p.state){case .Zero:{0}case .One:{42}};}`, strings.ReplaceAll(base.String(), ".", "::"))
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				_, err := cbackend.Emit(checked.MIR, "main::main", profile)
				if base == hir.TypeCSSizeT && profile.Target == "x86_64-windows-msvc" {
					if err == nil {
						t.Fatal("Windows admitted unavailable c.ssize_t")
					}
				} else if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	if count != 13 {
		t.Fatalf("tested %d C integer bases, want 13", count)
	}
}

func TestCEnumForeignBoundaries(t *testing.T) {
	for name, source := range map[string]string{
		"plain-enum-field": `enum State{Ready;} @repr(C) struct Packet{State state;}
fn i32 main(){let Packet p=Packet{.state=.Ready};return switch(p.state){case .Ready:{42}};}`,
		"bare-result": `@repr(c::uint) enum State{Ready;}
unsafe extern C fn State make_state(); fn i32 main(){let State s=unsafe{make_state()};return switch(s){case .Ready:{42}};}`,
		"bare-argument": `@repr(c::uint) enum State{Ready;}
unsafe extern C fn i32 consume_state(State s); fn i32 main(){return unsafe{consume_state(.Ready)};}`,
	} {
		t.Run(name, func(t *testing.T) {
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;"+source)})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			}
			if _, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0Profile()); err == nil {
				t.Fatal("unadmitted foreign enum shape emitted")
			}
		})
	}
}
