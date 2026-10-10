// Independent C11 lexical reference. The broad Go-oracle build leaves decimal
// expansions beyond 16 MiB unsupported. The candidate C budget build models
// numeric expansion work, but other work paths still need parity evidence.
#include <errno.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#ifdef _WIN32
#include <fcntl.h>
#include <io.h>
#endif

#ifdef FC_CANDIDATE_SOURCE_LIMIT
// Match candidate_limits::program_bytes for owned/joined scanner input.
enum { FC_MAX_SOURCE = 1 << 20, FC_CANDIDATE_SOURCE_SIZE = 4 };
#else
enum { FC_MAX_SOURCE = 32768 };
#endif
enum { FC_MAX_COOKED = 16 << 20, FC_UNSUPPORTED = 7, FC_BAD_INPUT = 8 };
#ifdef FC_CANDIDATE_DECIMAL_LIMIT
// The Fairy-written scanner materializes at most 16,384 decimal bytes.
// Keep the broader Go-oracle reference available without this build switch.
enum { FC_CANDIDATE_DECIMAL = 16384, FC_CANDIDATE_SIZE = 4 };
#endif
#ifdef FC_CANDIDATE_COOKED_LIMIT
enum { FC_CANDIDATE_COOKED = 65536, FC_CANDIDATE_COOKED_SIZE = 4 };
#endif
#ifdef FC_CANDIDATE_WORK_LIMIT
enum { FC_CANDIDATE_WORK = 16 << 20 };
#endif
#ifdef FC_CANDIDATE_DIAGNOSTIC_LIMIT
enum { FC_CANDIDATE_DIAGNOSTICS = 256 };
#endif
#ifdef FC_CANDIDATE_TOKEN_LIMIT
enum { FC_CANDIDATE_TOKENS = 1 << 20 };
#endif
#if defined(FC_CANDIDATE_DIAGNOSTIC_ENTRY_LAYOUT) && !defined(FC_CANDIDATE_DIAGNOSTIC_PAYLOAD)
#error FC_CANDIDATE_DIAGNOSTIC_ENTRY_LAYOUT requires owned diagnostic payloads
#endif
#if defined(FC_CANDIDATE_BYTE_LIMIT) && !defined(FC_CANDIDATE_WORK_LIMIT)
#error FC_CANDIDATE_BYTE_LIMIT requires the candidate terminal diagnostic path
#endif
enum { FC_INVALID, FC_EOF, FC_TRIVIA, FC_IDENTIFIER, FC_KEYWORD, FC_INTEGER,
       FC_FLOAT, FC_STRING, FC_BYTE_STRING, FC_C_STRING, FC_SYMBOL };
enum { FC_NO_TRIVIA, FC_PREAMBLE, FC_WHITESPACE, FC_LINE_COMMENT,
       FC_BLOCK_COMMENT, FC_DOC_LINE };

struct fc_token {
    unsigned kind, trivia;
    size_t start, end;
    const unsigned char *cooked;
    size_t cooked_length;
};
// Only the candidate budget lane uses the Fairy Entry/Buffer field shape.
// The broad Go-oracle reference retains its independently readable strings.
#ifdef FC_CANDIDATE_DIAGNOSTIC_ENTRY_LAYOUT
struct fc_diagnostic_buffer {
    unsigned char *data;
    size_t capacity, alignment, length;
};
struct fc_diagnostic {
    uint32_t code;
    size_t start, end;
    unsigned char exit_repair;
    struct fc_diagnostic_buffer token;
};
#else
struct fc_diagnostic {
    const char *code, *repair, *token;
    size_t start, end;
#ifdef FC_CANDIDATE_DIAGNOSTIC_PAYLOAD
    size_t token_length;
#endif
};
#endif
struct fc_scanner {
    const unsigned char *source;
    size_t length, position, token_count, diagnostic_count;
    struct fc_token *tokens;
    struct fc_diagnostic *diagnostics;
    int previous_revision;
#ifdef FC_CANDIDATE_GEOMETRIC_TOKENS
    size_t token_capacity;
    int token_allocation_failed;
#endif
#ifdef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    size_t diagnostic_capacity;
    int diagnostic_allocation_failed;
#endif
#ifdef FC_CANDIDATE_COOKED_LIMIT
    size_t cooked_total;
#endif
#ifdef FC_CANDIDATE_WORK_LIMIT
    size_t expansion_work, work_limit;
#endif
#ifdef FC_CANDIDATE_BYTE_LIMIT
    size_t byte_limit;
    int byte_limit_set;
#endif
#ifdef FC_CANDIDATE_DIAGNOSTIC_LIMIT
    size_t diagnostic_limit;
    int diagnostic_limit_set;
    int diagnostic_limit_exhausted;
#endif
#ifdef FC_CANDIDATE_TOKEN_LIMIT
    size_t token_limit;
    int token_limit_exhausted;
#endif
#ifdef FC_CANDIDATE_DIAGNOSTIC_PAYLOAD
    int diagnostic_payload_failed;
#endif
};

#ifdef FC_CANDIDATE_GEOMETRIC_TOKENS
// Match Vec<Token>'s empty, 8, 16, ... capacity path for the C budget lane.
// This mode requires an initially null token pointer owned by the scanner.
static int fc_reserve_token_slot(struct fc_scanner *s) {
    if (s->token_count > s->token_capacity) return 0;
    if (s->token_capacity > s->token_count) return s->tokens != NULL;
    size_t maximum = SIZE_MAX / sizeof(*s->tokens);
    if ((s->tokens != NULL && s->token_capacity == 0) ||
        s->token_capacity >= maximum) return 0;
    size_t next = s->token_capacity == 0 ? (maximum < 8u ? maximum : 8u) :
                  s->token_capacity > maximum / 2u ? maximum : s->token_capacity * 2u;
    if (next <= s->token_count) return 0;
    struct fc_token *replacement = (struct fc_token *)malloc(next * sizeof(*replacement));
    if (replacement == NULL) return 0;
    if (s->token_count != 0)
        memcpy(replacement, s->tokens, s->token_count * sizeof(*replacement));
    free(s->tokens);
    s->tokens = replacement;
    s->token_capacity = next;
    return 1;
}
#endif

