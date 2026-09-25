#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

static LARGE_INTEGER started;
static LARGE_INTEGER frequency;
static uint32_t rounds;
#ifdef FC_SCANNER_CPU_TIME
static uint64_t cpu_started;

static uint64_t fc_budget_cpu_time(void) {
    FILETIME created, exited, kernel, user;
    if (!GetThreadTimes(GetCurrentThread(), &created, &exited, &kernel, &user)) abort();
    uint64_t kernel_ticks = ((uint64_t)kernel.dwHighDateTime << 32) | kernel.dwLowDateTime;
    uint64_t user_ticks = ((uint64_t)user.dwHighDateTime << 32) | user.dwLowDateTime;
    if (UINT64_MAX - kernel_ticks < user_ticks) abort();
    return kernel_ticks + user_ticks;
}
#endif

uint32_t fc_budget_seed(void) {
    unsigned seed_input, rounds_input;
    if (scanf_s("%u %u", &seed_input, &rounds_input) != 2) abort();
    rounds = (uint32_t)rounds_input;
    return (uint32_t)seed_input;
}
uint32_t fc_budget_rounds(void) { return rounds; }
void fc_budget_begin(void) {
    if (!QueryPerformanceFrequency(&frequency)) abort();
#ifdef FC_SCANNER_CPU_TIME
    cpu_started = fc_budget_cpu_time();
#endif
    if (!QueryPerformanceCounter(&started)) abort();
}
void fc_budget_finish(uint32_t value) {
    LARGE_INTEGER ended;
    if (!QueryPerformanceCounter(&ended) || ended.QuadPart < started.QuadPart || frequency.QuadPart <= 0) abort();
#ifdef FC_SCANNER_CPU_TIME
    uint64_t cpu_ended = fc_budget_cpu_time();
    if (cpu_ended < cpu_started) abort();
    printf("%u %llu %llu %llu\n", (unsigned)value,
        (unsigned long long)(ended.QuadPart - started.QuadPart),
        (unsigned long long)frequency.QuadPart,
        (unsigned long long)(cpu_ended - cpu_started));
#else
    printf("%u %llu %llu\n", (unsigned)value,
        (unsigned long long)(ended.QuadPart - started.QuadPart),
        (unsigned long long)frequency.QuadPart);
#endif
}
