package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const atlasStaticOperationProvider = `module answer;
@interface pub fn T transform<T>(T value){}
@implements(transform,i32) fn i32 twice(i32 value){return value*2;}
@implements(transform,i64) fn i64 triple(i64 value){return value*3;}
@implements(transform,bool) fn bool logical(bool value){return value;}
pub fn T algorithm<T>(T value){return transform<T>(value);}
pub fn i32 value(){return 42;}
fn i32 private_value(){return 1;}`

const atlasStaticOperationUsed = `module main;import answer as a;
fn i32 main(){discard a::algorithm<i64>(14);return a::algorithm<i32>(21);}`

type atlasStaticOperationFixture struct {
	name, source, provider string
	used                   bool
}

func atlasStaticOperationFixtures() []atlasStaticOperationFixture {
	return []atlasStaticOperationFixture{
		{"used", atlasStaticOperationUsed, atlasStaticOperationProvider, true},
		{"unused", `module main;import answer as a;fn i32 main(){return a::value();}`, atlasStaticOperationProvider, false},
		{"selected-provider-edit", atlasStaticOperationUsed, strings.Replace(atlasStaticOperationProvider, "value*2", "value*3", 1), true},
		{"unused-provider-edit", atlasStaticOperationUsed, strings.Replace(atlasStaticOperationProvider, "return value;", "return false;", 1), true},
		{"private-edit", atlasStaticOperationUsed, strings.Replace(atlasStaticOperationProvider, "return 1;", "return 2;", 1), true},
	}
}

func atlasStaticOperationWorkspace(t *testing.T, source, provider string) string {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, source)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", provider)
	return root
}

func atlasStaticOperationOriginalFacts(t *testing.T, partition atlasmap.Partition) string {
	t.Helper()
	data, err := json.Marshal([]any{partition.Declarations(), partition.Symbols()})
	if err != nil {
		t.Fatal(err)
	}
	// Body changes legitimately change exact-snapshot package instances. Keep
	// every other declaration/span/symbol fact in the independent comparison.
	return strings.ReplaceAll(string(data), partition.Header().PackageInstance, "current-package")
}