#ifdef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
// The candidate C image follows Vec<Entry>'s empty, 8, 16, ... capacities.
// Diagnostic token payload ownership remains a separate parity question.
static int fc_reserve_diagnostic_slot(struct fc_scanner *s) {
    if (s->diagnostic_count > s->diagnostic_capacity) return 0;
    if (s->diagnostic_capacity > s->diagnostic_count)
        return s->diagnostics != NULL;
    size_t maximum = SIZE_MAX / sizeof(*s->diagnostics);
    if ((s->diagnostics != NULL && s->diagnostic_capacity == 0) ||
        s->diagnostic_capacity >= maximum) return 0;
    size_t next = s->diagnostic_capacity == 0 ? (maximum < 8u ? maximum : 8u) :
                  s->diagnostic_capacity > maximum / 2u ? maximum :
                  s->diagnostic_capacity * 2u;
    if (next <= s->diagnostic_count) return 0;
    struct fc_diagnostic *replacement =
        (struct fc_diagnostic *)malloc(next * sizeof(*replacement));
    if (replacement == NULL) return 0;
    if (s->diagnostic_count != 0)
        memcpy(replacement, s->diagnostics,
               s->diagnostic_count * sizeof(*replacement));
    free(s->diagnostics);
    s->diagnostics = replacement;
    s->diagnostic_capacity = next;
    return 1;
}
#endif

#ifdef FC_CANDIDATE_COOKED_LIMIT
static int fc_accept_cooked(struct fc_scanner *s, size_t length) {
    if (length > FC_CANDIDATE_COOKED - s->cooked_total) return 0;
    s->cooked_total += length;
    return 1;
}
#endif

static void fc_token_unlimited(struct fc_scanner *s, unsigned kind, unsigned trivia,
                               size_t start, size_t end, const unsigned char *cooked,
                               size_t cooked_length) {
#ifdef FC_CANDIDATE_GEOMETRIC_TOKENS
    if (s->token_allocation_failed || !fc_reserve_token_slot(s)) {
        s->token_allocation_failed = 1;
        if (kind == FC_INTEGER || kind == FC_FLOAT || kind == FC_STRING ||
            kind == FC_BYTE_STRING || kind == FC_C_STRING) free((void *)cooked);
        return;
    }
#endif
    s->tokens[s->token_count++] = (struct fc_token){kind, trivia, start, end,
                                                   cooked, cooked_length};
}

static void fc_token(struct fc_scanner *s, unsigned kind, unsigned trivia,
                     size_t start, size_t end, const unsigned char *cooked,
                     size_t cooked_length) {
#ifdef FC_CANDIDATE_TOKEN_LIMIT
    if (s->token_limit_exhausted || s->token_count >= s->token_limit) {
        if (kind == FC_INTEGER || kind == FC_FLOAT || kind == FC_STRING ||
            kind == FC_BYTE_STRING || kind == FC_C_STRING) free((void *)cooked);
        if (!s->token_limit_exhausted) {
            s->token_limit_exhausted = 1;
            if (start < end)
                fc_token_unlimited(s, FC_INVALID, FC_NO_TRIVIA, start, end, NULL, 0);
        }
        return;
    }
#endif
    fc_token_unlimited(s, kind, trivia, start, end, cooked, cooked_length);
}

#ifdef FC_CANDIDATE_DIAGNOSTIC_ENTRY_LAYOUT
static uint32_t fc_diagnostic_number(const char *code) {
    if (strcmp(code, "E0001") == 0) return 1u;
    if (strcmp(code, "E0004") == 0) return 4u;
    if (strcmp(code, "E0005") == 0) return 5u;
    if (strcmp(code, "E0006") == 0) return 6u;
    if (strcmp(code, "E0010") == 0) return 10u;
    abort();
}

static const char *fc_diagnostic_code_text(const struct fc_diagnostic *d) {
    switch (d->code) {
        case 1u: return "E0001";
        case 4u: return "E0004";
        case 5u: return "E0005";
        case 6u: return "E0006";
        case 10u: return "E0010";
        default: abort();
    }
}
#endif

static const unsigned char *fc_diagnostic_token_data(const struct fc_diagnostic *d) {
#ifdef FC_CANDIDATE_DIAGNOSTIC_ENTRY_LAYOUT
    return d->token.data;
#else
    return (const unsigned char *)d->token;
#endif
}

static void fc_diagnostic(struct fc_scanner *s, const char *code,
                          size_t start, size_t end, const char *repair,
                          const char *token) {
#ifdef FC_CANDIDATE_TOKEN_LIMIT
    if (s->token_limit_exhausted && strcmp(code, "E0010") != 0) return;
#endif
#ifdef FC_CANDIDATE_DIAGNOSTIC_LIMIT
    // The one terminal E0010 is outside the ordinary diagnostic budget.
    size_t limit = s->diagnostic_limit_set ? s->diagnostic_limit : FC_CANDIDATE_DIAGNOSTICS;
    if (s->diagnostic_count >= limit &&
        strcmp(code, "E0010") != 0) {
        s->diagnostic_limit_exhausted = 1;
        return;
    }
#endif
#ifdef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    if (s->diagnostic_allocation_failed) return;
#endif
#ifdef FC_CANDIDATE_DIAGNOSTIC_PAYLOAD
    size_t token_length = strlen(token);
    char *owned = NULL;
    if (token_length != 0) {
        owned = (char *)malloc(token_length);
        if (owned == NULL) {
            s->diagnostic_payload_failed = 1;
            return;
        }
        memcpy(owned, token, token_length);
        token = owned;
    }
#endif
#ifdef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    if (!fc_reserve_diagnostic_slot(s)) {
#ifdef FC_CANDIDATE_DIAGNOSTIC_PAYLOAD
        free(owned);
#endif
        s->diagnostic_allocation_failed = 1;
        return;
    }
#endif
#ifdef FC_CANDIDATE_DIAGNOSTIC_ENTRY_LAYOUT
    struct fc_diagnostic value = {0};
    value.code = fc_diagnostic_number(code);
    value.start = start;
    value.end = end;
    value.exit_repair = (unsigned char)(strcmp(repair, "exit") == 0);
    value.token = (struct fc_diagnostic_buffer){(unsigned char *)owned,
                                                token_length, 1u, token_length};
#else
    struct fc_diagnostic value = {code, repair, token, start, end};
#ifdef FC_CANDIDATE_DIAGNOSTIC_PAYLOAD
    value.token_length = token_length;
#endif
#endif
    s->diagnostics[s->diagnostic_count++] = value;
}

static size_t fc_diagnostic_token_length(const struct fc_diagnostic *diagnostic) {
#ifdef FC_CANDIDATE_DIAGNOSTIC_ENTRY_LAYOUT
    return diagnostic->token.length;
#elif defined(FC_CANDIDATE_DIAGNOSTIC_PAYLOAD)
    return diagnostic->token_length;
#else
    return strlen(diagnostic->token);
#endif
}

static int fc_match(const struct fc_scanner *s, size_t at, const char *text) {
    size_t length = strlen(text);
    return at <= s->length && length <= s->length - at &&
           memcmp(s->source + at, text, length) == 0;
}

