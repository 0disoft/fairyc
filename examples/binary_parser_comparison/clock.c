#include <windows.h>
#include <stddef.h>

size_t parser_ticks(void) {
    LARGE_INTEGER value;
    return QueryPerformanceCounter(&value) ? (size_t)value.QuadPart : 0;
}

size_t parser_frequency(void) {
    LARGE_INTEGER value;
    return QueryPerformanceFrequency(&value) ? (size_t)value.QuadPart : 0;
}