// Pin original declarations and concrete provider endpoints independently of the
// Fairy producer. Synthetic operation wrappers must never replace written headers.
func TestAtlasStaticOperationSpecializationOracle(t *testing.T) {
	var original atlasmap.Partition
	var originalIDs map[string]string
	for _, fixture := range atlasStaticOperationFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			root := atlasStaticOperationWorkspace(t, fixture.source, fixture.provider)
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSuccess {
				t.Fatalf("static operation source oracle: %d %s", code, &diagnostic)
			}
			before := remoteTree(t, root)
			var member, app atlasmap.Partition
			for _, partition := range prepared.partitions {
				encoded, err := atlasmap.EncodeV3(partition, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := atlasmap.DecodeV3(encoded, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				again, err := atlasmap.EncodeV3(decoded, 4<<20)
				if err != nil || !bytes.Equal(encoded, again) {
					t.Fatalf("static operation codec lost facts: %v", err)
				}
				switch decoded.Header().PackageUID {
				case "example.session.answer":
					member = decoded
				case "example.session.app":
					app = decoded
				}
			}
			if member.Header().GenericInterfaceHash == "" || app.Header().PackageInstance == "" {
				t.Fatal("missing generic interface or application")
			}
			declared := map[string]bool{}
			for _, declaration := range member.Declarations() {
				declared[declaration.Key.Name] = true
				if declaration.Key.Name == "transform" || declaration.Key.Name == "algorithm" {
					parameter := semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeParameter, Name: "0"}
					if !declaration.Public || !slices.Equal(declaration.TypeParameters, []string{"T"}) || len(declaration.Parameters) != 1 || !reflect.DeepEqual(declaration.Type, parameter) || !reflect.DeepEqual(declaration.Parameters[0].Type, parameter) {
						t.Fatal("lost written generic header", declaration)
					}
				}
			}
			wantDeclarations := map[string]bool{"transform": true, "algorithm": true, "twice": true, "triple": true, "logical": true, "value": true, "private_value": true}
			if !reflect.DeepEqual(declared, wantDeclarations) {
				t.Fatalf("source inventory differs: got %v want %v", declared, wantDeclarations)
			}
			ids := map[string]string{}
			for _, specialization := range member.Specializations() {
				if !strings.HasPrefix(specialization.ID, "g0_") || len(specialization.TypeArguments) != 1 || specialization.TypeArguments[0].Kind != semanticsnapshot.TypeBuiltin || specialization.Aggregate != nil || specialization.Cleanup != nil {
					t.Fatal("invalid operation implementation", specialization)
				}
				key := specialization.Declaration.Name + "<" + specialization.TypeArguments[0].Name + ">"
				if ids[key] != "" {
					t.Fatal("duplicate operation implementation", key)
				}
				ids[key] = specialization.ID
			}
			wantCount := 0
			if fixture.used {
				wantCount = 4
			}
			if len(ids) != wantCount || len(app.Specializations()) != 0 {
				t.Fatalf("implementation inventory differs: got %v", ids)
			}
			functions := app.Functions()
			if len(functions) != 1 || functions[0].Function.Name != "main" {
				t.Fatal("missing original root function", functions)
			}
			wantRoots := []string{""}
			if fixture.used {
				wantRoots = []string{ids["algorithm<i64>"], ids["algorithm<i32>"]}
			}
			if len(functions[0].Calls) != len(wantRoots) {
				t.Fatal("root call inventory differs", functions[0].Calls)
			}
			for i, call := range functions[0].Calls {
				wantName := "value"
				if fixture.used {
					wantName = "algorithm"
				}
				if call.Target.Name != wantName || call.Specialization != wantRoots[i] || call.Target.PackageInstance != member.Header().PackageInstance || call.TargetKind != semanticsnapshot.CallTargetResolved {
					t.Fatal("cross-package root endpoint differs", call)
				}
			}
			for _, specialization := range member.Specializations() {
				scalar := specialization.TypeArguments[0].Name
				wantTarget, wantID := "transform", ids["transform<"+scalar+">"]
				if specialization.Declaration.Name == "transform" {
					wantTarget, wantID = map[string]string{"i32": "twice", "i64": "triple"}[scalar], ""
				}
				calls := specialization.Function.Calls
				if wantTarget == "" || len(calls) != 1 || calls[0].Target.Name != wantTarget || calls[0].Specialization != wantID || calls[0].Target.PackageInstance != member.Header().PackageInstance || calls[0].TargetKind != semanticsnapshot.CallTargetResolved {
					t.Fatalf("provider endpoint differs: %+v want %s/%s", specialization, wantTarget, wantID)
				}
			}
			if fixture.name == "used" {
				original, originalIDs = member, ids
			} else {
				if atlasStaticOperationOriginalFacts(t, member) != atlasStaticOperationOriginalFacts(t, original) || member.Header().InterfaceHash != original.Header().InterfaceHash {
					t.Fatal("body/use change altered original declarations or symbol identity")
				}
				stable := fixture.name == "unused" || fixture.name == "private-edit"
				if (member.Header().GenericInterfaceHash == original.Header().GenericInterfaceHash) != stable {
					t.Fatal("provider/private edit generic interface policy differs")
				}
				if fixture.used && reflect.DeepEqual(ids, originalIDs) != stable {
					t.Fatal("provider/private edit specialization identity policy differs")
				}
			}
			if !reflect.DeepEqual(before, remoteTree(t, root)) {
				t.Fatal("oracle map encoding changed workspace")
			}
			t.Logf("original declarations=%d concrete implementations=%d", len(declared), len(ids))
		})
	}
}