static int fc_horizontal(unsigned char byte) {
    return byte == ' ' || byte == '\t';
}

static int fc_identifier_start(unsigned char byte) {
    return byte == '_' || (byte >= 'a' && byte <= 'z') ||
           (byte >= 'A' && byte <= 'Z');
}

static int fc_identifier_continue(unsigned char byte) {
    return fc_identifier_start(byte) || (byte >= '0' && byte <= '9');
}

static int fc_decimal_digit(unsigned char byte) {
    return byte >= '0' && byte <= '9';
}

static int fc_digit_value(unsigned char byte) {
    if (fc_decimal_digit(byte)) return byte - '0';
    if (byte >= 'a' && byte <= 'f') return byte - 'a' + 10;
    if (byte >= 'A' && byte <= 'F') return byte - 'A' + 10;
    return -1;
}

// Return 1 for a cooked integer, 0 for an invalid spelling and -1 on OOM.
// The decimal accumulator is deliberately independent of Go's math/big.
static int fc_cook_integer(const unsigned char *raw, size_t length,
                           unsigned char **cooked, size_t *cooked_length) {
    unsigned base = 10;
    size_t start = 0;
    if (length >= 2 && raw[0] == '0' && (raw[1] == 'x' || raw[1] == 'b')) {
        base = raw[1] == 'x' ? 16u : 2u;
        start = 2;
    }
    if (start == length) return 0;
    for (size_t i = start; i < length; i++) {
        int value = fc_digit_value(raw[i]);
        if (value >= 0 && (unsigned)value < base) continue;
        if (raw[i] != '_' || i == start || i + 1 >= length ||
            fc_digit_value(raw[i - 1]) < 0 ||
            (unsigned)fc_digit_value(raw[i - 1]) >= base ||
            fc_digit_value(raw[i + 1]) < 0 ||
            (unsigned)fc_digit_value(raw[i + 1]) >= base) return 0;
    }
    unsigned char *decimal = (unsigned char *)malloc(length * 2u + 1u);
    if (decimal == NULL) return -1;
    size_t used = 1;
    decimal[0] = 0;
    for (size_t i = start; i < length; i++) {
        if (raw[i] == '_') continue;
        unsigned carry = (unsigned)fc_digit_value(raw[i]);
        for (size_t j = 0; j < used; j++) {
            unsigned product = (unsigned)decimal[j] * base + carry;
            decimal[j] = (unsigned char)(product % 10u);
            carry = product / 10u;
        }
        while (carry != 0) {
            decimal[used++] = (unsigned char)(carry % 10u);
            carry /= 10u;
        }
    }
    unsigned char *text = (unsigned char *)malloc(used);
    if (text == NULL) { free(decimal); return -1; }
    for (size_t i = 0; i < used; i++) text[i] = (unsigned char)('0' + decimal[used - 1u - i]);
    free(decimal);
    *cooked = text;
    *cooked_length = used;
    return 1;
}

// Return 1 for cooked decimal, 0 for an invalid spelling, -1 on OOM,
// -2 for the broad reference's unsupported size, -3 for candidate size,
// and -4 for candidate work exhaustion before allocation.
static int fc_cook_float(struct fc_scanner *s, const unsigned char *raw, size_t length,
                          unsigned char **cooked, size_t *cooked_length) {
#ifndef FC_CANDIDATE_WORK_LIMIT
    (void)s;
#endif
    unsigned char *clean = (unsigned char *)malloc(length + 1u);
    if (clean == NULL) return -1;
    size_t clean_length = 0;
    for (size_t i = 0; i < length; i++) {
        if (raw[i] != '_') clean[clean_length++] = raw[i];
    }
    clean[clean_length] = 0;
    size_t exponent_at = clean_length;
    for (size_t i = 0; i < clean_length; i++) {
        if (clean[i] == 'e' || clean[i] == 'E') { exponent_at = i; break; }
    }
    long long exponent = 0;
    if (exponent_at < clean_length) {
        char *end = NULL;
        errno = 0;
        exponent = strtoll((const char *)clean + exponent_at + 1u, &end, 10);
        if (errno == ERANGE || end == (char *)clean + exponent_at + 1u || *end != 0) {
            free(clean); return 0;
        }
    }
    size_t integer_length = 0, fraction_length = 0;
    int dot = 0, valid = 1;
    for (size_t i = 0; i < exponent_at; i++) {
        if (clean[i] == '.') {
            if (dot) { valid = 0; break; }
            dot = 1;
        } else if (!fc_decimal_digit(clean[i])) {
            valid = 0; break;
        } else if (dot) fraction_length++;
        else integer_length++;
    }
    if (!valid || integer_length == 0 || (dot && fraction_length == 0) ||
        (!dot && exponent_at == clean_length)) {
        free(clean); return 0;
    }
    unsigned char *digits = (unsigned char *)malloc(integer_length + fraction_length + 1u);
    if (digits == NULL) { free(clean); return -1; }
    size_t digit_count = 0;
    for (size_t i = 0; i < exponent_at; i++) {
        if (clean[i] != '.') digits[digit_count++] = clean[i];
    }
    free(clean);
    size_t first = 0;
    while (first < digit_count && digits[first] == '0') first++;
    if (first == digit_count) {
        digits[0] = '0';
        digit_count = 1;
        first = 0;
    }
    size_t significant = digit_count - first;
    if (exponent < -(long long)FC_MAX_COOKED ||
        exponent > (long long)FC_MAX_COOKED) {
        free(digits);
#ifdef FC_CANDIDATE_WORK_LIMIT
        return -4;
#else
        return -2;
#endif
    }
    long long places = (long long)fraction_length - exponent;
    size_t output_length = places <= 0 ? significant + (size_t)-places :
                           places >= (long long)significant ? 2u + (size_t)places :
                           significant + 1u;
#ifdef FC_CANDIDATE_WORK_LIMIT
    // Numeric scanning charges one unit per consumed source byte. Other
    // scanner work paths need their own parity proof before full admission.
    size_t scanned = s->position;
    if (scanned > s->work_limit || s->expansion_work > s->work_limit - scanned ||
        output_length > s->work_limit - scanned - s->expansion_work) {
        free(digits); return -4;
    }
    s->expansion_work += output_length;
#endif
    if (output_length > FC_MAX_COOKED) { free(digits); return -2; }
#ifdef FC_CANDIDATE_DECIMAL_LIMIT
    if (output_length > FC_CANDIDATE_DECIMAL) { free(digits); return -3; }
#endif
    unsigned char *output = (unsigned char *)malloc(output_length);
    if (output == NULL) { free(digits); return -1; }
    if (places <= 0) {
        memcpy(output, digits + first, significant);
        memset(output + significant, '0', output_length - significant);
    } else if (places >= (long long)significant) {
        output[0] = '0'; output[1] = '.';
        memset(output + 2, '0', (size_t)places - significant);
        memcpy(output + 2u + (size_t)places - significant, digits + first, significant);
    } else {
        size_t whole = significant - (size_t)places;
        memcpy(output, digits + first, whole);
        output[whole] = '.';
        memcpy(output + whole + 1u, digits + first + whole, (size_t)places);
    }
    free(digits);
    if (places > 0) {
        while (output_length > 0 && output[output_length - 1u] == '0') output_length--;
        if (output_length > 0 && output[output_length - 1u] == '.') output_length--;
    }
    *cooked = output;
    *cooked_length = output_length;
    return 1;
}

