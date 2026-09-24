#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

/* Independent C reference for the quoted-ASCII pages::run workload only. */
static uint32_t little_u32(const unsigned char *bytes) {
    return (uint32_t)bytes[0] | ((uint32_t)bytes[1] << 8) |
           ((uint32_t)bytes[2] << 16) | ((uint32_t)bytes[3] << 24);
}

struct page_window {
    size_t start;
    size_t position;
};

static int number(struct page_window *window, size_t value) {
    if (window->position >= window->start && window->position - window->start < 4096u) {
        if (printf("%zu ", value) < 0) return 0;
    }
    window->position++;
    return 1;
}

static int cooked_text(struct page_window *window, const unsigned char *data, size_t length) {
    static const unsigned char hex[] = "0123456789abcdef";
    if (!number(window, length * 2u)) return 0;
    for (size_t i = 0; i < length; i++) {
        if (!number(window, hex[data[i] >> 4]) ||
            !number(window, hex[data[i] & 15u])) return 0;
    }
    return 1;
}

int main(void) {
    unsigned char header[9];
    if (fread(header, 1, sizeof(header), stdin) != sizeof(header)) return 8;
    uint32_t length = little_u32(header + 1);
    uint32_t start = little_u32(header + 5);
    if (header[0] > 1 || start > 1048576u) return 8;
    if (length > 32768u) return 4;
    unsigned char *input = (unsigned char *)malloc(length == 0 ? 1u : (size_t)length);
    if (input == NULL) return 3;
    if (fread(input, 1, length, stdin) != length || getchar() != EOF) {
        free(input);
        return 8;
    }
    if (length < 2 || input[0] != '"' || input[length - 1] != '"') {
        free(input);
        return 2;
    }
    for (uint32_t i = 1; i + 1 < length; i++) {
        if (input[i] < 32 || input[i] > 126 || input[i] == '"' || input[i] == '\\') {
            free(input);
            return 2;
        }
    }

    struct page_window window = {(size_t)start, 0};
    size_t cooked_length = (size_t)length - 2u;
    int ok = number(&window, 2u) && number(&window, 7u) &&
             number(&window, 0u) && number(&window, 0u) &&
             number(&window, length) && cooked_text(&window, input + 1, cooked_length) &&
             number(&window, 1u) && number(&window, 0u) &&
             number(&window, length) && number(&window, length) &&
             cooked_text(&window, input, 0u) && number(&window, 0u);
    free(input);
    if (!ok || fflush(stdout) != 0) return 3;
    return 0;
}
