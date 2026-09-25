#include <stddef.h>
#include <stdint.h>
#include <string.h>

// Safe callers supply initialized views; bounds are checked before pointer arithmetic.
uint32_t fy_std_bytes_match_at_v0(const uint8_t *input, size_t input_length,
                                  size_t position, const uint8_t *word, size_t word_length) {
    if (position > input_length || word_length > input_length - position) return UINT32_C(0);
    if (word_length == 0) return UINT32_C(1);
    if (input == NULL || word == NULL) return UINT32_C(0);
    return memcmp(input + position, word, word_length) == 0 ? UINT32_C(1) : UINT32_C(0);
}