#ifdef FC_CANDIDATE_WORK_LIMIT
static int fc_work_available(const struct fc_scanner *s, size_t units);
static int fc_work_exhausted(struct fc_scanner *s, size_t start, size_t end);
#endif

static int fc_scan_number(struct fc_scanner *s) {
    size_t start = s->position;
    int is_float = 0;
    if (fc_match(s, start, "0x") || fc_match(s, start, "0b")) {
#ifdef FC_CANDIDATE_WORK_LIMIT
        if (!fc_work_available(s, 2u))
            return fc_work_exhausted(s, start, start + 1u);
#endif
        s->position += 2;
        while (s->position < s->length && fc_identifier_continue(s->source[s->position])) {
#ifdef FC_CANDIDATE_WORK_LIMIT
            if (!fc_work_available(s, 1u))
                return fc_work_exhausted(s, start, s->position);
#endif
            s->position++;
        }
    } else {
        s->position++;
        while (s->position < s->length &&
               (fc_decimal_digit(s->source[s->position]) || s->source[s->position] == '_')) {
#ifdef FC_CANDIDATE_WORK_LIMIT
            if (!fc_work_available(s, 1u))
                return fc_work_exhausted(s, start, s->position);
#endif
            s->position++;
        }
        if (s->position + 1 < s->length && s->source[s->position] == '.' &&
            fc_decimal_digit(s->source[s->position + 1])) {
            is_float = 1;
#ifdef FC_CANDIDATE_WORK_LIMIT
            if (!fc_work_available(s, 2u))
                return fc_work_exhausted(s, start, s->position);
#endif
            s->position += 2;
            while (s->position < s->length &&
                   (fc_decimal_digit(s->source[s->position]) || s->source[s->position] == '_')) {
#ifdef FC_CANDIDATE_WORK_LIMIT
                if (!fc_work_available(s, 1u))
                    return fc_work_exhausted(s, start, s->position);
#endif
                s->position++;
            }
        }
        if (s->position < s->length &&
            (s->source[s->position] == 'e' || s->source[s->position] == 'E')) {
            is_float = 1;
#ifdef FC_CANDIDATE_WORK_LIMIT
            if (!fc_work_available(s, 1u))
                return fc_work_exhausted(s, start, s->position);
#endif
            s->position++;
            if (s->position < s->length &&
                (s->source[s->position] == '+' || s->source[s->position] == '-')) {
#ifdef FC_CANDIDATE_WORK_LIMIT
                if (!fc_work_available(s, 1u))
                    return fc_work_exhausted(s, start, s->position);
#endif
                s->position++;
            }
            while (s->position < s->length &&
                   (fc_decimal_digit(s->source[s->position]) || s->source[s->position] == '_')) {
#ifdef FC_CANDIDATE_WORK_LIMIT
                if (!fc_work_available(s, 1u))
                    return fc_work_exhausted(s, start, s->position);
#endif
                s->position++;
            }
        }
        while (s->position < s->length && fc_identifier_continue(s->source[s->position])) {
#ifdef FC_CANDIDATE_WORK_LIMIT
            if (!fc_work_available(s, 1u))
                return fc_work_exhausted(s, start, s->position);
#endif
            s->position++;
        }
    }
    unsigned char *cooked = NULL;
    size_t cooked_length = 0;
    int valid = is_float ? fc_cook_float(s, s->source + start, s->position - start,
                                         &cooked, &cooked_length) :
                           fc_cook_integer(s->source + start, s->position - start,
                                           &cooked, &cooked_length);
#ifdef FC_CANDIDATE_WORK_LIMIT
    if (valid == -4) return fc_work_exhausted(s, start, s->position);
#endif
    if (valid == -2) return FC_UNSUPPORTED;
#ifdef FC_CANDIDATE_DECIMAL_LIMIT
    if (valid == -3) return FC_CANDIDATE_SIZE;
#endif
    if (valid < 0) return 3;
#ifdef FC_CANDIDATE_COOKED_LIMIT
    if (!fc_accept_cooked(s, cooked_length)) { free(cooked); return FC_CANDIDATE_COOKED_SIZE; }
#endif
    fc_token(s, is_float ? FC_FLOAT : FC_INTEGER, FC_NO_TRIVIA, start,
             s->position, cooked, cooked_length);
    if (!valid) fc_diagnostic(s, "E0004", start, s->position, "none", "");
    return 0;
}

static int fc_keyword(const struct fc_scanner *s, const unsigned char *bytes, size_t length) {
    if (s->previous_revision && length == 5u && memcmp(bytes, "defer", 5u) == 0)
        return 0;
    static const char *const words[] = {
        "as", "assert", "break", "case", "cleanup", "const", "continue",
        "defer", "discard", "else", "enum", "extern", "false", "fn", "for",
        "if", "import", "in", "let", "module", "mut", "null", "pub",
        "ptr", "ref", "region", "resource", "return", "struct", "switch",
        "true", "unsafe", "var", "while"
    };
    for (size_t i = 0; i < sizeof(words) / sizeof(words[0]); i++) {
        if (strlen(words[i]) == length && memcmp(bytes, words[i], length) == 0)
            return 1;
    }
    return 0;
}

static int fc_literal_keyword(const unsigned char *bytes, size_t length) {
    return (length == 4 && (memcmp(bytes, "true", 4) == 0 ||
                            memcmp(bytes, "null", 4) == 0)) ||
           (length == 5 && memcmp(bytes, "false", 5) == 0);
}

