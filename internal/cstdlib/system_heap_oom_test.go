package cstdlib

import (
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

// Uses the existing allocator seam with the real target-selected heap implementation.
const systemHeapOOMProbe = `
#ifdef NDEBUG
#undef NDEBUG
#endif
#include <assert.h>

static void fy_heap_assert_empty(const struct fy_std_system_heap_block_v0 *block) {
    assert(block->data == NULL && block->size == 0 && block->alignment == 0);
    assert(fy_std_system_heap_probe_live_allocations_v0() == 0);
}

static void fy_heap_check_resize(struct fy_std_system_heap_block_v0 *block,
    size_t size, size_t alignment) {
    uint8_t original[32];
    assert(block->size <= sizeof original);
    for (size_t i=0; i<block->size; i++) ((uint8_t *)block->data)[i]=(uint8_t)(i^0x5a);
    memcpy(original,block->data,block->size);
    const struct fy_std_system_heap_block_v0 before=*block;
    fy_std_system_heap_probe_fail_after_allocations_v0(0);
    assert(fy_std_system_heap_resize_v0(block,size,alignment) == FY_STD_ALLOC_OUT_OF_MEMORY_V0);
    assert(block->data == before.data && block->size == before.size && block->alignment == before.alignment);
    assert(memcmp(block->data,original,before.size) == 0);
    assert(fy_std_system_heap_probe_live_allocations_v0() == 1);
    /* The failure is one-shot: the same owner can recover without a reset. */
    assert(fy_std_system_heap_resize_v0(block,size,alignment) == FY_STD_ALLOC_OK_V0);
    assert(block->data != NULL && block->size == size && block->alignment == alignment);
    assert((uintptr_t)block->data%alignment == 0);
    assert(memcmp(block->data,original,size<before.size?size:before.size) == 0);
    assert(fy_std_system_heap_probe_live_allocations_v0() == 1);
}

int main(void) {
    struct fy_std_system_heap_block_v0 block={NULL,0,0};
    fy_std_system_heap_probe_fail_after_allocations_v0(0);
    assert(fy_std_system_heap_allocate_v0(0,1,&block) == FY_STD_ALLOC_OK_V0);
    fy_heap_assert_empty(&block);
    /* Empty allocation must not consume the armed platform-allocation failure. */
    assert(fy_std_system_heap_allocate_v0(8,1,&block) == FY_STD_ALLOC_OUT_OF_MEMORY_V0);
    fy_heap_assert_empty(&block);
    assert(fy_std_system_heap_allocate_v0(8,1,&block) == FY_STD_ALLOC_OK_V0);
    fy_heap_check_resize(&block,16,8);
    fy_heap_check_resize(&block,4,1);
    fy_heap_check_resize(&block,4,8);
    fy_std_system_heap_probe_fail_after_allocations_v0(0);
    assert(fy_std_system_heap_resize_v0(&block,0,1) == FY_STD_ALLOC_OK_V0);
    fy_heap_assert_empty(&block);
    /* Zero resize frees the prior owner without attempting a replacement. */
    assert(fy_std_system_heap_resize_v0(&block,4,1) == FY_STD_ALLOC_OUT_OF_MEMORY_V0);
    fy_heap_assert_empty(&block);
    assert(fy_std_system_heap_resize_v0(&block,4,1) == FY_STD_ALLOC_OK_V0);
    assert(fy_std_system_heap_probe_live_allocations_v0() == 1);
    assert(fy_std_system_heap_deallocate_v0(&block) == FY_STD_ALLOC_OK_V0);
    fy_heap_assert_empty(&block);
    return 0;
}
`

func TestSystemHeapOOMNative(t *testing.T) {
	if runtime.GOARCH != "amd64" || (runtime.GOOS != "linux" && runtime.GOOS != "windows") {
		t.Skip("requires Linux GCC/Clang or Windows MSVC on amd64")
	}
	compilers := []string{"gcc", "clang"}
	if runtime.GOOS == "windows" {
		compilers = []string{"msvc"}
	}
	for _, cc := range compilers {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(cc+"/"+profile.Name, func(t *testing.T) {
				symbols := []string{"fy_std_system_heap_allocate_v0"}
				var implementation string
				var err error
				if cc == "msvc" {
					implementation, err = WindowsMSVCSourceForSymbols(symbols)
				} else {
					implementation, err = SourceForSymbols(symbols)
				}
				if err != nil {
					t.Fatal(err)
				}
				implementation, err = systemHeapProbeImplementationSource("SystemHeap OOM", implementation)
				if err != nil {
					t.Fatal(err)
				}
				if cc == "msvc" {
					tc, err := cdriver.ResolveWindowsHostMSVC()
					if err != nil {
						t.Fatal(err)
					}
					artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, implementation+systemHeapOOMProbe)
					if err != nil {
						t.Fatal(err)
					}
					outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
					if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
						t.Fatalf("heap OOM: %+v, %v", outcome, err)
					}
				} else {
					tc, err := cdriver.Resolve(target.X8664LinuxGNU, cc)
					if err != nil {
						t.Fatal(err)
					}
					artifact, err := cdriver.Build(tc, profile, implementation+systemHeapOOMProbe, "typedef int fy_heap_oom_runtime;\n")
					if err != nil {
						t.Fatal(err)
					}
					outcome, err := cdriver.ExecuteLinuxTarget(artifact)
					if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapCategory != 0 {
						t.Fatalf("heap OOM: %+v, %v", outcome, err)
					}
				}
			})
		}
	}
}
