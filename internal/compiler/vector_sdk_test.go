package compiler_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/semantic"
)

func vectorSDKInput(t *testing.T, client string) (semantic.PackageInput, map[hir.PackageID]packagegraph.CoordinateID) {
	return vectorSDKInputWithCandidate(t, client, false)
}

func vectorSDKInputWithCandidate(t *testing.T, client string, candidate bool) (semantic.PackageInput, map[hir.PackageID]packagegraph.CoordinateID) {
	t.Helper()
	input, coordinates := vectorPackageInput(t)
	input.Files[0] = parseCompilerFile(t, 1, "main.fy", client)
	load := sdkstdlib.Load
	if candidate {
		load = sdkstdlib.LoadVectorCandidate
		input.Files = input.Files[:1]
		input.Imports[0] = semantic.PackageImport{FromPackage: 1, ImportPath: "std::vec", ToPackage: 3}
		delete(coordinates, 2)
	}
	definition, err := load()
	if err != nil {
		t.Fatal(err)
	}
	sdkCoordinate := coordinates[1]
	sdkCoordinate[0] ^= 2
	coordinates[3] = sdkCoordinate
	input.Imports = append(input.Imports, semantic.PackageImport{FromPackage: 1, ImportPath: "std::bytes", ToPackage: 3})
	for _, file := range definition.Sources {
		input.Files = append(input.Files, semantic.File{Package: 3, Path: file.Path, Module: file.Module})
	}
	seen := map[string]bool{}
	for _, file := range definition.Sources {
		for _, imported := range file.Module.Imports {
			if !seen[imported.Path] {
				seen[imported.Path] = true
				input.Imports = append(input.Imports, semantic.PackageImport{FromPackage: 3, ImportPath: imported.Path, ToPackage: 3})
			}
		}
	}
	return input, coordinates
}

func TestPrivateVectorSDKBufferOwnership(t *testing.T) {
	input, coordinates := vectorSDKInput(t, vectorSDKProgram)
	before, _ := json.Marshal(input)
	result, _, err := compiler.CheckStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("SDK vector: %v %+v", err, result.Semantic)
	}
	after, _ := json.Marshal(input)
	if string(before) != string(after) {
		t.Fatal("SDK source mutated")
	}
	runVectorStorageMIRNativeLane(t, result.MIR, hir.QualifiedModulePath(1, "main")+"::main", vectorSDKProbe, true)
}

func TestPrivateVectorSDKBufferEncapsulation(t *testing.T) {
	for name, replacement := range map[string][2]string{
		"private SDK field": {"assert(valid(ref last));", "assert(last.length == 5);"},
		"moved donor":       {"assert(retry(ref mut storage, make()));", "let bytes::Buffer donor = make(); assert(retry(ref mut storage, donor)); assert(valid(ref donor));"},
		"active borrow":     {"assert(retry(ref mut storage, make()));", "let bytes::Buffer donor = make(); let ref u8[] view = bytes::view(ref donor); assert(retry(ref mut storage, donor)); assert(view.len == 5);"},
	} {
		t.Run(name, func(t *testing.T) {
			text := strings.Replace(vectorSDKProgram, replacement[0], replacement[1], 1)
			if text == vectorSDKProgram {
				t.Fatal("rejection fixture did not change")
			}
			input, coordinates := vectorSDKInput(t, text)
			result, _, err := compiler.CheckStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
			if err == nil && result.Verified {
				t.Fatal("SDK ownership violation admitted")
			}
		})
	}
}

const vectorSDKProgram = `module main;
import storage as v;
import std::bytes as bytes;
@link_name("fairyc_fixture_resource_fail") unsafe extern C fn i32 fail_next();
@link_name("fairyc_fixture_resource_finished") unsafe extern C fn i32 finished();
fn bytes::Buffer make() {
    return switch (bytes::copy(b"Fairy")) {
        case .ok(value) : { value }
        case .err(_) : { assert(false); bytes::empty() }
    };
}
fn bool valid(ref bytes::Buffer value) {
    let ref u8[] view = bytes::view(value);
    return view.len == 5 && view[0] == 70 && view[4] == 121;
}
fn bool push_ok(ref mut v::Items<bytes::Buffer> storage, bytes::Buffer value) {
    return switch (v::push<bytes::Buffer>(storage, value)) {
        case .ok(_) : { true }
        case .err(_) : { false }
    };
}
fn bool retry(ref mut v::Items<bytes::Buffer> storage, bytes::Buffer value) {
    assert(unsafe { fail_next() } == 1);
    return switch (v::push<bytes::Buffer>(storage, value)) {
        case .ok(_) : { false }
        case .err(returned) : { assert(valid(ref returned)); push_ok(storage, returned) }
    };
}
fn i32 exercise() {
    var v::Items<bytes::Buffer> storage = v::empty_items<bytes::Buffer>();
    assert(v::length<bytes::Buffer>(ref storage) == 0);
    assert(switch (v::pop<bytes::Buffer>(ref mut storage)) { case .some(_) : { false } case .none : { true } });
    assert(retry(ref mut storage, make()));
    for (var i32 i = 1; i < 8; i += 1) { assert(push_ok(ref mut storage, make())); }
    assert(retry(ref mut storage, make()));
    let bytes::Buffer last = switch (v::pop<bytes::Buffer>(ref mut storage)) {
        case .some(value) : { value }
        case .none : { assert(false); bytes::empty() }
    };
    assert(valid(ref last));
    assert(v::length<bytes::Buffer>(ref storage) == 8);
    let v::Items<bytes::Buffer> moved = storage;
    return 41;
}
fn i32 main() { return exercise() + unsafe { finished() }; }
`

const vectorSDKProbe = `
static size_t sdk_commits = 0, sdk_failures = 0, sdk_pops = 0;
static int sdk_bad = 0;
uint32_t fairyc_fixture_resource_prepare(void *storage, size_t size, void *output) {
    uint32_t status = fy_private_vector_prepare_push_v0(storage, size, output);
    if (status == FY_STD_ALLOC_OUT_OF_MEMORY_V0) sdk_failures++;
    return status;
}
uint32_t fairyc_fixture_resource_commit(void *storage, void *slot, size_t size) {
    if (size != sizeof(struct fy_std_buffer_v0)) sdk_bad = 1;
    uint32_t status = fy_private_vector_commit_push_v0(storage, slot, size);
    if (status == 0) sdk_commits++;
    return status;
}
void *fairyc_fixture_resource_pop_slot(void *storage, size_t size) {
    void *slot = fy_private_vector_pop_slot_v0(storage, size);
    if (slot != NULL) {
        struct fy_std_buffer_v0 value;
        memcpy(&value, slot, sizeof value);
        if (size != sizeof value || value.length != 5 || value.block.data == NULL ||
            memcmp(value.block.data, "Fairy", 5) != 0) sdk_bad = 1;
        sdk_pops++;
    }
    return slot;
}
int32_t fairyc_fixture_resource_fail(void) { vector_fail = 1; return 1; }
int32_t fairyc_fixture_resource_finished(void) {
    return vector_live == 0 && vector_drop_calls == 1 && vector_drop_status == 0 &&
        sdk_commits == 9 && sdk_failures == 2 && sdk_pops == 9 && !sdk_bad ? 1 : 0;
}
`