// Match the UTF-8 width used for invalid source-token recovery. Invalid,
// truncated, overlong, surrogate, and out-of-range encodings consume one byte.
static size_t fc_utf8_width(const unsigned char *bytes, size_t length) {
    unsigned char a = bytes[0];
    size_t width = a >= 0xc2 && a <= 0xdf ? 2u :
                   a >= 0xe0 && a <= 0xef ? 3u :
                   a >= 0xf0 && a <= 0xf4 ? 4u : 1u;
    if (width == 1 || width > length) return 1;
    for (size_t i = 1; i < width; i++) {
        if (bytes[i] < 0x80 || bytes[i] > 0xbf) return 1;
    }
    if ((a == 0xe0 && bytes[1] < 0xa0) ||
        (a == 0xed && bytes[1] > 0x9f) ||
        (a == 0xf0 && bytes[1] < 0x90) ||
        (a == 0xf4 && bytes[1] > 0x8f)) return 1;
    return width;
}

static int fc_valid_utf8(const unsigned char *bytes, size_t length) {
    for (size_t i = 0; i < length;) {
        if (bytes[i] < 0x80) { i++; continue; }
        size_t width = fc_utf8_width(bytes + i, length - i);
        if (width == 1) return 0;
        i += width;
    }
    return 1;
}

#ifdef FC_CANDIDATE_WORK_LIMIT
static int fc_work_available(const struct fc_scanner *s, size_t units) {
    return s->position <= s->work_limit &&
           s->expansion_work <= s->work_limit - s->position &&
           units <= s->work_limit - s->position - s->expansion_work;
}

static int fc_work_exhausted(struct fc_scanner *s, size_t start, size_t end) {
    if (end > start)
        fc_token_unlimited(s, FC_INVALID, FC_NO_TRIVIA, start, end, NULL, 0);
    fc_diagnostic(s, "E0010", end, end, "none", "");
    if (end < s->length)
        fc_token_unlimited(s, FC_INVALID, FC_NO_TRIVIA, end, s->length, NULL, 0);
    s->position = s->length;
    return 0;
}

static int fc_string_work_exhausted(struct fc_scanner *s, size_t start,
                                    unsigned char *bytes) {
    free(bytes);
    return fc_work_exhausted(s, start, s->position);
}
#endif

static int fc_scan_string(struct fc_scanner *s, unsigned kind, size_t opener) {
    size_t start = s->position;
#ifdef FC_CANDIDATE_WORK_LIMIT
    if (!fc_work_available(s, 1u))
        return fc_string_work_exhausted(s, start, NULL);
#endif
    s->position++;
    if (opener > 1u) {
#ifdef FC_CANDIDATE_WORK_LIMIT
        if (!fc_work_available(s, opener - 1u))
            return fc_string_work_exhausted(s, start, NULL);
#endif
        s->position += opener - 1u;
    }
    unsigned char *bytes = (unsigned char *)malloc(s->length - start + 2u);
    if (bytes == NULL) return 3;
    size_t used = 0;
    int terminated = 0;
    while (s->position < s->length) {
        unsigned char current = s->source[s->position];
        if (current == '"') {
#ifdef FC_CANDIDATE_WORK_LIMIT
            if (!fc_work_available(s, 1u))
                return fc_string_work_exhausted(s, start, bytes);
#endif
            s->position++; terminated = 1; break;
        }
        if (current == '\n' || current == '\r') {
            fc_diagnostic(s, "E0005", s->position, s->position + 1u, "exit", "\"");
            break;
        }
        if (current == '\\') {
#ifdef FC_CANDIDATE_WORK_LIMIT
            if (!fc_work_available(s, 1u))
                return fc_string_work_exhausted(s, start, bytes);
#endif
            size_t escape = s->position++;
            if (s->position == s->length) {
                fc_diagnostic(s, "E0005", escape, s->position, "none", "");
                break;
            }
#ifdef FC_CANDIDATE_WORK_LIMIT
            if (!fc_work_available(s, 1u))
                return fc_string_work_exhausted(s, start, bytes);
#endif
            current = s->source[s->position++];
            switch (current) {
            case '\\': case '"': bytes[used++] = current; break;
            case 'n': bytes[used++] = '\n'; break;
            case 'r': bytes[used++] = '\r'; break;
            case 't': bytes[used++] = '\t'; break;
            case '0':
                bytes[used++] = 0;
                if (kind == FC_C_STRING)
                    fc_diagnostic(s, "E0006", escape, s->position, "none", "");
                break;
            case 'x': {
                if (s->position + 2u > s->length ||
                    fc_digit_value(s->source[s->position]) < 0 ||
                    fc_digit_value(s->source[s->position + 1u]) < 0) {
                    while (s->position < s->length && s->position < escape + 4u &&
                           fc_digit_value(s->source[s->position]) >= 0) {
#ifdef FC_CANDIDATE_WORK_LIMIT
                        if (!fc_work_available(s, 1u))
                            return fc_string_work_exhausted(s, start, bytes);
#endif
                        s->position++;
                    }
                    fc_diagnostic(s, "E0005", escape, s->position, "none", "");
                    break;
                }
                unsigned value = (unsigned)(fc_digit_value(s->source[s->position]) * 16 +
                                            fc_digit_value(s->source[s->position + 1u]));
#ifdef FC_CANDIDATE_WORK_LIMIT
                if (!fc_work_available(s, 2u))
                    return fc_string_work_exhausted(s, start, bytes);
#endif
                s->position += 2;
                bytes[used++] = (unsigned char)value;
                if (kind == FC_C_STRING && value == 0)
                    fc_diagnostic(s, "E0006", escape, s->position, "none", "");
                break;
            }
            case 'u': {
                if (kind != FC_STRING || s->position == s->length ||
                    s->source[s->position] != '{') {
                    fc_diagnostic(s, "E0005", escape, s->position, "none", "");
                    break;
                }
#ifdef FC_CANDIDATE_WORK_LIMIT
                if (!fc_work_available(s, 1u))
                    return fc_string_work_exhausted(s, start, bytes);
#endif
                s->position++;
                size_t digits_start = s->position;
                while (s->position < s->length &&
                       fc_digit_value(s->source[s->position]) >= 0 &&
                       s->position - digits_start < 6u) {
#ifdef FC_CANDIDATE_WORK_LIMIT
                    if (!fc_work_available(s, 1u))
                        return fc_string_work_exhausted(s, start, bytes);
#endif
                    s->position++;
                }
                if (digits_start == s->position || s->position == s->length ||
                    s->source[s->position] != '}') {
                    while (s->position < s->length && s->source[s->position] != '}' &&
                           s->source[s->position] != '"') {
#ifdef FC_CANDIDATE_WORK_LIMIT
                        if (!fc_work_available(s, 1u))
                            return fc_string_work_exhausted(s, start, bytes);
#endif
                        s->position++;
                    }
                    if (s->position < s->length && s->source[s->position] == '}') {
#ifdef FC_CANDIDATE_WORK_LIMIT
                        if (!fc_work_available(s, 1u))
                            return fc_string_work_exhausted(s, start, bytes);
#endif
                        s->position++;
                    }
                    fc_diagnostic(s, "E0005", escape, s->position, "none", "");
                    break;
                }
                unsigned value = 0;
                for (size_t i = digits_start; i < s->position; i++)
                    value = value * 16u + (unsigned)fc_digit_value(s->source[i]);
#ifdef FC_CANDIDATE_WORK_LIMIT
                if (!fc_work_available(s, 1u))
                    return fc_string_work_exhausted(s, start, bytes);
#endif
                s->position++;
                if (value > 0x10ffffu || (value >= 0xd800u && value <= 0xdfffu)) {
                    fc_diagnostic(s, "E0005", escape, s->position, "none", "");
                    break;
                }
                if (value < 0x80u) bytes[used++] = (unsigned char)value;
                else if (value < 0x800u) {
                    bytes[used++] = (unsigned char)(0xc0u | (value >> 6));
                    bytes[used++] = (unsigned char)(0x80u | (value & 0x3fu));
                } else if (value < 0x10000u) {
                    bytes[used++] = (unsigned char)(0xe0u | (value >> 12));
                    bytes[used++] = (unsigned char)(0x80u | ((value >> 6) & 0x3fu));
                    bytes[used++] = (unsigned char)(0x80u | (value & 0x3fu));
                } else {
                    bytes[used++] = (unsigned char)(0xf0u | (value >> 18));
                    bytes[used++] = (unsigned char)(0x80u | ((value >> 12) & 0x3fu));
                    bytes[used++] = (unsigned char)(0x80u | ((value >> 6) & 0x3fu));
                    bytes[used++] = (unsigned char)(0x80u | (value & 0x3fu));
                }
                break;
            }
            default: fc_diagnostic(s, "E0005", escape, s->position, "none", ""); break;
            }
            continue;
        }
        if (current == 0) {
#ifdef FC_CANDIDATE_WORK_LIMIT
            if (!fc_work_available(s, 1u))
                return fc_string_work_exhausted(s, start, bytes);
#endif
            s->position++;
            fc_diagnostic(s, kind == FC_C_STRING ? "E0006" : "E0005",
                          s->position - 1u, s->position, "none", "");
            bytes[used++] = 0;
            continue;
        }
        if (current >= 0x80) {
            size_t width = fc_utf8_width(s->source + s->position,
                                          s->length - s->position);
#ifdef FC_CANDIDATE_WORK_LIMIT
            if (!fc_work_available(s, width))
                return fc_string_work_exhausted(s, start, bytes);
#endif
            if (width == 1) {
                fc_diagnostic(s, "E0001", s->position, s->position + 1u, "none", "");
                s->position++;
                continue;
            }
            memcpy(bytes + used, s->source + s->position, width);
            used += width;
            s->position += width;
            continue;
        }
#ifdef FC_CANDIDATE_WORK_LIMIT
        if (!fc_work_available(s, 1u))
            return fc_string_work_exhausted(s, start, bytes);
#endif
        bytes[used++] = current;
        s->position++;
    }
    if (!terminated && s->position == s->length)
        fc_diagnostic(s, "E0005", start, s->position, "exit", "\"");
    if (kind == FC_STRING && terminated && !fc_valid_utf8(bytes, used))
        fc_diagnostic(s, "E0005", start, s->position, "none", "");
    if (kind == FC_C_STRING) bytes[used++] = 0;
    unsigned char *hex = (unsigned char *)malloc(used * 2u + 1u);
    if (hex == NULL) { free(bytes); return 3; }
    static const char digits[] = "0123456789abcdef";
    for (size_t i = 0; i < used; i++) {
        hex[i * 2u] = (unsigned char)digits[bytes[i] >> 4];
        hex[i * 2u + 1u] = (unsigned char)digits[bytes[i] & 15u];
    }
    free(bytes);
#ifdef FC_CANDIDATE_COOKED_LIMIT
    if (!fc_accept_cooked(s, used * 2u)) { free(hex); return FC_CANDIDATE_COOKED_SIZE; }
#endif
    fc_token(s, kind, FC_NO_TRIVIA, start, s->position, hex, used * 2u);
    return 0;
}

