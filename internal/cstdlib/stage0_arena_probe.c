#include <stddef.h>
#include <stdint.h>
#include <string.h>

enum fy_std_alloc_status_v0 {
    FY_STD_ALLOC_OK_V0 = 0,
    FY_STD_ALLOC_OUT_OF_MEMORY_V0 = 1,
    FY_STD_ALLOC_SIZE_OVERFLOW_V0 = 2,
    FY_STD_ALLOC_UNSUPPORTED_ALIGNMENT_V0 = 3,
    FY_STD_ALLOC_INVALID_ARGUMENT_V0 = 4
};

struct fy_std_arena_node_v0;

struct fy_std_arena_v0 {
    struct fy_std_arena_node_v0 *head;
    uint64_t generation;
    uint8_t active_region;
};

struct fy_std_region_v0 {
    struct fy_std_arena_v0 *arena;
    struct fy_std_arena_node_v0 *mark;
    uint64_t generation;
    uint8_t active;
};

uint32_t fy_std_arena_initialize_v0(struct fy_std_arena_v0 *output);
uint32_t fy_std_region_begin_v0(struct fy_std_arena_v0 *arena, struct fy_std_region_v0 *output);
uint32_t fy_std_region_allocate_zeroed_v0(
    struct fy_std_region_v0 *region,
    size_t size,
    size_t alignment,
    void **output
);
uint32_t fy_std_region_copy_bytes_v0(
    struct fy_std_region_v0 *region,
    const void *source,
    size_t size,
    size_t alignment,
    void **output
);
uint32_t fy_std_region_reset_v0(struct fy_std_region_v0 *region);
uint32_t fy_std_arena_drop_v0(struct fy_std_arena_v0 *arena);
void fy_std_system_heap_probe_fail_after_allocations_v0(size_t successful_allocations);
size_t fy_std_system_heap_probe_live_allocations_v0(void);

int main(void) {
    struct fy_std_arena_v0 arena = {NULL, 0, 0};
    struct fy_std_region_v0 region = {NULL, NULL, 0, 0};
    if (fy_std_arena_drop_v0(&arena) != FY_STD_ALLOC_INVALID_ARGUMENT_V0) {
        return 1;
    }
    if (fy_std_arena_initialize_v0(&arena) != FY_STD_ALLOC_OK_V0 ||
        arena.head != NULL || arena.generation != 1 || arena.active_region != 0) {
        return 2;
    }
    if (fy_std_region_begin_v0(&arena, &region) != FY_STD_ALLOC_OK_V0 ||
        arena.active_region != 1 || region.active != 1 || region.generation != 1) {
        return 3;
    }
    struct fy_std_region_v0 second = {NULL, NULL, 0, 0};
    if (fy_std_region_begin_v0(&arena, &second) != FY_STD_ALLOC_INVALID_ARGUMENT_V0 ||
        second.active != 0) {
        return 4;
    }
    uint8_t *zeroed = NULL;
    if (fy_std_region_allocate_zeroed_v0(&region, 8, 8, (void **)&zeroed) != FY_STD_ALLOC_OK_V0 ||
        zeroed == NULL || ((uintptr_t)zeroed % 8) != 0 ||
        fy_std_system_heap_probe_live_allocations_v0() != 2) {
        return 5;
    }
    for (size_t index = 0; index < 8; ++index) {
        if (zeroed[index] != 0) {
            return 6;
        }
        zeroed[index] = (uint8_t)(index + 1);
    }
    uint8_t *copied = NULL;
    if (fy_std_region_copy_bytes_v0(&region, zeroed, 8, 8, (void **)&copied) != FY_STD_ALLOC_OK_V0 ||
        copied == NULL || copied == zeroed || memcmp(copied, zeroed, 8) != 0 ||
        fy_std_system_heap_probe_live_allocations_v0() != 4) {
        return 7;
    }
    struct fy_std_arena_node_v0 *before_failure = arena.head;
    void *invalid = (void *)(uintptr_t)1;
    if (fy_std_region_allocate_zeroed_v0(&region, 4, 3, &invalid) != FY_STD_ALLOC_UNSUPPORTED_ALIGNMENT_V0 ||
        arena.head != before_failure || invalid != (void *)(uintptr_t)1) {
        return 8;
    }
    void *empty = (void *)(uintptr_t)1;
    if (fy_std_region_allocate_zeroed_v0(&region, 0, 1, &empty) != FY_STD_ALLOC_OK_V0 ||
        empty != NULL || arena.head != before_failure) {
        return 9;
    }
    if (fy_std_arena_drop_v0(&arena) != FY_STD_ALLOC_INVALID_ARGUMENT_V0 || arena.active_region != 1 ||
        fy_std_system_heap_probe_live_allocations_v0() != 4) {
        return 10;
    }
    if (fy_std_region_reset_v0(&region) != FY_STD_ALLOC_OK_V0 ||
        arena.head != NULL || arena.active_region != 0 || arena.generation != 2 || region.active != 0 ||
        fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 11;
    }
    if (fy_std_region_allocate_zeroed_v0(&region, 1, 1, &empty) != FY_STD_ALLOC_INVALID_ARGUMENT_V0) {
        return 12;
    }
    if (fy_std_arena_drop_v0(&arena) != FY_STD_ALLOC_OK_V0 ||
        arena.head != NULL || arena.active_region != 0 || arena.generation != 3 ||
        fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 13;
    }

    if (fy_std_arena_initialize_v0(&arena) != FY_STD_ALLOC_OK_V0 ||
        fy_std_region_begin_v0(&arena, &region) != FY_STD_ALLOC_OK_V0) {
        return 14;
    }
    before_failure = arena.head;
    void *failed_output = (void *)(uintptr_t)1;
    fy_std_system_heap_probe_fail_after_allocations_v0(0);
    if (fy_std_region_allocate_zeroed_v0(&region, 8, 8, &failed_output) != FY_STD_ALLOC_OUT_OF_MEMORY_V0 ||
        arena.head != before_failure || failed_output != (void *)(uintptr_t)1 ||
        fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 15;
    }
    fy_std_system_heap_probe_fail_after_allocations_v0(1);
    if (fy_std_region_allocate_zeroed_v0(&region, 8, 8, &failed_output) != FY_STD_ALLOC_OUT_OF_MEMORY_V0 ||
        arena.head != before_failure || failed_output != (void *)(uintptr_t)1 ||
        fy_std_system_heap_probe_live_allocations_v0() != 0) {
        return 16;
    }
    if (fy_std_region_allocate_zeroed_v0(&region, 8, 8, (void **)&zeroed) != FY_STD_ALLOC_OK_V0 ||
        zeroed == NULL || arena.head == before_failure || fy_std_system_heap_probe_live_allocations_v0() != 2) {
        return 17;
    }
    for (size_t index = 0; index < 8; ++index) {
        if (zeroed[index] != 0) {
            return 18;
        }
    }
    if (fy_std_region_reset_v0(&region) != FY_STD_ALLOC_OK_V0 ||
        fy_std_system_heap_probe_live_allocations_v0() != 0 ||
        fy_std_arena_drop_v0(&arena) != FY_STD_ALLOC_OK_V0) {
        return 19;
    }
    return 0;
}
