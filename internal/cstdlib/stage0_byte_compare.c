#include <stddef.h>
#include <stdint.h>
#include <string.h>

// Callers supply two initialized byte ranges of exactly length bytes.
uint32_t fy_std_bytes_equal_v0(const uint8_t *left, const uint8_t *right, size_t length) {
    if (length == 0) return UINT32_C(1);
    if (left == NULL || right == NULL) return UINT32_C(0);
    return memcmp(left, right, length) == 0 ? UINT32_C(1) : UINT32_C(0);
}
