#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

static LARGE_INTEGER started;
static LARGE_INTEGER frequency;
static uint32_t rounds;

uint32_t fc_budget_seed(void) {
    unsigned seed_input, rounds_input;
    if (scanf_s("%u %u", &seed_input, &rounds_input) != 2) abort();
    rounds = (uint32_t)rounds_input;
    return (uint32_t)seed_input;
}
uint32_t fc_budget_rounds(void) { return rounds; }
void fc_budget_begin(void) {
    if (!QueryPerformanceFrequency(&frequency) || !QueryPerformanceCounter(&started)) abort();
}
void fc_budget_finish(uint32_t value) {
    LARGE_INTEGER ended;
    if (!QueryPerformanceCounter(&ended) || ended.QuadPart < started.QuadPart || frequency.QuadPart <= 0) abort();
    printf("%u %llu %llu\n", (unsigned)value,
        (unsigned long long)(ended.QuadPart - started.QuadPart),
        (unsigned long long)frequency.QuadPart);
}
