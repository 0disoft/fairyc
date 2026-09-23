#include <stdint.h>
#include <stdlib.h>

extern uint32_t fc_budget_seed(void);
extern uint32_t fc_budget_rounds(void);
extern void fc_budget_begin(void);
extern void fc_budget_finish(uint32_t value);

/* For admitted inputs *slot < 65521 and i < 50000000. Multiplication,
   addition and increment fit uint32_t; divisors are nonzero constants. */
static void step(uint32_t *slot, uint32_t i) {
    *slot = (*slot * 33u + i % 251u) % 65521u;
}

static uint32_t checksum_through_ref(uint32_t seed, uint32_t rounds) {
    uint32_t value = seed;
    for (uint32_t i = 0; i < rounds; i++) {
        step(&value, i);
    }
    return value;
}

int main(void) {
    uint32_t seed = fc_budget_seed();
    uint32_t rounds = fc_budget_rounds();
    if (!(seed < 65521 && rounds <= 50000000)) abort();
    fc_budget_begin();
    uint32_t result = checksum_through_ref(seed, rounds);
    fc_budget_finish(result);
    return 0;
}