static size_t fc_symbol_width(const struct fc_scanner *s) {
    static const char *const symbols[] = {
        "<<=", ">>=", "+=", "-=", "*=", "/=", "%=", "&=", "^=", "|=",
        "::", "->", "=>", "==", "!=", "<=", ">=", "<<", ">>", "&&", "||", "..",
        "@", "(", ")", "{", "}", "[", "]", ",", ";", ":", ".", "?", "=",
        "+", "-", "*", "/", "%", "!", "~", "&", "|", "^", "<", ">"
    };
    for (size_t i = 0; i < sizeof(symbols) / sizeof(symbols[0]); i++) {
        if (fc_match(s, s->position, symbols[i])) return strlen(symbols[i]);
    }
    return 0;
}

static int fc_scan_one(struct fc_scanner *s) {
    size_t start = s->position;
#ifdef FC_CANDIDATE_WORK_LIMIT
    if (!fc_work_available(s, 1u)) return fc_work_exhausted(s, start, start);
#endif
    unsigned char current = s->source[start];
    if (current == '\r' && !fc_match(s, start, "\r\n")) {
        s->position++;
        fc_token(s, FC_INVALID, FC_NO_TRIVIA, start, s->position, NULL, 0);
        fc_diagnostic(s, "E0001", start, s->position, "none", "");
        return 0;
    }
    if (fc_horizontal(current) || current == '\n' || current == '\r') {
        while (s->position < s->length) {
            current = s->source[s->position];
            size_t width = fc_horizontal(current) || current == '\n' ? 1u :
                           fc_match(s, s->position, "\r\n") ? 2u : 0u;
            if (width == 0) break;
#ifdef FC_CANDIDATE_WORK_LIMIT
            if (!fc_work_available(s, width))
                return fc_work_exhausted(s, start,
                                         s->position > start ? s->position : start + 1u);
#endif
            s->position += width;
        }
        fc_token(s, FC_TRIVIA, FC_WHITESPACE, start, s->position, NULL, 0);
        return 0;
    }
    if (fc_match(s, start, "//")) {
#ifdef FC_CANDIDATE_WORK_LIMIT
        if (!fc_work_available(s, 2u))
            return fc_work_exhausted(s, start, start + 1u);
#endif
        s->position += 2;
        while (s->position < s->length && s->source[s->position] != '\n' &&
               s->source[s->position] != '\r') {
#ifdef FC_CANDIDATE_WORK_LIMIT
            if (!fc_work_available(s, 1u))
                return fc_work_exhausted(s, start, s->position);
#endif
            s->position++;
        }
        fc_token(s, FC_TRIVIA, fc_match(s, start, "///") ? FC_DOC_LINE : FC_LINE_COMMENT,
                 start, s->position, NULL, 0);
        return 0;
    }
    if (fc_match(s, start, "/*")) {
        size_t depth = 1;
#ifdef FC_CANDIDATE_WORK_LIMIT
        if (!fc_work_available(s, 2u))
            return fc_work_exhausted(s, start, start + 1u);
#endif
        s->position += 2;
        while (s->position < s->length && depth > 0) {
            int open = fc_match(s, s->position, "/*");
            int close = !open && fc_match(s, s->position, "*/");
            size_t width = open || close ? 2u : 1u;
#ifdef FC_CANDIDATE_WORK_LIMIT
            if (!fc_work_available(s, width))
                return fc_work_exhausted(s, start, s->position);
#endif
            if (open) depth++;
            else if (close) depth--;
            s->position += width;
        }
        fc_token(s, FC_TRIVIA, FC_BLOCK_COMMENT, start, s->position, NULL, 0);
        if (depth != 0) fc_diagnostic(s, "E0001", start, s->position, "exit", "*/");
        return 0;
    }
    if ((current == 'b' || current == 'c') && start + 1 < s->length &&
        s->source[start + 1u] == '"')
        return fc_scan_string(s, current == 'b' ? FC_BYTE_STRING : FC_C_STRING, 2);
    if (current == '"') return fc_scan_string(s, FC_STRING, 1);
    if (fc_decimal_digit(current)) return fc_scan_number(s);
    if (fc_identifier_start(current)) {
        s->position++;
        while (s->position < s->length && fc_identifier_continue(s->source[s->position])) {
#ifdef FC_CANDIDATE_WORK_LIMIT
            if (!fc_work_available(s, 1u))
                return fc_work_exhausted(s, start, s->position);
#endif
            s->position++;
        }
        size_t length = s->position - start;
        int keyword = fc_keyword(s, s->source + start, length);
        int literal = keyword && fc_literal_keyword(s->source + start, length);
        fc_token(s, keyword ? FC_KEYWORD : FC_IDENTIFIER, FC_NO_TRIVIA, start,
                 s->position, literal ? s->source + start : NULL, literal ? length : 0);
        return 0;
    }
    size_t width = fc_symbol_width(s);
    if (width != 0) {
#ifdef FC_CANDIDATE_WORK_LIMIT
        if (!fc_work_available(s, width))
            return fc_work_exhausted(s, start, start + 1u);
#endif
        s->position += width;
        fc_token(s, FC_SYMBOL, FC_NO_TRIVIA, start, s->position, NULL, 0);
        return 0;
    }
    width = current >= 0x80 ? fc_utf8_width(s->source + start, s->length - start) : 1;
#ifdef FC_CANDIDATE_WORK_LIMIT
    if (!fc_work_available(s, width))
        return fc_work_exhausted(s, start, start + 1u);
#endif
    s->position += width;
    fc_token(s, FC_INVALID, FC_NO_TRIVIA, start, s->position, NULL, 0);
    fc_diagnostic(s, "E0001", start, s->position, "none", "");
    return 0;
}

