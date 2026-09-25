// Concatenate after scanner_reference.c with FC_SCANNER_LIBRARY defined.
// The first scan reports exact lexical observations; each round owns and
// releases a source copy, token storage, diagnostics, and cooked values.
#ifdef FC_BUDGET_FAIRY_REPORT
#ifndef FC_CANDIDATE_DIAGNOSTIC_ENTRY_LAYOUT
#error FC_BUDGET_FAIRY_REPORT requires the candidate diagnostic layout
#endif
static void fc_report_number(size_t value) {
    if (printf("%zu ", value) < 0) abort();
}

static void fc_report_bytes(const unsigned char *bytes, size_t length) {
    fc_report_number(length);
    if (length != 0 && bytes == NULL) abort();
    for (size_t i = 0; i < length; i++) fc_report_number(bytes[i]);
}

static void fc_report_fairy(const struct fc_scanner *s) {
    size_t position = 0;
    for (size_t i = 0; i < s->token_count; i++) {
        const struct fc_token *token = &s->tokens[i];
        if (token->start != position || token->end < token->start ||
            token->end > s->length) abort();
        fc_report_number(token->kind);
        fc_report_number(token->trivia);
        fc_report_number(token->start);
        fc_report_number(token->end);
        const unsigned char *payload = token->cooked;
        size_t length = token->cooked_length;
        size_t width = token->end - token->start;
        if (token->kind == FC_KEYWORD &&
            ((width == 4 && (memcmp(s->source + token->start, "true", 4) == 0 ||
                             memcmp(s->source + token->start, "null", 4) == 0)) ||
             (width == 5 && memcmp(s->source + token->start, "false", 5) == 0))) {
            payload = s->source + token->start;
            length = width;
        }
        fc_report_bytes(payload, length);
        position = token->end;
    }
    if (position != s->length) abort();
    fc_report_number(s->diagnostic_count);
    for (size_t i = 0; i < s->diagnostic_count; i++) {
        const struct fc_diagnostic *d = &s->diagnostics[i];
        const char *code = fc_diagnostic_code_text(d);
        const char *repair = d->exit_repair ? "exit" : "none";
        fc_report_bytes((const unsigned char *)code, strlen(code));
        fc_report_bytes((const unsigned char *)repair, strlen(repair));
        fc_report_bytes(fc_diagnostic_token_data(d), fc_diagnostic_token_length(d));
        fc_report_number(d->start);
        fc_report_number(d->end);
    }
}
#endif

static int fc_workload_round(const unsigned char *input, size_t length,
                             int report, uint64_t *work) {
    unsigned char *owned = (unsigned char *)malloc(length == 0 ? 1u : length);
    struct fc_token *tokens = NULL;
#ifndef FC_CANDIDATE_GEOMETRIC_TOKENS
    tokens = (struct fc_token *)malloc((length + 1u) * sizeof(*tokens));
#endif
    struct fc_diagnostic *diagnostics = NULL;
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    diagnostics = (struct fc_diagnostic *)malloc((length + 1u) * sizeof(*diagnostics));
#endif
    if (owned == NULL
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
        || diagnostics == NULL
#endif
#ifndef FC_CANDIDATE_GEOMETRIC_TOKENS
        || tokens == NULL
#endif
    ) {
        free(owned); free(tokens); free(diagnostics);
        return 3;
    }
    if (length != 0) memcpy(owned, input, length);
    struct fc_scanner scanner = {owned, length, 0, 0, 0, tokens, diagnostics};
#ifdef FC_CANDIDATE_WORK_LIMIT
    scanner.work_limit = FC_CANDIDATE_WORK;
#endif
#ifdef FC_CANDIDATE_TOKEN_LIMIT
    scanner.token_limit = FC_CANDIDATE_TOKENS;
#endif
    int status = fc_scan(&scanner);
    tokens = scanner.tokens;
    diagnostics = scanner.diagnostics;
    if (status == 0) {
        *work = (uint64_t)length + scanner.token_count + scanner.diagnostic_count;
        for (size_t i = 0; i < scanner.token_count; i++)
            *work += scanner.tokens[i].cooked_length;
        for (size_t i = 0; i < scanner.diagnostic_count; i++)
            *work += fc_diagnostic_token_length(&scanner.diagnostics[i]);
        if (report) {
            qsort(scanner.diagnostics, scanner.diagnostic_count,
                  sizeof(*scanner.diagnostics), fc_compare_diagnostic);
#ifdef FC_BUDGET_FAIRY_REPORT
            fc_report_fairy(&scanner);
#else
            fc_print(&scanner);
#endif
        }
    }
    fc_release_cooked(&scanner);
    fc_release_diagnostics(&scanner);
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
