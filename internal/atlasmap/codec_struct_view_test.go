package atlasmap

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestStructViewMapV3RoundTrip(t *testing.T) {
	workspace, _ := loadGenericPlanWorkspace(t, false)
	source := "module main; pub struct Token {i64 value;} pub fn i64 read(ref Token[] tokens){return tokens[0].value;} pub fn usize count<T>(ref T[] tokens){return missing;} fn i32 main(){return 0;}"
	if err := os.WriteFile(filepath.Join(workspace.Root, "src", "main.fy"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	workspace, err := workspaceloader.Load(workspace.Root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v %+v", err, result.Semantic)
	}
	_, plans, err := PrepareWorkspaceV3(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	var visit func(semanticsnapshot.TypeRef)
	visit = func(typ semanticsnapshot.TypeRef) {
		if typ.Kind == semanticsnapshot.TypeSlice {
			count++
		}
		for _, arg := range typ.Arguments {
			visit(arg)
		}
	}
	for _, plan := range plans {
		encoded, err := EncodeV3(plan, testMapLimit)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeV3(encoded, testMapLimit)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(documentFromPartition(plan), documentFromPartition(decoded)) {
			t.Fatal("round trip lost struct slice identities")
		}
		again, err := EncodeV3(decoded, testMapLimit)
		if err != nil || !bytes.Equal(encoded, again) {
			t.Fatalf("noncanonical: %v", err)
		}
		for _, d := range decoded.Declarations() {
			visit(d.Type)
			for _, p := range d.Parameters {
				visit(p.Type)
			}
		}
	}
	if count == 0 {
		t.Fatal("no struct or generic slice survived map transport")
	}
}

func TestStructSliceTransportRejectsInvalidShape(t *testing.T) {
	parameter := semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeParameter, Name: "0"}
	good := semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeSlice, Arguments: []semanticsnapshot.TypeRef{parameter}}
	if err := validateBoundTypeRef(good, 0, 1); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []semanticsnapshot.TypeRef{
		{Kind: semanticsnapshot.TypeSlice},
		{Kind: semanticsnapshot.TypeSlice, Mutable: true, Arguments: good.Arguments},
		{Kind: semanticsnapshot.TypeSlice, Length: 1, Arguments: good.Arguments},
		{Kind: semanticsnapshot.TypeSlice, Arguments: []semanticsnapshot.TypeRef{parameter, parameter}},
		{Kind: semanticsnapshot.TypeSlice, Arguments: []semanticsnapshot.TypeRef{{Kind: semanticsnapshot.TypeBuiltin, Name: "bool"}}},
		{Kind: semanticsnapshot.TypeSlice, Arguments: []semanticsnapshot.TypeRef{good}},
	} {
		if err := validateBoundTypeRef(bad, 0, 1); err == nil {
			t.Fatalf("invalid slice accepted: %+v", bad)
		}
	}
	if err := validateBoundTypeRef(good, 0, 0); err == nil {
		t.Fatal("unbound slice parameter accepted")
	}
}
