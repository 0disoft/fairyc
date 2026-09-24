#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>

extern uint32_t fc_budget_seed(void);
extern uint32_t fc_budget_rounds(void);
extern void fc_budget_begin(void);
extern void fc_budget_finish(uint32_t value);

struct checked_vector_u32 {
    uint32_t *data;
    size_t bytes;
    size_t alignment;
    size_t length;
    size_t element_size;
    size_t element_alignment;
};

static const uint32_t *checked_at(const struct checked_vector_u32 *values, size_t index) {
    if (values->data == NULL || values->alignment != _Alignof(uint32_t) ||
        values->element_size != sizeof(uint32_t) ||
        values->element_alignment != _Alignof(uint32_t) ||
        values->bytes % values->element_size != 0 ||
        values->length > values->bytes / values->element_size ||
        index >= values->length) abort();
    return &values->data[index];
}

static void checksum_through_vector(uint32_t seed, uint32_t rounds) {
    struct checked_vector_u32 values = {0};
    values.data = malloc(8u * sizeof(uint32_t));
    if (values.data == NULL) abort();
    values.bytes = 8u * sizeof(uint32_t);
    values.alignment = _Alignof(uint32_t);
    values.length = 4u;
    values.element_size = sizeof(uint32_t);
    values.element_alignment = _Alignof(uint32_t);
    values.data[0] = 7u;
    values.data[1] = 11u;
    values.data[2] = 13u;
    values.data[3] = 17u;
    uint32_t value = seed;
    fc_budget_begin();
    for (uint32_t i = 0; i < rounds; i++) {
        for (size_t index = 0; index < values.length; index++) {
            /* The admitted seed and rounds keep this arithmetic below UINT32_MAX. */
            value = (value * 33u + *checked_at(&values, index) + i % 251u) % 65521u;
        }
    }
    fc_budget_finish(value);
    free(values.data);
}

int main(void) {
    uint32_t seed = fc_budget_seed();
    uint32_t rounds = fc_budget_rounds();
    if (!(seed < 65521u && rounds <= 50000000u)) abort();
    checksum_through_vector(seed, rounds);
    return 0;
}
