package cli

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func atlasCopiedClosureSources() []struct{ name, source string } {
	return []struct{ name, source string }{
		{"copied-capture", `module main;fn i32 main(){var i32 offset=40;let add=fn [i32 offset] i32(i32 x){return offset+x;};offset=1;return add(2);}`},
		{"no-captures", `module main;fn i32 main(){let answer=fn [] i32(){return 42;};return answer();}`},
		{"four-captures-parameters", `module main;fn i32 main(){let i32 a=1;let i32 b=2;let i32 c=3;let i32 d=4;let sum=fn [i32 a,i32 b,i32 c,i32 d] i32(i32 p,i32 q,i32 r,i32 s){return a+b+c+d+p+q+r+s;};return sum(5,6,7,14);}`},
		{"two-owners", `module main;fn i32 other(){let second=fn [] i32(){return 2;};return second();}fn i32 main(){let first=fn [] i32(){return 40;};return first()+other();}`},
		{"generic-owner", `module main;fn i32 choose<T>(T marker){discard marker;let i32 offset=40;let add=fn [i32 offset] i32(i32 x){return offset+x;};return add(2);}fn i32 main(){discard choose<bool>(true);return choose<i32>(1);}`},
	}
}

func TestAtlasCopiedClosureSourceOracle(t *testing.T) {
	for _, fixture := range atlasCopiedClosureSources() {
		t.Run(fixture.name, func(t *testing.T) {
			root := writeWorkspaceSessionFixture(t, fixture.source)
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSuccess {
				t.Fatalf("copied closure source oracle: %d %s", code, &diagnostic)
			}
			found := false
			for _, partition := range prepared.partitions {
				data, err := atlasmap.EncodeV3(partition, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				partition, err = atlasmap.DecodeV3(data, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				again, err := atlasmap.EncodeV3(partition, 4<<20)
				if err != nil || !bytes.Equal(data, again) {
					t.Fatalf("closure codec lost canonical facts: %v", err)
				}
				if partition.Header().PackageUID != "example.session.app" {
					continue
				}
				found = true
				helpers := map[string]semanticsnapshot.Declaration{}
				for _, declaration := range partition.Declarations() {
					if strings.HasPrefix(declaration.Key.Name, "_fy_closure_") {
						helpers[declaration.Key.Name] = declaration
						if declaration.Public || declaration.Foreign || declaration.Owner != nil || declaration.Type.Kind != semanticsnapshot.TypeBuiltin || declaration.Type.Name != "i32" || len(declaration.TypeParameters) != 0 {
							t.Fatal("helper acquired a public/generic/foreign shape", declaration)
						}
						span := declaration.Span
						if span.Source != "main.fy" || span.StartByte < 0 || span.EndByte > len(fixture.source) || span.StartByte >= span.EndByte || !strings.HasPrefix(fixture.source[span.StartByte:span.EndByte], "fn [") {
							t.Fatal("helper lost its exact written closure span", span)
						}
					}
				}
				wantHelpers, wantCalls, wantArity := 1, 1, 2
				switch fixture.name {
				case "no-captures":
					wantArity = 0
				case "four-captures-parameters":
					wantArity = 8
				case "two-owners":
					wantHelpers, wantCalls, wantArity = 2, 3, 0
				case "generic-owner":
					wantHelpers, wantCalls = 2, 4
				}
				if len(helpers) != wantHelpers {
					t.Fatalf("helper inventory: got %d want %d", len(helpers), wantHelpers)
				}
				for _, declaration := range helpers {
					if len(declaration.Parameters) != wantArity {
						t.Fatal("capture/parameter transport differs", declaration)
					}
				}
				functions := partition.Functions()
				if fixture.name == "generic-owner" {
					if len(partition.Specializations()) != 2 {
						t.Fatal("missing distinct generic owner implementations")
					}
					var shared *semanticsnapshot.Span
					for _, implementation := range partition.Specializations() {
						calls := implementation.Function.Calls
						if implementation.Declaration.Name != "choose" || len(calls) != 1 || !strings.HasPrefix(calls[0].Target.Name, "_fy_closure__fy_generic_"+implementation.ID+"_") {
							t.Fatal("closure helper lost its concrete generic owner", implementation)
						}
						declaration, ok := helpers[calls[0].Target.Name]
						if !ok || shared != nil && !reflect.DeepEqual(*shared, declaration.Span) {
							t.Fatal("specialized helpers lost shared written identity", declaration)
						}
						shared = &declaration.Span
						functions = append(functions, implementation.Function)
					}
				} else {
					if len(partition.Specializations()) != 0 {
						t.Fatal("ordinary closure became a generic implementation")
					}
					if _, ok := helpers["_fy_closure_main_1"]; !ok && fixture.name != "two-owners" {
						t.Fatal("ordinary closure identity differs", helpers)
					}
					if fixture.name == "two-owners" {
						if _, ok := helpers["_fy_closure_other_1"]; !ok {
							t.Fatal("first module helper identity differs")
						}
						if _, ok := helpers["_fy_closure_main_2"]; !ok {
							t.Fatal("module-wide helper ordinal was reset")
						}
					}
				}
				calls := 0
				for _, function := range functions {
					for _, call := range function.Calls {
						calls++
						if call.TargetKind != semanticsnapshot.CallTargetResolved || call.Certainty != semanticsnapshot.CertaintyExact || call.Target.PackageInstance != partition.Header().PackageInstance {
							t.Fatal("closure call lost exact source identity", call)
						}
					}
				}
				if calls != wantCalls {
					t.Fatalf("call inventory: got %d want %d", calls, wantCalls)
				}
				t.Logf("closure helpers=%d calls=%d parameters=%d", len(helpers), calls, wantArity)
			}
			if !found {
				t.Fatal("missing closure root partition")
			}
		})
	}
}