static int fc_scan(struct fc_scanner *s) {
#ifdef FC_CANDIDATE_SOURCE_LIMIT
    if (s->length > FC_MAX_SOURCE) return FC_CANDIDATE_SOURCE_SIZE;
#endif
#ifdef FC_CANDIDATE_BYTE_LIMIT
    if (s->byte_limit_set && s->length > s->byte_limit) {
        fc_work_exhausted(s, 0, 0);
#ifdef FC_CANDIDATE_DIAGNOSTIC_PAYLOAD
        if (s->diagnostic_payload_failed) return 3;
#endif
#ifdef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
        if (s->diagnostic_allocation_failed) return 3;
#endif
#ifdef FC_CANDIDATE_GEOMETRIC_TOKENS
        if (s->token_allocation_failed) return 3;
#endif
        goto fc_source_byte_limit_finished;
    }
#endif
    if (s->length >= 3 && s->source[0] == 0xef && s->source[1] == 0xbb &&
        s->source[2] == 0xbf) {
#ifdef FC_CANDIDATE_WORK_LIMIT
        if (!fc_work_available(s, 3u)) {
            fc_work_exhausted(s, 0, 0);
        } else
#endif
        {
            fc_token(s, FC_TRIVIA, FC_PREAMBLE, 0, 3, NULL, 0);
#ifdef FC_CANDIDATE_GEOMETRIC_TOKENS
            if (s->token_allocation_failed) return 3;
#endif
            s->position = 3;
        }
    }
    while (s->position < s->length
#ifdef FC_CANDIDATE_TOKEN_LIMIT
           && !s->token_limit_exhausted
#endif
    ) {
        int status = fc_scan_one(s);
#ifdef FC_CANDIDATE_DIAGNOSTIC_PAYLOAD
        if (s->diagnostic_payload_failed) return 3;
#endif
        if (status != 0) return status;
#ifdef FC_CANDIDATE_DIAGNOSTIC_LIMIT
        if (s->diagnostic_limit_exhausted) {
            fc_diagnostic(s, "E0010", s->position, s->position, "none", "");
#ifdef FC_CANDIDATE_DIAGNOSTIC_PAYLOAD
            if (s->diagnostic_payload_failed) return 3;
#endif
#ifdef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
            if (s->diagnostic_allocation_failed) return 3;
#endif
            if (s->position < s->length)
                fc_token(s, FC_INVALID, FC_NO_TRIVIA, s->position,
                         s->length, NULL, 0);
            s->position = s->length;
        }
#endif
#ifdef FC_CANDIDATE_GEOMETRIC_TOKENS
        if (s->token_allocation_failed) return 3;
#endif
#ifdef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
        if (s->diagnostic_allocation_failed) return 3;
#endif
    }
#ifdef FC_CANDIDATE_TOKEN_LIMIT
    if (s->token_limit_exhausted) {
        fc_diagnostic(s, "E0010", s->position, s->position, "none", "");
#ifdef FC_CANDIDATE_DIAGNOSTIC_PAYLOAD
        if (s->diagnostic_payload_failed) return 3;
#endif
#ifdef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
        if (s->diagnostic_allocation_failed) return 3;
#endif
        if (s->position < s->length)
            fc_token_unlimited(s, FC_INVALID, FC_NO_TRIVIA, s->position,
                               s->length, NULL, 0);
#ifdef FC_CANDIDATE_GEOMETRIC_TOKENS
        if (s->token_allocation_failed) return 3;
#endif
        s->position = s->length;
    }
#endif
#ifdef FC_CANDIDATE_BYTE_LIMIT
fc_source_byte_limit_finished:
#endif
    fc_token_unlimited(s, FC_EOF, FC_NO_TRIVIA, s->length, s->length, NULL, 0);
#ifdef FC_CANDIDATE_GEOMETRIC_TOKENS
    if (s->token_allocation_failed) return 3;
#endif
    return 0;
}

