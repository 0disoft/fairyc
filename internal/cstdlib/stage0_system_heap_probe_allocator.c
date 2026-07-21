#include <stddef.h>
#include <stdlib.h>

static int fy_std_system_heap_probe_failure_armed_v0 = 0;
static size_t fy_std_system_heap_probe_successes_before_failure_v0 = 0;
static size_t fy_std_system_heap_probe_live_v0 = 0;

void fy_std_system_heap_probe_fail_after_allocations_v0(size_t successful_allocations) {
    fy_std_system_heap_probe_failure_armed_v0 = 1;
    fy_std_system_heap_probe_successes_before_failure_v0 = successful_allocations;
}

size_t fy_std_system_heap_probe_live_allocations_v0(void) {
    return fy_std_system_heap_probe_live_v0;
}

static void *fy_std_system_heap_probe_malloc_v0(size_t size) {
    if (fy_std_system_heap_probe_failure_armed_v0 &&
        fy_std_system_heap_probe_successes_before_failure_v0 == 0) {
        fy_std_system_heap_probe_failure_armed_v0 = 0;
        return NULL;
    }
    void *data = malloc(size);
    if (data != NULL) {
        fy_std_system_heap_probe_live_v0++;
        if (fy_std_system_heap_probe_failure_armed_v0) {
            fy_std_system_heap_probe_successes_before_failure_v0--;
        }
    }
    return data;
}

static void fy_std_system_heap_probe_free_v0(void *data) {
    if (data != NULL && fy_std_system_heap_probe_live_v0 != 0) {
        fy_std_system_heap_probe_live_v0--;
    }
    free(data);
}

#define FY_STD_SYSTEM_HEAP_MALLOC_V0(size) fy_std_system_heap_probe_malloc_v0(size)
#define FY_STD_SYSTEM_HEAP_FREE_V0(data) fy_std_system_heap_probe_free_v0(data)
