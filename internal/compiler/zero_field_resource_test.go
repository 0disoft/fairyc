package compiler_test

import (
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"slices"
	"testing"
)

const zeroFieldResourceSource = `module main;
resource Empty{cleanup{}}
resource Token{cleanup{let Empty temporary=Empty{};}}
resource Parent{Token first;Token last;cleanup{}}
fn Token make(){return Token{};}
fn Token forward(Token value){return value;}
fn i32 main(){
 {let Token value=forward(make());let Token moved=value;}
 {let Parent p=Parent{.first=make(),.last=make()};}
 {let Option<Token> no=.none;}
 return 42;
}`

func TestZeroFieldResourceMoveDropAndLayout(t *testing.T) {
	result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", zeroFieldResourceSource)})
	if err != nil || !result.Verified {
		t.Fatalf("check: %v %+v %+v", err, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
	encoded, err := mircodec.Encode(result.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	program, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	out, err := interpreter.Execute(program, "main::main")
	if err != nil || out.Value != 42 {
		t.Fatalf("interpreter: %+v %v", out, err)
	}
	var types []string
	for _, event := range out.DropTrace {
		types = append(types, event.ResourceType.String())
	}
	want := []string{"main::Token", "main::Empty", "main::Parent", "main::Token", "main::Empty", "main::Token", "main::Empty"}
	if !slices.Equal(types, want) {
		t.Fatalf("drop order: %v", types)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		if _, err := cbackend.Emit(program, "main::main", profile); err != nil {
			t.Fatal(err)
		}
		storage, err := cbackend.StorageLayout(program, hir.ParseType("main::Token"), profile)
		if err != nil || storage.SizeBytes != 1 || storage.AlignmentBytes != 1 {
			t.Fatalf("private storage: %+v %v", storage, err)
		}
	}
	for _, mutation := range []string{"extra-field", "missing-definition", "invalid-drop"} {
		t.Run(mutation, func(t *testing.T) {
			forged, err := mircodec.Decode(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "missing-definition":
				forged.ResourceTypes = nil
			case "invalid-drop":
				forged.ResourceTypes[0].DropKind = "missing"
			case "extra-field":
				for fi := range forged.Functions {
					for bi := range forged.Functions[fi].Blocks {
						for oi := range forged.Functions[fi].Blocks[bi].Operations {
							op := &forged.Functions[fi].Blocks[bi].Operations[oi]
							if op.Kind == mir.OperationResourceConstruct && len(op.Fields) == 0 {
								op.Fields = []string{"fake"}
								op.Operands = []mir.Operand{{Kind: mir.OperandI32, I32: 1}}
							}
						}
					}
				}
			}
			if len(mir.Verify(forged)) == 0 {
				t.Fatal("forged empty resource admitted")
			}
		})
	}
}

func TestZeroFieldResourceNative(t *testing.T) {
	runNestedResourceCleanupNative(t, `module main;
unsafe extern C fn i32 empty_record(i32 value);
unsafe extern C fn i32 empty_finished();
resource Silent{cleanup{}}
resource Event{cleanup{assert(unsafe{empty_record(1)}==1);}}
resource Parent{Event first;Silent empty;Event last;cleanup{assert(unsafe{empty_record(9)}==1);}}
fn Event make(){return Event{};}
fn Event forward(Event value){return value;}
fn void inspect(ref Event value){}
fn i32 exercise(){
 {let Event value=forward(make());inspect(ref value);let Event moved=value;}
 {let Parent value=Parent{.first=make(),.empty=Silent{},.last=make()};}
 {let Option<Event> inactive=.none;}
 return 41;
}
fn i32 main(){return exercise()+unsafe{empty_finished()};}
`, `
static const int32_t empty_expected[]={1,9,1,1};
static size_t empty_position=0;
int32_t empty_record(int32_t value){if(empty_position>=4||value!=empty_expected[empty_position])return 0;empty_position++;return 1;}
int32_t empty_finished(void){return empty_position==4?1:0;}
`)
}
