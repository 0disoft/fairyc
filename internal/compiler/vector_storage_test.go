package compiler_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestPrivateVectorStorageGeneratedNative(t *testing.T) {
	runPrivateVectorStorageNative(t, vectorStorageProgram, "")
}

func runPrivateVectorStorageNative(t *testing.T, program, probe string) {
	runVectorStorageNative(t, program, probe, false)
}

func runPrivateVectorStorageNativeWithQueries(t *testing.T, program, probe string) {
	runVectorStorageNative(t, program, probe, true)
}

func runVectorStorageNative(t *testing.T, program, probe string, queries bool) {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("private vector storage proof requires Windows amd64")
	}
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", program)
	var result compiler.Result
	var err error
	if queries {
		result, _, err = compiler.CheckStorageGenericFunctions(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
	} else {
		result, err = compiler.CheckExplicitFiles([]semantic.File{file})
	}
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v phase=%s diagnostics=%+v unsupported=%+v", err,
			result.Semantic.FailedPhase, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
	runVectorStorageMIRNative(t, result.MIR, "main::main", probe)
}

func runVectorStorageMIRNative(t *testing.T, program mir.Program, entry, probe string) {
	runVectorStorageMIRNativeLane(t, program, entry, probe, false)
}

func runVectorStorageMIRNativeLane(t *testing.T, program mir.Program, entry, probe string, transfers bool) {
	t.Helper()
	runVectorStorageMIRNativeExit(t, program, entry, probe, transfers, 42)
}

func runVectorStorageMIRNativeExit(t *testing.T, program mir.Program, entry, probe string, transfers bool, expectedExit int32) {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("private vector storage proof requires Windows amd64")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	emit := cbackend.Emit
	if transfers {
		emit = cbackend.EmitStorageTransfers
	}
	unit, err := emit(program, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	storage, err := cstdlib.WindowsMSVCVectorStorageSource()
	if err != nil {
		t.Fatal(err)
	}
	symbols := []string{"fy_std_system_heap_allocate_v0"}
	for _, symbol := range unit.ForeignSymbols {
		if strings.HasPrefix(symbol, "fy_std_") {
			symbols = append(symbols, symbol)
		}
	}
	sdk, err := cstdlib.WindowsMSVCSourceForSymbols(symbols)
	if err != nil {
		t.Fatal(err)
	}
	heap, err := cstdlib.WindowsMSVCSourceForSymbols(symbols[:1])
	if err != nil || !strings.HasPrefix(storage, heap) {
		t.Fatalf("private storage heap boundary: %v", err)
	}
	storage = sdk + strings.TrimPrefix(storage, heap)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalar+vectorStorageAllocator+storage+vectorStorageProbe+probe)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsHost(artifact)
			if err != nil || outcome.ExitCode != expectedExit || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("native: %+v %v", outcome, err)
			}
		})
	}
}

const vectorStorageProgram = `module main;
@link_name("fairyc_fixture_vector_initialize") unsafe extern C fn u32 initialize(ptr mut void storage, usize size, usize alignment);
@link_name("fairyc_fixture_vector_reserve") unsafe extern C fn u32 reserve(ptr mut void storage, usize capacity);
@link_name("fairyc_fixture_vector_seed") unsafe extern C fn i32 seed(ptr mut void storage);
@link_name("fairyc_fixture_vector_failure") unsafe extern C fn i32 failure(ptr mut void storage);
@link_name("fairyc_fixture_vector_grown") unsafe extern C fn i32 grown(ptr mut void storage);
@link_name("fairyc_fixture_vector_drop") unsafe extern C fn u32 release(ptr mut void storage);
@link_name("fairyc_fixture_vector_finished") unsafe extern C fn i32 finished();
resource Storage<T> {
    ptr mut void data; usize bytes; usize alignment; usize length; usize element_size; usize element_alignment;
    cleanup { let u32 status = unsafe { release(ref mut this as ptr mut void) }; }
}
fn Storage<i32> empty() {
    return Storage<i32> { .data = null, .bytes = 0, .alignment = 0, .length = 0, .element_size = 0, .element_alignment = 0 };
}
fn i32 exercise() {
    var Storage<i32> storage = empty();
    assert(unsafe { initialize(ref mut storage as ptr mut void, 4, 4) } == 0);
    assert(unsafe { reserve(ref mut storage as ptr mut void, 0) } == 0);
    assert(unsafe { reserve(ref mut storage as ptr mut void, 2) } == 0);
    assert(unsafe { seed(ref mut storage as ptr mut void) } == 1);
    assert(unsafe { failure(ref mut storage as ptr mut void) } == 1);
    assert(unsafe { reserve(ref mut storage as ptr mut void, 9) } == 0);
    assert(storage.length == 1 && storage.bytes == 64 && storage.element_size == 4 && storage.element_alignment == 4);
    assert(unsafe { grown(ref mut storage as ptr mut void) } == 1);
    let Storage<i32> moved = storage;
    return 41;
}
fn i32 main() { return exercise() + unsafe { finished() }; }
`