static int fc_compare_diagnostic(const void *left, const void *right) {
    const struct fc_diagnostic *a = (const struct fc_diagnostic *)left;
    const struct fc_diagnostic *b = (const struct fc_diagnostic *)right;
    if (a->start != b->start) return a->start < b->start ? -1 : 1;
    if (a->end != b->end) return a->end < b->end ? -1 : 1;
#ifdef FC_CANDIDATE_DIAGNOSTIC_ENTRY_LAYOUT
    if (a->code != b->code) return a->code < b->code ? -1 : 1;
    if (a->exit_repair != b->exit_repair)
        return a->exit_repair < b->exit_repair ? -1 : 1;
#else
    int value = strcmp(a->code, b->code);
    if (value != 0) return value;
    value = strcmp(a->repair, b->repair);
    if (value != 0) return value;
#endif
    size_t a_length = fc_diagnostic_token_length(a);
    size_t b_length = fc_diagnostic_token_length(b);
    size_t minimum = a_length < b_length ? a_length : b_length;
    if (minimum != 0) {
        int token_cmp = memcmp(fc_diagnostic_token_data(a), fc_diagnostic_token_data(b), minimum);
        if (token_cmp != 0) return token_cmp;
    }
    return a_length == b_length ? 0 : a_length < b_length ? -1 : 1;
}

static void fc_hex(const unsigned char *bytes, size_t length) {
    static const char digits[] = "0123456789abcdef";
    if (length == 0) { putchar('-'); return; }
    for (size_t i = 0; i < length; i++) {
        putchar(digits[bytes[i] >> 4]);
        putchar(digits[bytes[i] & 15]);
    }
}

static void fc_print(const struct fc_scanner *s) {
    for (size_t i = 0; i < s->token_count; i++) {
        const struct fc_token *token = &s->tokens[i];
        printf("T %u %u %zu %zu ", token->kind, token->trivia, token->start, token->end);
        fc_hex(token->cooked, token->cooked_length);
        putchar('\n');
    }
    for (size_t i = 0; i < s->diagnostic_count; i++) {
        const struct fc_diagnostic *diagnostic = &s->diagnostics[i];
#ifdef FC_CANDIDATE_DIAGNOSTIC_ENTRY_LAYOUT
        const char *code = fc_diagnostic_code_text(diagnostic);
        const char *repair = diagnostic->exit_repair ? "exit" : "none";
#else
        const char *code = diagnostic->code;
        const char *repair = diagnostic->repair;
#endif
        printf("D %s %zu %zu %s ", code, diagnostic->start,
               diagnostic->end, repair);
        fc_hex(fc_diagnostic_token_data(diagnostic),
               fc_diagnostic_token_length(diagnostic));
        putchar('\n');
    }
}

static void fc_release_cooked(const struct fc_scanner *s) {
    for (size_t i = 0; i < s->token_count; i++) {
        if (s->tokens[i].kind == FC_INTEGER || s->tokens[i].kind == FC_FLOAT ||
            s->tokens[i].kind == FC_STRING || s->tokens[i].kind == FC_BYTE_STRING ||
            s->tokens[i].kind == FC_C_STRING)
            free((void *)s->tokens[i].cooked);
    }
}

static void fc_release_diagnostics(const struct fc_scanner *s) {
#ifdef FC_CANDIDATE_DIAGNOSTIC_PAYLOAD
    for (size_t i = 0; i < s->diagnostic_count; i++) {
        if (fc_diagnostic_token_length(&s->diagnostics[i]) != 0)
            free((void *)fc_diagnostic_token_data(&s->diagnostics[i]));
    }
#else
    (void)s;
#endif
}

#ifndef FC_SCANNER_LIBRARY
int main(void) {
#ifdef _WIN32
    if (_setmode(_fileno(stdin), _O_BINARY) == -1 ||
        _setmode(_fileno(stdout), _O_BINARY) == -1) return FC_BAD_INPUT;
#endif
    unsigned char header[4];
    if (fread(header, 1, 4, stdin) != 4) return FC_BAD_INPUT;
    uint32_t length = (uint32_t)header[0] | ((uint32_t)header[1] << 8) |
                      ((uint32_t)header[2] << 16) | ((uint32_t)header[3] << 24);
    if (length > FC_MAX_SOURCE) {
#ifdef FC_CANDIDATE_SOURCE_LIMIT
        return FC_CANDIDATE_SOURCE_SIZE;
#else
        return FC_BAD_INPUT;
#endif
    }
    unsigned char *source = (unsigned char *)malloc(length == 0 ? 1u : (size_t)length);
    struct fc_token *tokens = NULL;
#ifndef FC_CANDIDATE_GEOMETRIC_TOKENS
    tokens = (struct fc_token *)calloc((size_t)length + 1u, sizeof(*tokens));
#endif
    struct fc_diagnostic *diagnostics = NULL;
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
    diagnostics = (struct fc_diagnostic *)calloc((size_t)length + 1u, sizeof(*diagnostics));
#endif
    if (source == NULL
#ifndef FC_CANDIDATE_GEOMETRIC_DIAGNOSTICS
        || diagnostics == NULL
#endif
#ifndef FC_CANDIDATE_GEOMETRIC_TOKENS
        || tokens == NULL
#endif
    ) {
        free(source); free(tokens); free(diagnostics); return 3;
    }
    if (fread(source, 1, length, stdin) != length || getchar() != EOF) {
        free(source); free(tokens); free(diagnostics); return FC_BAD_INPUT;
    }
    struct fc_scanner scanner = {0};
    scanner.source = source;
    scanner.length = (size_t)length;
    scanner.tokens = tokens;
    scanner.diagnostics = diagnostics;
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
        if (scanner.diagnostic_count != 0)
            qsort(scanner.diagnostics, scanner.diagnostic_count,
                  sizeof(*scanner.diagnostics), fc_compare_diagnostic);
        fc_print(&scanner);
    }
    fc_release_cooked(&scanner);
    fc_release_diagnostics(&scanner);
    free(source); free(tokens); free(diagnostics);
    return status;
}
#endif
