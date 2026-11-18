package cli

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

func TestAtlasPrepareSDKVectorWorkspace(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, `module main;import std::vec as v;fn i32 main(){var values=v::empty<i32>();assert(v::push<i32>(ref mut values,42).is_ok);return *v::at<i32>(ref values,0);}`)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("prepare SDK vector workspace: exit %d: %s", code, &diagnostic)
	}
	if len(prepared.partitions) == 0 {
		t.Fatal("prepare SDK vector workspace returned no partitions")
	}
	declarations := map[string]bool{}
	for _, p := range prepared.partitions {
		data, err := atlasmap.EncodeV3(p, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := atlasmap.DecodeV3(data, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range decoded.Declarations() {
			if d.Key.Module == "std::vec" {
				declarations[d.Key.Name] = true
			}
		}
	}
	for _, name := range []string{"Vec", "empty", "push", "at"} {
		if !declarations[name] {
			t.Fatal("missing authenticated SDK declaration", name)
		}
	}
	result, err := workspacecompiler.CheckGeneric(prepared.workspace)
	if err != nil || !result.Verified || !result.VerificationCapabilities.AllowsStorageTransfers() {
		t.Fatalf("lost SDK MIR verification admission: %v %v", result.Verified, err)
	}
	build, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	result.VerificationCapabilities = admission.Capabilities{}
	if _, _, err := atlasmap.PrepareWorkspaceV3(prepared.workspace, result, build); err == nil || !strings.Contains(err.Error(), "storage-layout") {
		t.Fatalf("snapshot accepted storage MIR without its admission: %v", err)
	}
}

// Fix the independent source/implementation expectations before adding the
// Fairy Vec producer. An unused import can still instantiate SDK-owned types
// required by ordinary SDK signatures; it must not instantiate root-only calls.
func TestAtlasSDKVectorSpecializationOracle(t *testing.T) {
	const unused = `module main;import std::vec as v;fn i32 main(){return 0;}`
	const used = `module main;import std::vec as v;fn i32 main(){var values=v::empty<i32>();assert(v::push<i32>(ref mut values,42).is_ok);return *v::at<i32>(ref values,0);}`
	var original atlasmap.Partition
	for _, fixture := range []struct{ name, source string }{{"unused", unused}, {"instantiated", used}} {
		t.Run(fixture.name, func(t *testing.T) {
			root := writeWorkspaceSessionFixture(t, fixture.source)
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSuccess {
				t.Fatalf("vector specialization oracle: %d %s", code, &diagnostic)
			}
			var sdk, app atlasmap.Partition
			for _, p := range prepared.partitions {
				data, err := atlasmap.EncodeV3(p, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := atlasmap.DecodeV3(data, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				roundtrip, err := atlasmap.EncodeV3(decoded, 4<<20)
				if err != nil || !bytes.Equal(data, roundtrip) {
					t.Fatalf("vector specialization codec lost canonical facts: %v", err)
				}
				switch p.Header().PackageUID {
				case "fairyc.std":
					sdk = decoded
				case "example.session.app":
					app = decoded
				}
			}
			if sdk.Header().GenericInterfaceHash == "" || app.Header().PackageInstance == "" {
				t.Fatal("missing SDK generic interface or application")
			}
			instances := map[string]semanticsnapshot.SpecializationSummary{}
			observed := map[string]bool{}
			explicit := map[string]string{}
			var vector semanticsnapshot.SpecializationSummary
			for _, s := range sdk.Specializations() {
				if _, exists := instances[s.ID]; exists || !strings.HasPrefix(s.ID, "g0_") {
					t.Fatal("duplicate or non-source specialization identity", s.ID)
				}
				instances[s.ID] = s
				if s.Declaration.Module != "std::vec" || len(s.TypeArguments) != 1 || s.TypeArguments[0].Kind != semanticsnapshot.TypeBuiltin {
					t.Fatal("unexpected SDK specialization family", s)
				}
				observed[s.Declaration.Name+"<"+s.TypeArguments[0].Name+">"] = true
				if s.Declaration.Module != "std::vec" || len(s.TypeArguments) != 1 || s.TypeArguments[0].Kind != semanticsnapshot.TypeBuiltin || s.TypeArguments[0].Name != "i32" {
					continue
				}
				switch s.Declaration.Name {
				case "empty", "push", "at":
					explicit[s.Declaration.Name] = s.ID
				case "Vec":
					vector = s
				}
			}
			want := map[string]bool{}
			for _, scalar := range []string{"u8", "i64", "i32", "u32", "usize"} {
				want["Vec<"+scalar+">"] = true
				want["take_slot<"+scalar+">"] = true
			}
			if fixture.name == "instantiated" {
				for _, name := range []string{"empty", "push", "at"} {
					want[name+"<i32>"] = true
				}
			}
			if len(instances) != len(want) || !reflect.DeepEqual(observed, want) {
				t.Fatalf("SDK specialization closure differs: got %v want %v", observed, want)
			}
			t.Logf("SDK specializations=%d root-requested=%d", len(instances), len(explicit))
			if fixture.name == "unused" {
				if len(explicit) != 0 {
					t.Fatal("unused import acquired root-requested function implementations", explicit)
				}
				original = sdk
				return
			}
			if sdk.Header().GenericInterfaceHash != original.Header().GenericInterfaceHash || !reflect.DeepEqual(sdk.Declarations(), original.Declarations()) {
				t.Fatal("usage changed original SDK declarations or generic interface")
			}
			if len(explicit) != 3 || vector.Aggregate == nil || vector.Cleanup == nil || vector.Aggregate.Key != vector.Declaration || vector.Cleanup.Function != vector.Declaration {
				t.Fatal("missing separate original/function/aggregate/cleanup facts", explicit, vector)
			}
			fields := vector.Aggregate.Fields
			for i, name := range []string{"data", "bytes", "alignment", "length", "element_size", "element_alignment"} {
				if len(fields) != 6 || fields[i].Name != name || fields[i].Span.Source != vector.Aggregate.Span.Source {
					t.Fatal("vector physical field identity lost", fields)
				}
				if i == 0 {
					value := fields[i].Type
					if value.Kind != semanticsnapshot.TypePointer || !value.Mutable || len(value.Arguments) != 1 || value.Arguments[0].Kind != semanticsnapshot.TypeBuiltin || value.Arguments[0].Name != "void" {
						t.Fatal("vector storage pointer type changed", value)
					}
				} else if fields[i].Type.Kind != semanticsnapshot.TypeBuiltin || fields[i].Type.Name != "usize" {
					t.Fatal("vector storage metadata type changed", fields[i])
				}
			}
			if len(vector.Cleanup.UnsafeSites) == 0 || len(vector.Cleanup.Calls) == 0 || len(instances[explicit["at"]].Function.Borrows) == 0 {
				t.Fatal("vector cleanup unsafe/call facts or element-borrow facts lost")
			}
			seen := map[string]bool{}
			for _, f := range app.Functions() {
				if f.Function.Name != "main" {
					continue
				}
				for _, call := range f.Calls {
					if call.Target.Module != "std::vec" {
						continue
					}
					if explicit[call.Target.Name] != call.Specialization || call.Specialization == "" || instances[call.Specialization].Declaration != call.Target || call.Span.StartByte < 0 || call.Span.EndByte > len(fixture.source) || call.Span.EndByte <= call.Span.StartByte {
						t.Fatal("root call lost its exact SDK implementation or source span", call)
					}
					seen[call.Target.Name] = true
				}
			}
			if len(seen) != 3 {
				t.Fatal("missing exact root SDK calls", seen)
			}
		})
	}
}
