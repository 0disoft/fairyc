#include <stdint.h>
#include <stdlib.h>

extern uint32_t fc_budget_seed(void);
extern uint32_t fc_budget_rounds(void);
extern void fc_budget_begin(void);
extern void fc_budget_finish(uint32_t value);

/* All additions and multiplications stay below uint32_t under the input guard. */
static uint32_t accumulate(uint32_t seed, uint32_t rounds) {
    const uint8_t data[4] = {7u, 11u, 13u, 17u};
    uint32_t value = seed;
    for (uint32_t i = 0; i < rounds; i++) {
        for (uint32_t index = 0; index < 4u; index++) {
            value = (value * 33u + data[index] + i % 251u) % 65521u;
        }
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
