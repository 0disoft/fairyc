#include <windows.h>
#include <stddef.h>

// Linked only into a temporary profiling build, never the distributed utility.
size_t hash_profile_counter(void) {
    LARGE_INTEGER value;
    return QueryPerformanceCounter(&value) ? (size_t)value.QuadPart : 0;
}
size_t hash_profile_frequency(void) {
    LARGE_INTEGER value;
    return QueryPerformanceFrequency(&value) ? (size_t)value.QuadPart : 0;
}
