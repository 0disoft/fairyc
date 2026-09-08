package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestPrivateVectorGenericScalarOwnership(t *testing.T) {
	// Reuse the exact container declarations exercised with owning Items. Only
	// the entry workload and external observation probe differ for copy scalars.
	declarations, _, found := strings.Cut(vectorResourceProgram, "fn i32 exercise() {")
	if !found {
		t.Fatal("resource fixture entry boundary missing")
	}
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", declarations+vectorGenericScalarEntry)
	result, _, err := compiler.CheckStorageTransferFunctions(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("generic scalar container: %v %+v", err, result.Semantic)
	}
	runVectorStorageMIRNativeLane(t, result.MIR, "main::main", vectorGenericScalarProbe, true)
}

const vectorGenericScalarEntry = `
fn i32 scalar_push(ref mut Items<i32> storage, i32 value) {
    return switch (push<i32>(storage, value)) {
        case .ok(_) : { 1 }
        case .err(returned) : { assert(returned == value); 0 }
    };
}
fn i32 exercise() {
    var Items<i32> storage = empty_items<i32>();
    assert(storage.element_size == 4 && storage.element_alignment == 4);
    assert(unsafe { fail_next() } == 1);
    assert(scalar_push(ref mut storage, 1) == 0 && storage.length == 0);
    for (var i32 value = 1; value < 9; value += 1) { assert(scalar_push(ref mut storage, value) == 1); }
    assert(unsafe { fail_next() } == 1);
    assert(scalar_push(ref mut storage, 9) == 0 && storage.length == 8);
    assert(scalar_push(ref mut storage, 9) == 1);
    assert(pop<i32>(ref mut storage) == 9 && storage.length == 8);
    let Items<i32> moved = storage;
    return 41;
}
fn i32 main() { return exercise() + unsafe { finished() }; }
`

const vectorGenericScalarProbe = `
static size_t scalar_writes = 0, scalar_pops = 0, scalar_failures = 0;
static int scalar_bad = 0;
uint32_t fairyc_fixture_resource_prepare(void *storage, size_t size, void *output) {
    struct fy_private_vector_storage_v0 before;
    void *prior;
    memcpy(&before, storage, sizeof before);
    memcpy(&prior, output, sizeof prior);
    uint32_t status = fy_private_vector_prepare_push_v0(storage, size, output);
    if (status != 0) {
        void *after;
        memcpy(&after, output, sizeof after);
        if (status != FY_STD_ALLOC_OUT_OF_MEMORY_V0 || memcmp(&before, storage, sizeof before) || prior != after) scalar_bad = 1;
        scalar_failures++;
    }
    return status;
}
uint32_t fairyc_fixture_resource_commit(void *storage, void *slot, size_t size) {
    int32_t value;
    if (size != sizeof value) { scalar_bad = 1; return FY_STD_ALLOC_INVALID_ARGUMENT_V0; }
    memcpy(&value, slot, sizeof value);
    if (value != (int32_t)(scalar_writes + 1)) scalar_bad = 1;
    uint32_t status = fy_private_vector_commit_push_v0(storage, slot, size);
    if (status != 0) scalar_bad = 1;
    scalar_writes++;
    return status;
}
void *fairyc_fixture_resource_pop_slot(void *storage, size_t size) {
    void *slot = fy_private_vector_pop_slot_v0(storage, size);
    if (slot == NULL || size != sizeof(int32_t)) { scalar_bad = 1; return slot; }
    int32_t value;
    memcpy(&value, slot, sizeof value);
    if (scalar_pops >= 9 || value != (int32_t)(9 - scalar_pops)) scalar_bad = 1;
    scalar_pops++;
    return slot;
}
int32_t fairyc_fixture_resource_fail(void) { vector_fail = 1; return 1; }
int32_t fairyc_fixture_resource_finished(void) {
    return fairyc_fixture_vector_finished() == 1 && scalar_writes == 9 && scalar_pops == 9 && scalar_failures == 2 && !scalar_bad;
}
`
