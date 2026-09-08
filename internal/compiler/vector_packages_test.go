package compiler_test

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

func vectorPackageInput(t *testing.T, replacements ...[2]string) (semantic.PackageInput, map[hir.PackageID]packagegraph.CoordinateID) {
	t.Helper()
	between := func(start, end string) string {
		_, tail, ok := strings.Cut(vectorResourceProgram, start)
		if !ok {
			t.Fatalf("missing fixture boundary %q", start)
		}
		body, _, ok := strings.Cut(tail, end)
		if !ok {
			t.Fatalf("missing fixture boundary %q", end)
		}
		return start + body
	}
	// Keep the tested container implementation unchanged except public names.
	externs := between("@link_name", "resource Item {")
	provider := "module storage;\n" + externs +
		between("resource Items<T>", "fn Item make") +
		between("fn Items<T> empty_items", "fn i32 push_ok") +
		between("fn T pop<T>", "fn i32 exercise")
	for _, declaration := range []string{"resource Items<T>", "fn Items<T> empty_items", "fn Result<u32, T> push", "fn T pop<T>"} {
		provider = strings.Replace(provider, declaration, "pub "+declaration, 1)
	}
	provider += "pub fn usize length<T>(ref Items<T> storage) { return storage.length; }\n"
	client := "module main;\nimport storage as v;\n" + externs +
		between("resource Item {", "resource Items<T>") +
		between("fn Item make", "fn Items<T> empty_items") + vectorPackageEntry
	for _, replacement := range replacements {
		changed := strings.Replace(client, replacement[0], replacement[1], 1)
		if changed == client {
			t.Fatal("package rejection fixture did not change")
		}
		client = changed
	}
	input := semantic.PackageInput{Files: []semantic.File{
		parseCompilerFile(t, 1, "main.fy", client),
		parseCompilerFile(t, 2, "storage.fy", provider),
	}, Imports: []semantic.PackageImport{{FromPackage: 1, ImportPath: "storage", ToPackage: 2}}}
	first := genericCoordinate(t)
	second := first
	second[0] ^= 1
	return input, map[hir.PackageID]packagegraph.CoordinateID{1: first, 2: second}
}

func TestPrivateVectorPackageOwnership(t *testing.T) {
	input, coordinates := vectorPackageInput(t)
	before, _ := json.Marshal(input)
	result, plan, err := compiler.CheckStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("package vector: %v %+v", err, result.Semantic)
	}
	after, _ := json.Marshal(input)
	if string(before) != string(after) {
		t.Fatal("package source mutated")
	}
	if len(mir.Verify(result.MIR)) == 0 || len(mir.VerifyStorageQueries(result.MIR)) == 0 {
		t.Fatal("ordinary verifier admitted transfers")
	}
	ordinary, _, err := compiler.CheckNominalStoragePackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err == nil && ordinary.Verified {
		t.Fatal("query-only package lane admitted transfers")
	}
	runVectorStorageMIRNativeLane(t, result.MIR, hir.QualifiedModulePath(1, "main")+"::main", vectorResourceProbe, true)
	slices.Reverse(input.Files)
	for i := range input.Files {
		input.Files[i].Package += 10
	}
	input.Imports[0].FromPackage += 10
	input.Imports[0].ToPackage += 10
	repeated, repeatedPlan, err := compiler.CheckStorageTransferPackages(input,
		map[hir.PackageID]packagegraph.CoordinateID{11: coordinates[1], 12: coordinates[2]}, genericplan.DefaultPlanLimits())
	if err != nil || !repeated.Verified || !reflect.DeepEqual(plan, repeatedPlan) {
		t.Fatalf("package order/ID drift: %v", err)
	}
}

func TestPrivateVectorPackageEncapsulation(t *testing.T) {
	for name, replacement := range map[string][2]string{
		"private field":   {"return 41;", "assert(moved.length == 8); return 41;"},
		"raw helper":      {"return 41;", "let u32 status = unsafe { v::release(null) }; return 41;"},
		"moved container": {"return 41;", "assert(v::length<Item>(ref storage) == 8); return 41;"},
		"popped donor":    {"let i32 discarded = discard_item(last);", "let i32 discarded = discard_item(last); assert(item_id(ref last) == 9);"},
		"wrong element":   {"v::pop<Item>(ref mut storage)", "v::pop<i32>(ref mut storage)"},
	} {
		t.Run(name, func(t *testing.T) {
			input, coordinates := vectorPackageInput(t, replacement)
			result, _, err := compiler.CheckStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
			if err == nil && result.Verified {
				t.Fatal("package ownership violation admitted")
			}
		})
	}
}

const vectorPackageEntry = `
fn i32 push_ok(ref mut v::Items<Item> storage, Item value) {
    return switch (v::push<Item>(storage, value)) {
        case .ok(_) : { 1 }
        case .err(_) : { 0 }
    };
}
fn i32 recover_and_retry(ref mut v::Items<Item> storage, Item value, i32 expected) {
    return switch (v::push<Item>(storage, value)) {
        case .ok(_) : { 0 }
        case .err(returned) : {
            assert(item_id(ref returned) == expected);
            push_ok(storage, returned)
        }
    };
}
fn i32 exercise() {
    var v::Items<Item> storage = v::empty_items<Item>();
    assert(v::length<Item>(ref storage) == 0);
    assert(unsafe { fail_next() } == 1);
    assert(recover_and_retry(ref mut storage, make(1), 1) == 1);
    for (var i32 id = 2; id < 9; id += 1) { assert(push_ok(ref mut storage, make(id)) == 1); }
    assert(v::length<Item>(ref storage) == 8);
    assert(unsafe { fail_next() } == 1);
    assert(recover_and_retry(ref mut storage, make(9), 9) == 1);
    let Item last = v::pop<Item>(ref mut storage);
    assert(item_id(ref last) == 9);
    let i32 discarded = discard_item(last);
    let v::Items<Item> moved = storage;
    return 41;
}
fn i32 main() { return exercise() + unsafe { finished() }; }
`
