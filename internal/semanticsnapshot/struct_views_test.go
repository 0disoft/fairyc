package semanticsnapshot_test

import (
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestStructViewSnapshotRetainsNominalElement(t *testing.T) {
	text := "module main; struct Token {i64 value;} pub fn i64 read(ref Token[] tokens) {return tokens[0].value;} fn i32 main(){return 0;}"
	result := compileSinglePackage(t, 7, "main.fy", text)
	input := packageInput(t, 7, "test.struct.snapshot", "tokens", "1.0.0", "main.fy", text, "root")
	snapshot, err := semanticsnapshot.Build(result.Semantic, result.MIR, []semanticsnapshot.PackageInput{input})
	if err != nil {
		t.Fatal(err)
	}
	read := findDeclaration(t, snapshot.Declarations(), semanticsnapshot.DeclarationFunction, "read")
	want := semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeBorrow, Arguments: []semanticsnapshot.TypeRef{{Kind: semanticsnapshot.TypeSlice, Arguments: []semanticsnapshot.TypeRef{{Kind: semanticsnapshot.TypeNominal, Name: "main::Token", PackageCoordinate: input.Coordinate.String(), Arguments: []semanticsnapshot.TypeRef{}}}}}}
	if len(read.Parameters) != 1 || !reflect.DeepEqual(read.Parameters[0].Type, want) {
		t.Fatalf("nominal slice lost: %+v", read)
	}
}

func TestMutableStructViewSnapshotRetainsNominalElement(t *testing.T) {
	text := "module main; struct Token {i64 value;} pub fn void edit(ref mut Token[] tokens) {tokens[0]=Token{.value=4000000000};} fn i32 main(){return 0;}"
	result := compileSinglePackage(t, 7, "main.fy", text)
	input := packageInput(t, 7, "test.struct.mutable.snapshot", "tokens", "1.0.0", "main.fy", text, "root")
	snapshot, err := semanticsnapshot.Build(result.Semantic, result.MIR, []semanticsnapshot.PackageInput{input})
	if err != nil {
		t.Fatal(err)
	}
	edit := findDeclaration(t, snapshot.Declarations(), semanticsnapshot.DeclarationFunction, "edit")
	want := semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeBorrow, Mutable: true, Arguments: []semanticsnapshot.TypeRef{{Kind: semanticsnapshot.TypeSlice, Arguments: []semanticsnapshot.TypeRef{{Kind: semanticsnapshot.TypeNominal, Name: "main::Token", PackageCoordinate: input.Coordinate.String(), Arguments: []semanticsnapshot.TypeRef{}}}}}}
	if len(edit.Parameters) != 1 || !reflect.DeepEqual(edit.Parameters[0].Type, want) {
		t.Fatalf("exclusive nominal slice lost: %+v", edit)
	}
}

func TestGenericStructViewHeaderRetainsSlices(t *testing.T) {
	for _, element := range []string{"Token", "T"} {
		text := "module main; struct Token {i64 value;} pub fn usize count<T>(ref " + element + "[] tokens) {return missing;} fn i32 main(){return 0;}"
		declarations, input, err := genericSourceDeclarations(t, text)
		if err != nil || len(declarations) != 1 {
			t.Fatalf("generic header: %v %+v", err, declarations)
		}
		typ := declarations[0].Parameters[0].Type
		want := semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeParameter, Name: "0"}
		if element == "Token" {
			want = semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeNominal, Name: "main::Token", PackageCoordinate: input.Coordinate.String(), Arguments: []semanticsnapshot.TypeRef{}}
		}
		if typ.Kind != semanticsnapshot.TypeBorrow || len(typ.Arguments) != 1 || typ.Arguments[0].Kind != semanticsnapshot.TypeSlice || len(typ.Arguments[0].Arguments) != 1 || !reflect.DeepEqual(typ.Arguments[0].Arguments[0], want) {
			t.Fatalf("slice header lost: %+v", typ)
		}
	}
}

func TestGenericStructViewHeaderRetainsBoolSlice(t *testing.T) {
	text := "module main; pub fn usize count<T>(ref (bool)[] tokens) {return missing;} fn i32 main(){return 0;}"
	declarations, _, err := genericSourceDeclarations(t, text)
	if err != nil || len(declarations) != 1 {
		t.Fatalf("bool slice header: %v %+v", err, declarations)
	}
	want := semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeBorrow, Arguments: []semanticsnapshot.TypeRef{{Kind: semanticsnapshot.TypeBuiltin, Name: "[bool]"}}}
	if !reflect.DeepEqual(declarations[0].Parameters[0].Type, want) {
		t.Fatalf("bool slice header lost: %+v", declarations[0])
	}
}

func TestGenericStructViewHeaderRejectsNonNominalSliceChildren(t *testing.T) {
	for _, element := range []string{"ref T", "Option<T>"} {
		text := "module main; pub fn usize count<T>(ref (" + element + ")[] tokens) {return missing;} fn i32 main(){return 0;}"
		if _, _, err := genericSourceDeclarations(t, text); err == nil {
			t.Fatalf("slice of %s admitted", element)
		}
	}
}
