//go:build ignore

/* Private amd64 monotonic milliseconds, including explicit observation errors. */
#ifndef _POSIX_C_SOURCE
#define _POSIX_C_SOURCE 200809L
#endif
#include <errno.h>
#include <stdint.h>
#include <stddef.h>
#include <time.h>
_Static_assert(sizeof(size_t) == 8, "private Linux clock pointer width");
uint32_t fairyc_private_linux_monotonic_ms_v1(size_t *output) {
    if (output == NULL) { return EINVAL; }
    *output = 0;
    struct timespec now;
    if (clock_gettime(CLOCK_MONOTONIC, &now) != 0) { return (uint32_t)errno; }
    if (now.tv_sec < 0 || now.tv_nsec < 0 || now.tv_nsec >= 1000000000L) { return EOVERFLOW; }
    size_t fraction = (size_t)now.tv_nsec / 1000000u;
    if ((uint64_t)now.tv_sec > (SIZE_MAX - fraction) / 1000u) { return EOVERFLOW; }
    *output = (size_t)now.tv_sec * 1000u + fraction;
    return 0;
}
