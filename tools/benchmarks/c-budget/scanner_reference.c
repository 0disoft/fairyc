// Independent C11 lexical reference. Decimal expansions beyond the Go lexer's
// default 16 MiB work budget remain unsupported; this is not a full scanner.
#include <errno.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#ifdef _WIN32
#include <fcntl.h>
#include <io.h>
#endif

enum { FC_MAX_SOURCE = 32768, FC_MAX_COOKED = 16 << 20,
       FC_UNSUPPORTED = 7, FC_BAD_INPUT = 8 };
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
struct fc_diagnostic {
    const char *code, *repair, *token;
    size_t start, end;
};
struct fc_scanner {
    const unsigned char *source;
    size_t length, position, token_count, diagnostic_count;
    struct fc_token *tokens;
    struct fc_diagnostic *diagnostics;
};

static void fc_token(struct fc_scanner *s, unsigned kind, unsigned trivia,
                     size_t start, size_t end, const unsigned char *cooked,
                     size_t cooked_length) {
    s->tokens[s->token_count++] = (struct fc_token){kind, trivia, start, end,
                                                   cooked, cooked_length};
}

static void fc_diagnostic(struct fc_scanner *s, const char *code,
                          size_t start, size_t end, const char *repair,
                          const char *token) {
    s->diagnostics[s->diagnostic_count++] =
        (struct fc_diagnostic){code, repair, token, start, end};
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

// Return 1 for cooked decimal, 0 for an invalid spelling, -1 on OOM and
// -2 when its expansion exceeds this still-partial reference's limit.
static int fc_cook_float(const unsigned char *raw, size_t length,
                          unsigned char **cooked, size_t *cooked_length) {
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
        free(digits); return -2;
    }
    long long places = (long long)fraction_length - exponent;
    size_t output_length = places <= 0 ? significant + (size_t)-places :
                           places >= (long long)significant ? 2u + (size_t)places :
                           significant + 1u;
    if (output_length > FC_MAX_COOKED) { free(digits); return -2; }
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

static int fc_scan_number(struct fc_scanner *s) {
    size_t start = s->position;
    int is_float = 0;
    if (fc_match(s, start, "0x") || fc_match(s, start, "0b")) {
        s->position += 2;
        while (s->position < s->length && fc_identifier_continue(s->source[s->position]))
            s->position++;
    } else {
        s->position++;
        while (s->position < s->length &&
               (fc_decimal_digit(s->source[s->position]) || s->source[s->position] == '_'))
            s->position++;
        if (s->position + 1 < s->length && s->source[s->position] == '.' &&
            fc_decimal_digit(s->source[s->position + 1])) {
            is_float = 1;
            s->position += 2;
            while (s->position < s->length &&
                   (fc_decimal_digit(s->source[s->position]) || s->source[s->position] == '_'))
                s->position++;
        }
        if (s->position < s->length &&
            (s->source[s->position] == 'e' || s->source[s->position] == 'E')) {
            is_float = 1;
            s->position++;
            if (s->position < s->length &&
                (s->source[s->position] == '+' || s->source[s->position] == '-')) s->position++;
            while (s->position < s->length &&
                   (fc_decimal_digit(s->source[s->position]) || s->source[s->position] == '_'))
                s->position++;
        }
        while (s->position < s->length && fc_identifier_continue(s->source[s->position]))
            s->position++;
    }
    unsigned char *cooked = NULL;
    size_t cooked_length = 0;
    int valid = is_float ? fc_cook_float(s->source + start, s->position - start,
                                        &cooked, &cooked_length) :
                           fc_cook_integer(s->source + start, s->position - start,
                                           &cooked, &cooked_length);
    if (valid == -2) return FC_UNSUPPORTED;
    if (valid < 0) return 3;
    fc_token(s, is_float ? FC_FLOAT : FC_INTEGER, FC_NO_TRIVIA, start,
             s->position, cooked, cooked_length);
    if (!valid) fc_diagnostic(s, "E0004", start, s->position, "none", "");
    return 0;
}

static int fc_keyword(const unsigned char *bytes, size_t length) {
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

static int fc_scan_string(struct fc_scanner *s, unsigned kind, size_t opener) {
    size_t start = s->position;
    s->position += opener;
    unsigned char *bytes = (unsigned char *)malloc(s->length - start + 2u);
    if (bytes == NULL) return 3;
    size_t used = 0;
    int terminated = 0;
    while (s->position < s->length) {
        unsigned char current = s->source[s->position];
        if (current == '"') { s->position++; terminated = 1; break; }
        if (current == '\n' || current == '\r') {
            fc_diagnostic(s, "E0005", s->position, s->position + 1u, "exit", "\"");
            break;
        }
        if (current == '\\') {
            size_t escape = s->position++;
            if (s->position == s->length) {
                fc_diagnostic(s, "E0005", escape, s->position, "none", "");
                break;
            }
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
                           fc_digit_value(s->source[s->position]) >= 0)
                        s->position++;
                    fc_diagnostic(s, "E0005", escape, s->position, "none", "");
                    break;
                }
                unsigned value = (unsigned)(fc_digit_value(s->source[s->position]) * 16 +
                                            fc_digit_value(s->source[s->position + 1u]));
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
                s->position++;
                size_t digits_start = s->position;
                while (s->position < s->length &&
                       fc_digit_value(s->source[s->position]) >= 0 &&
                       s->position - digits_start < 6u)
                    s->position++;
                if (digits_start == s->position || s->position == s->length ||
                    s->source[s->position] != '}') {
                    while (s->position < s->length && s->source[s->position] != '}' &&
                           s->source[s->position] != '"')
                        s->position++;
                    if (s->position < s->length && s->source[s->position] == '}')
                        s->position++;
                    fc_diagnostic(s, "E0005", escape, s->position, "none", "");
                    break;
                }
                unsigned value = 0;
                for (size_t i = digits_start; i < s->position; i++)
                    value = value * 16u + (unsigned)fc_digit_value(s->source[i]);
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
            s->position++;
            fc_diagnostic(s, kind == FC_C_STRING ? "E0006" : "E0005",
                          s->position - 1u, s->position, "none", "");
            bytes[used++] = 0;
            continue;
        }
        if (current >= 0x80) {
            size_t width = fc_utf8_width(s->source + s->position,
                                          s->length - s->position);
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
            if (fc_horizontal(current) || current == '\n') s->position++;
            else if (fc_match(s, s->position, "\r\n")) s->position += 2;
            else break;
        }
        fc_token(s, FC_TRIVIA, FC_WHITESPACE, start, s->position, NULL, 0);
        return 0;
    }
    if (fc_match(s, start, "//")) {
        s->position += 2;
        while (s->position < s->length && s->source[s->position] != '\n' &&
               s->source[s->position] != '\r') s->position++;
        fc_token(s, FC_TRIVIA, fc_match(s, start, "///") ? FC_DOC_LINE : FC_LINE_COMMENT,
                 start, s->position, NULL, 0);
        return 0;
    }
    if (fc_match(s, start, "/*")) {
        size_t depth = 1;
        s->position += 2;
        while (s->position < s->length && depth > 0) {
            if (fc_match(s, s->position, "/*")) { depth++; s->position += 2; }
            else if (fc_match(s, s->position, "*/")) { depth--; s->position += 2; }
            else s->position++;
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
        while (s->position < s->length && fc_identifier_continue(s->source[s->position]))
            s->position++;
        size_t length = s->position - start;
        int keyword = fc_keyword(s->source + start, length);
        int literal = keyword && fc_literal_keyword(s->source + start, length);
        fc_token(s, keyword ? FC_KEYWORD : FC_IDENTIFIER, FC_NO_TRIVIA, start,
                 s->position, literal ? s->source + start : NULL, literal ? length : 0);
        return 0;
    }
    size_t width = fc_symbol_width(s);
    if (width != 0) {
        s->position += width;
        fc_token(s, FC_SYMBOL, FC_NO_TRIVIA, start, s->position, NULL, 0);
        return 0;
    }
    width = current >= 0x80 ? fc_utf8_width(s->source + start, s->length - start) : 1;
    s->position += width;
    fc_token(s, FC_INVALID, FC_NO_TRIVIA, start, s->position, NULL, 0);
    fc_diagnostic(s, "E0001", start, s->position, "none", "");
    return 0;
}

static int fc_scan(struct fc_scanner *s) {
    if (s->length >= 3 && s->source[0] == 0xef && s->source[1] == 0xbb &&
        s->source[2] == 0xbf) {
        fc_token(s, FC_TRIVIA, FC_PREAMBLE, 0, 3, NULL, 0);
        s->position = 3;
    }
    while (s->position < s->length) {
        int status = fc_scan_one(s);
        if (status != 0) return status;
    }
    fc_token(s, FC_EOF, FC_NO_TRIVIA, s->length, s->length, NULL, 0);
    return 0;
}

static int fc_compare_diagnostic(const void *left, const void *right) {
    const struct fc_diagnostic *a = (const struct fc_diagnostic *)left;
    const struct fc_diagnostic *b = (const struct fc_diagnostic *)right;
    if (a->start != b->start) return a->start < b->start ? -1 : 1;
    if (a->end != b->end) return a->end < b->end ? -1 : 1;
    int value = strcmp(a->code, b->code);
    if (value != 0) return value;
    value = strcmp(a->repair, b->repair);
    return value != 0 ? value : strcmp(a->token, b->token);
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
        printf("D %s %zu %zu %s ", diagnostic->code, diagnostic->start,
               diagnostic->end, diagnostic->repair);
        fc_hex((const unsigned char *)diagnostic->token, strlen(diagnostic->token));
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
    if (length > FC_MAX_SOURCE) return FC_BAD_INPUT;
    unsigned char *source = (unsigned char *)malloc(length == 0 ? 1u : (size_t)length);
    struct fc_token *tokens = (struct fc_token *)calloc((size_t)length + 1u, sizeof(*tokens));
    struct fc_diagnostic *diagnostics = (struct fc_diagnostic *)calloc((size_t)length + 1u, sizeof(*diagnostics));
    if (source == NULL || tokens == NULL || diagnostics == NULL) {
        free(source); free(tokens); free(diagnostics); return 3;
    }
    if (fread(source, 1, length, stdin) != length || getchar() != EOF) {
        free(source); free(tokens); free(diagnostics); return FC_BAD_INPUT;
    }
    struct fc_scanner scanner = {source, (size_t)length, 0, 0, 0, tokens, diagnostics};
    int status = fc_scan(&scanner);
    if (status == 0) {
        qsort(scanner.diagnostics, scanner.diagnostic_count,
              sizeof(*scanner.diagnostics), fc_compare_diagnostic);
        fc_print(&scanner);
    }
    fc_release_cooked(&scanner);
    free(source); free(tokens); free(diagnostics);
    return status;
}
#endif
