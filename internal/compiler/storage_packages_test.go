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

const storageProvider = `module storage;
@link_name("fairyc_fixture_vector_initialize") unsafe extern C fn u32 initialize(ptr mut void storage, usize size, usize alignment);
@link_name("fairyc_fixture_vector_reserve") unsafe extern C fn u32 reserve_raw(ptr mut void storage, usize capacity);
@link_name("fairyc_fixture_vector_drop") unsafe extern C fn u32 release(ptr mut void storage);
pub resource Storage<T> {
    ptr mut void data; usize bytes; usize alignment; usize length; usize element_size; usize element_alignment;
    cleanup { let u32 status = unsafe { release(ref mut this as ptr mut void) }; }
}
pub fn Storage<T> empty<T>() {
    var Storage<T> value = Storage<T> { .data = null, .bytes = 0, .alignment = 0, .length = 0, .element_size = 0, .element_alignment = 0 };
    assert(unsafe { initialize(ref mut value as ptr mut void, storage_size<T>(), storage_alignment<T>()) } == 0);
    return value;
}
pub fn u32 reserve<T>(ref mut Storage<T> value, usize requested) { return unsafe { reserve_raw(value as ptr mut void, requested) }; }
pub fn usize capacity<T>(ref Storage<T> value) { return value.bytes / value.element_size; }
`

const storageClient = `module main;
import storage as v;
@link_name("fairyc_fixture_vector_finished") unsafe extern C fn i32 finished();
fn i32 exercise() {
    var v::Storage<bool> value = v::empty<bool>();
    assert(v::capacity<bool>(ref value) == 0);
    assert(v::reserve<bool>(ref mut value, 9) == 0);
    assert(v::capacity<bool>(ref value) == 16);
    let v::Storage<bool> moved = value;
    return 41;
}
fn i32 main() { return exercise() + unsafe { finished() }; }
`

func storagePackageInput(t *testing.T, client string) (semantic.PackageInput, map[hir.PackageID]packagegraph.CoordinateID) {
	t.Helper()
	input := semantic.PackageInput{Files: []semantic.File{
		parseCompilerFile(t, 1, "main.fy", client),
		parseCompilerFile(t, 2, "storage.fy", storageProvider),
	}, Imports: []semantic.PackageImport{{FromPackage: 1, ImportPath: "storage", ToPackage: 2}}}
	first := genericCoordinate(t)
	second := first
	second[0] ^= 1
	return input, map[hir.PackageID]packagegraph.CoordinateID{1: first, 2: second}
}

func TestPrivateStorageQueryPackages(t *testing.T) {
	input, coordinates := storagePackageInput(t, storageClient)
	before, _ := json.Marshal(input)
	result, plan, err := compiler.CheckStorageGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified || len(mir.Verify(result.MIR)) == 0 {
		t.Fatalf("private package compile: %v %+v", err, result.Semantic)
	}
	after, _ := json.Marshal(input)
	if string(before) != string(after) {
		t.Fatal("package inputs mutated")
	}
	ordinary, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err == nil && ordinary.Verified {
		t.Fatal("ordinary generic graph admitted private query")
	}
	runVectorStorageMIRNative(t, result.MIR, hir.QualifiedModulePath(1, "main")+"::main", "")
	slices.Reverse(input.Files)
	renumbered := map[hir.PackageID]packagegraph.CoordinateID{}
	for i := range input.Files {
		id := input.Files[i].Package
		input.Files[i].Package += 10
		renumbered[id+10] = coordinates[id]
	}
	input.Imports[0].FromPackage += 10
	input.Imports[0].ToPackage += 10
	repeated, repeatedPlan, err := compiler.CheckStorageGenericPackages(input, renumbered, genericplan.DefaultPlanLimits())
	if err != nil || !repeated.Verified || !reflect.DeepEqual(plan, repeatedPlan) {
		t.Fatalf("package order/local ID changed plan: %v", err)
	}
}

func TestPrivateStorageQueryPackageEncapsulation(t *testing.T) {
	for name, client := range map[string]string{
		"field":         strings.Replace(storageClient, "return 41;", "assert(moved.length == 0); return 41;", 1),
		"raw helper":    strings.Replace(storageClient, "return 41;", "let u32 status = unsafe { v::release(null) }; return 41;", 1),
		"donor reuse":   strings.Replace(storageClient, "return 41;", "assert(v::capacity<bool>(ref value) == 16); return 41;", 1),
		"wrong element": strings.Replace(storageClient, "v::reserve<bool>", "v::reserve<i32>", 1),
	} {
		t.Run(name, func(t *testing.T) {
			input, coordinates := storagePackageInput(t, client)
			result, _, err := compiler.CheckStorageGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
			if err == nil && result.Verified {
				t.Fatal("encapsulation violation admitted")
			}
		})
	}
}
