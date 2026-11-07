package packageartifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func closurePackageProgram(t *testing.T, count int) mir.Program {
	t.Helper()
	params, bindings, captures, body, arg := "", "", "", "x", 42
	if count == 4 {
		bindings = "var i32 a=seed;let i32 b=2;let i32 c=3;let i32 d=4;"
		captures, body, arg = "i32 a,i32 b,i32 c,i32 d", "a+b+c+d+x", 23
		params = "a=999;"
	}
	text := fmt.Sprintf(`module main;
struct Box{fn i32(i32) action;}
fn fn i32(i32) make(i32 seed){%s let fn i32(i32) f=fn [%s] i32(i32 x){return %s;};%s return f;}
fn i32 apply(fn i32(i32) f,i32 x){return f(x);}
fn i32 main(){let Box box=Box{.action=make(10)};let Box copy=box;let f=copy.action;return apply(f,%d);}`, bindings, captures, body, params, arg)
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("parse: %v %+v", err, parsed.Tree.Diagnostics)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil || !checked.Verified {
		t.Fatalf("source: %v %+v", err, checked.Semantic)
	}
	return checked.MIR
}

func TestClosureValuesPackageStableRebinding(t *testing.T) {
	for _, captures := range []int{0, 4} {
		t.Run(fmt.Sprint(captures), func(t *testing.T) {
			original := closurePackageProgram(t, captures)
			before, _ := json.Marshal(original)
			identities := map[hir.PackageID]packagegraph.InstanceID{hir.DefaultPackage: {1}}
			coordinates := map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: {2}}
			artifacts, err := ExtractAll(original, identities, coordinates)
			if err != nil || len(artifacts) != 1 {
				t.Fatal("extract", err)
			}
			process, err := Reassemble(artifacts)
			if err != nil {
				t.Fatal("process reassemble", err)
			}
			if out, err := interpreter.Execute(process, "main::main"); err != nil || out.Value != 42 {
				t.Fatalf("process execution: %+v %v", out, err)
			}
			codec, err := NewProcessCodec(8 << 20)
			if err != nil {
				t.Fatal(err)
			}
			wire, err := codec.Encode(artifacts[0])
			if err != nil {
				t.Fatal("process encode", err)
			}
			if _, err := codec.Decode(wire); err != nil {
				t.Fatal("process decode", err)
			}
			legacy := bytes.Replace(wire, []byte(ClosureProcessCodecSchema), []byte(ProcessCodecSchema), 1)
			legacy = bytes.Replace(legacy, []byte(ClosureProcessCodecVersion), []byte(ProcessCodecVersion), 1)
			if _, err := codec.Decode(legacy); err == nil {
				t.Fatal("legacy process version admitted closure")
			}
			stable, err := DescribeStableArtifact(artifacts[0], stablePackageArtifactLimit)
			if err != nil {
				t.Fatal("stable projection", err)
			}
			encoded, err := EncodeStableArtifact(stable, stablePackageArtifactLimit)
			if err != nil {
				t.Fatal("stable encode", err)
			}
			if !bytes.Contains(encoded, []byte(ClosureStableArtifactVersion)) {
				t.Fatal("closure version missing")
			}
			decoded, err := DecodeStableArtifact(encoded, stablePackageArtifactLimit)
			if err != nil {
				t.Fatal("stable decode", err)
			}
			freshCoordinates := map[hir.PackageID]packagegraph.CoordinateID{77: {2}}
			rebound, err := RebindStableArtifacts([]StableArtifact{decoded}, freshCoordinates, 1000, 10000, 500)
			if err != nil {
				t.Fatal("stable rebind", err)
			}
			entry := hir.QualifiedModulePath(77, "main") + "::main"
			out, err := interpreter.Execute(rebound, entry)
			if err != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 42 {
				t.Fatalf("restored execution: %+v %v", out, err)
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				if _, err := cbackend.Emit(rebound, entry, profile); err != nil {
					t.Fatal("restored C", err)
				}
			}
			fresh, err := ExtractAll(rebound, map[hir.PackageID]packagegraph.InstanceID{77: {1}}, freshCoordinates)
			if err != nil {
				t.Fatal("fresh extract", err)
			}
			freshStable, err := DescribeStableArtifact(fresh[0], stablePackageArtifactLimit)
			if err != nil {
				t.Fatal("fresh stable", err)
			}
			freshBytes, err := EncodeStableArtifact(freshStable, stablePackageArtifactLimit)
			if err != nil || !bytes.Equal(encoded, freshBytes) {
				t.Fatal("package/function/local renumbering changed stable bytes", err)
			}
			after, _ := json.Marshal(original)
			if !bytes.Equal(before, after) {
				t.Fatal("artifact extraction mutated source MIR")
			}
			for _, body := range stable.Functions {
				bodyBytes, err := EncodeStableBody(body, stablePackageArtifactLimit)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := DecodeStableBody(bodyBytes, stablePackageArtifactLimit); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
