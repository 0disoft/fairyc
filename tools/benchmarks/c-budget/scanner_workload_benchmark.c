// Concatenate after scanner_reference.c, scanner_workload_reference.c and
// support.c with both FC_SCANNER_LIBRARY macros defined.
int main(void) {
#ifdef _WIN32
    if (_setmode(_fileno(stdin), _O_BINARY) == -1 ||
        _setmode(_fileno(stdout), _O_BINARY) == -1) return FC_BAD_INPUT;
#endif
    static const unsigned char sample[] = "let n=12345; /*open";
    uint64_t reported = 0;
    int status = fc_workload_round(sample, sizeof(sample) - 1u, 1, &reported);
    if (status != 0) return status;
    fflush(stdout);

    uint32_t seed = fc_budget_seed();
    uint32_t repeat_count = fc_budget_rounds();
    if (seed != 12345u || repeat_count == 0 || repeat_count > 250000u) return FC_BAD_INPUT;
    fc_budget_begin();
    uint64_t total = 0;
    for (uint32_t i = 0; i < repeat_count; i++) {
        uint64_t work = 0;
        status = fc_workload_round(sample, sizeof(sample) - 1u, 0, &work);
        if (status != 0) return status;
        total += work;
    }
    fc_budget_finish(repeat_count);
    printf("%llu\n", (unsigned long long)total);
    return 0;
}
