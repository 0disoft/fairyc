package packageartifact

import (
	"bytes"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestBoundedCallbackArtifactRebinding(t *testing.T) {
	const text = `module main;
unsafe extern C fn i32 visit(i32 value,unsafe extern C fn i32(i32,ptr mut void) cb,ptr mut void data);
unsafe fn i32 handle(i32 value,ptr mut void data){return value+1;}
fn i32 main(){return unsafe{visit(41,handle,null)};}`
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v", err, checked.Semantic)
	}
	identities := map[hir.PackageID]packagegraph.InstanceID{hir.DefaultPackage: {1}}
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: {2}}
	artifacts, err := ExtractAll(checked.MIR, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	process, err := Reassemble(artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = cbackend.Emit(process, "main::main", cbackend.Stage0Profile()); err != nil {
		t.Fatal(err)
	}
	stable, err := DescribeStableArtifact(artifacts[0], stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeStableArtifact(stable, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	stable, err = DecodeStableArtifact(encoded, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	freshCoordinates := map[hir.PackageID]packagegraph.CoordinateID{77: {2}}
	rebound, err := RebindStableArtifacts([]StableArtifact{stable}, freshCoordinates, 1000, 10000, 500)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = cbackend.Emit(rebound, hir.QualifiedModulePath(77, "main")+"::main", cbackend.Stage0WindowsMSVCProfile()); err != nil {
		t.Fatal(err)
	}
	fresh, err := ExtractAll(rebound, map[hir.PackageID]packagegraph.InstanceID{77: {1}}, freshCoordinates)
	if err != nil {
		t.Fatal(err)
	}
	stableFresh, err := DescribeStableArtifact(fresh[0], stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	reencoded, err := EncodeStableArtifact(stableFresh, stablePackageArtifactLimit)
	if err != nil || !bytes.Equal(encoded, reencoded) {
		t.Fatalf("callback package bytes changed across rebinding: %v", err)
	}
	for fi := range stable.Functions {
		for bi := range stable.Functions[fi].Function.Blocks {
			for oi := range stable.Functions[fi].Function.Blocks[bi].Operations {
				op := &stable.Functions[fi].Function.Blocks[bi].Operations[oi]
				if op.Kind == mir.OperationCCallback {
					op.Variant = FunctionID{99}.String()
					if err := ValidateStableArtifact(stable); err == nil {
						t.Fatal("unbound callback target entered stable package")
					}
					return
				}
			}
		}
	}
	t.Fatal("no stable callback target was checked")
}
