#include <stdint.h>
#include <stdlib.h>

extern uint32_t fc_budget_seed(void);
extern uint32_t fc_budget_rounds(void);
extern void fc_budget_begin(void);
extern void fc_budget_finish(uint32_t value);

/* Input guard bounds i+seed below 50065521, value+kind below 65525,
   and i+1 at most 50000000. No admitted operation overflows. */
static uint32_t classify(uint32_t seed, uint32_t rounds) {
    uint32_t value = 0;
    for (uint32_t i = 0; i < rounds; i++) {
        uint32_t byte = (i + seed) % 128;
        uint32_t kind = 4;
        if (byte == 32 || byte == 10) kind = 1;
        else if (byte >= 48 && byte <= 57) kind = 2;
        else if (byte >= 65 && byte <= 90) kind = 3;
        value = (value + kind) % 65521;
    }
    return value;
}

int main(void) {
    uint32_t seed = fc_budget_seed();
    uint32_t rounds = fc_budget_rounds();
    if (!(seed < 65521 && rounds <= 50000000)) abort();
    fc_budget_begin();
    uint32_t result = classify(seed, rounds);
    fc_budget_finish(result);
    return 0;
}
