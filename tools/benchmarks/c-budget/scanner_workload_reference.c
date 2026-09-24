// Concatenate after scanner_reference.c with FC_SCANNER_LIBRARY defined.
// The first scan reports exact lexical observations; each round owns and
// releases a source copy, token storage, diagnostics, and cooked values.
static int fc_workload_round(const unsigned char *input, size_t length,
                             int report, uint64_t *work) {
    unsigned char *owned = (unsigned char *)malloc(length == 0 ? 1u : length);
    struct fc_token *tokens = (struct fc_token *)malloc((length + 1u) * sizeof(*tokens));
    struct fc_diagnostic *diagnostics =
        (struct fc_diagnostic *)malloc((length + 1u) * sizeof(*diagnostics));
    if (owned == NULL || tokens == NULL || diagnostics == NULL) {
        free(owned); free(tokens); free(diagnostics);
        return 3;
    }
    if (length != 0) memcpy(owned, input, length);
    struct fc_scanner scanner = {owned, length, 0, 0, 0, tokens, diagnostics};
    int status = fc_scan(&scanner);
    if (status == 0) {
        *work = (uint64_t)length + scanner.token_count + scanner.diagnostic_count;
        for (size_t i = 0; i < scanner.token_count; i++)
            *work += scanner.tokens[i].cooked_length;
        for (size_t i = 0; i < scanner.diagnostic_count; i++)
            *work += strlen(scanner.diagnostics[i].token);
        if (report) {
            qsort(scanner.diagnostics, scanner.diagnostic_count,
                  sizeof(*scanner.diagnostics), fc_compare_diagnostic);
            fc_print(&scanner);
        }
    }
    fc_release_cooked(&scanner);
    free(owned); free(tokens); free(diagnostics);
    return status;
}

#ifndef FC_SCANNER_WORKLOAD_LIBRARY
int main(void) {
#ifdef _WIN32
    if (_setmode(_fileno(stdin), _O_BINARY) == -1 ||
        _setmode(_fileno(stdout), _O_BINARY) == -1) return FC_BAD_INPUT;
#endif
    unsigned char header[8];
    if (fread(header, 1, sizeof(header), stdin) != sizeof(header)) return FC_BAD_INPUT;
    uint32_t rounds = (uint32_t)header[0] | ((uint32_t)header[1] << 8) |
                      ((uint32_t)header[2] << 16) | ((uint32_t)header[3] << 24);
    uint32_t length = (uint32_t)header[4] | ((uint32_t)header[5] << 8) |
                      ((uint32_t)header[6] << 16) | ((uint32_t)header[7] << 24);
    if (rounds == 0 || rounds > 250000 || length > FC_MAX_SOURCE) return FC_BAD_INPUT;
    unsigned char *input = (unsigned char *)malloc(length == 0 ? 1u : (size_t)length);
    if (input == NULL) return 3;
    if (fread(input, 1, length, stdin) != length || getchar() != EOF) {
        free(input);
        return FC_BAD_INPUT;
    }
    uint64_t total = 0;
    for (uint32_t i = 0; i < rounds; i++) {
        uint64_t work = 0;
        int status = fc_workload_round(input, length, i == 0, &work);
        if (status != 0) { free(input); return status; }
        total += work;
    }
    free(input);
    printf("S %llu\n", (unsigned long long)total);
    return 0;
}
#endif
