#include <stdint.h>
#include <stdlib.h>

extern uint32_t fc_budget_seed(void);
extern uint32_t fc_budget_rounds(void);
extern void fc_budget_begin(void);
extern void fc_budget_finish(uint32_t value);

/* The guarded seed and round count keep i+seed, value+delta, and i+1 in
   uint32_t. The success and failure paths match the Fairy Result switch. */
static uint32_t decode(uint32_t byte) {
    if (byte % 8u == 0u) return 3u;
    return byte + 1u;
}

static uint32_t accumulate(uint32_t seed, uint32_t rounds) {
    uint32_t value = 0;
    for (uint32_t i = 0; i < rounds; i++) {
        uint32_t byte = (i + seed) % 128u;
        value = (value + decode(byte)) % 65521u;
    }
    return value;
}

int main(void) {
    uint32_t seed = fc_budget_seed();
    uint32_t rounds = fc_budget_rounds();
    if (!(seed < 65521u && rounds <= 50000000u)) abort();
    fc_budget_begin();
    uint32_t result = accumulate(seed, rounds);
    fc_budget_finish(result);
    return 0;
}