const vectorStorageAllocator = `
#include <stdlib.h>
#include <string.h>
static size_t vector_live = 0;
static size_t vector_allocations = 0, vector_frees = 0, vector_failures = 0;
static int vector_fail = 0;
static int vector_drop_calls = 0;
static uint32_t vector_drop_status = 99;
static void *vector_malloc(size_t size) {
    if (vector_fail) { vector_fail = 0; vector_failures++; return NULL; }
    void *data = malloc(size);
    if (data != NULL) { vector_live++; vector_allocations++; memset(data, 0xa5, size); }
    return data;
}
static void vector_free(void *data) { if (data != NULL) { vector_live--; vector_frees++; free(data); } }
#define FY_STD_SYSTEM_HEAP_MALLOC_V0(size) vector_malloc(size)
#define FY_STD_SYSTEM_HEAP_FREE_V0(data) vector_free(data)
`

const vectorStorageProbe = `
_Static_assert(sizeof(struct fy_private_vector_storage_v0) == 48, "Windows descriptor size");
_Static_assert(offsetof(struct fy_private_vector_storage_v0, length) == 24, "Windows length offset");
_Static_assert(sizeof(int32_t) == 4 && _Alignof(int32_t) == 4, "fixture i32 layout");
uint32_t fairyc_fixture_vector_initialize(void *storage, size_t size, size_t alignment) {
    return fy_private_vector_initialize_v0(storage, size, alignment);
}
uint32_t fairyc_fixture_vector_reserve(void *storage, size_t capacity) {
    return fy_private_vector_reserve_v0(storage, capacity);
}
int32_t fairyc_fixture_vector_seed(void *storage) {
    struct fy_private_vector_storage_v0 value;
    memcpy(&value, storage, sizeof value);
    if (value.block.size != 32 || value.length != 0 || vector_live != 1) return 0;
    int32_t payload = 41;
    memset(value.block.data, 0x5a, value.block.size);
    memcpy(value.block.data, &payload, sizeof payload);
    value.length = 1;
    memcpy(storage, &value, sizeof value);
    return 1;
}
int32_t fairyc_fixture_vector_failure(void *storage) {
    unsigned char before[sizeof(struct fy_private_vector_storage_v0)];
    memcpy(before, storage, sizeof before);
    vector_fail = 1;
    if (fy_private_vector_reserve_v0(storage, 9) != FY_STD_ALLOC_OUT_OF_MEMORY_V0 ||
        memcmp(before, storage, sizeof before) != 0 || vector_live != 1) return 0;
    if (fy_private_vector_reserve_v0(storage, SIZE_MAX) != FY_STD_ALLOC_SIZE_OVERFLOW_V0 ||
        memcmp(before, storage, sizeof before) != 0 || vector_live != 1) return 0;
    if (fy_private_vector_release_empty_v0(storage) != FY_STD_ALLOC_INVALID_ARGUMENT_V0 ||
        memcmp(before, storage, sizeof before) != 0 || vector_live != 1) return 0;
    return 1;
}
int32_t fairyc_fixture_vector_grown(void *storage) {
    struct fy_private_vector_storage_v0 value;
    memcpy(&value, storage, sizeof value);
    int32_t payload = 0;
    memcpy(&payload, value.block.data, sizeof payload);
    if (payload != 41 || value.block.size != 64 || value.length != 1 || vector_live != 1) return 0;
    for (size_t i = sizeof payload; i < value.block.size; i++) {
        if (((unsigned char *)value.block.data)[i] != 0xa5) return 0;
    }
    /* This fixture has a copy scalar, so discarding its only element has no destructor. */
    value.length = 0;
    memcpy(storage, &value, sizeof value);
    return 1;
}
uint32_t fairyc_fixture_vector_drop(void *storage) {
    vector_drop_calls++;
    vector_drop_status = fy_private_vector_release_empty_v0(storage);
    return vector_drop_status;
}
int32_t fairyc_fixture_vector_finished(void) {
    if (vector_live != 0 || vector_drop_calls != 1 || vector_drop_status != 0) return 0;
    struct fy_private_vector_storage_v0 empty = {{NULL, 0, 0}, 0, 0, 0};
    if (fy_private_vector_initialize_v0(&empty, 0, 4) != FY_STD_ALLOC_INVALID_ARGUMENT_V0 ||
        fy_private_vector_initialize_v0(&empty, 4, 0) != FY_STD_ALLOC_UNSUPPORTED_ALIGNMENT_V0 ||
        fy_private_vector_initialize_v0(&empty, 3, 4) != FY_STD_ALLOC_INVALID_ARGUMENT_V0 ||
        empty.element_size != 0 || empty.element_alignment != 0) return 0;
    if (fy_private_vector_initialize_v0(&empty, 4, 4) != 0 ||
        fy_private_vector_initialize_v0(&empty, 4, 4) != FY_STD_ALLOC_INVALID_ARGUMENT_V0) return 0;
    vector_fail = 1;
    if (fy_private_vector_reserve_v0(&empty, 0) != 0 || vector_fail != 1 ||
        fy_private_vector_reserve_v0(&empty, 1) != FY_STD_ALLOC_OUT_OF_MEMORY_V0 ||
        empty.block.data != NULL || empty.block.size != 0 || empty.length != 0 || vector_live != 0) return 0;
    if (fy_private_vector_release_empty_v0(&empty) != 0 || empty.element_size != 0 || empty.element_alignment != 0) return 0;
    return 1;
}
`
