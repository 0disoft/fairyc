//go:build ignore

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>
#include <io.h>

static uint64_t graph_probe_number(size_t width) {
    uint64_t value = 0u;
    size_t i;
    for (i = 0u; i < width; ++i) {
        int next = getchar();
        if (next == EOF) { return UINT64_MAX; }
        value = (value << 8u) | (uint8_t)next;
    }
    return value;
}
static void graph_probe_put(uint64_t value, size_t width) {
    size_t i = width;
    while (i != 0u) { --i; (void)putchar((int)((value >> (i*8u)) & 255u)); }
}
static void graph_probe_value(int32_t status, uint8_t found, const fycs_graph_value_v1 *value) {
    graph_probe_put((uint32_t)status, 1u); graph_probe_put(found, 1u);
    graph_probe_put(status == 0 ? value->length : 0u, 4u);
    if (status == 0 && value->length != 0u) { (void)fwrite(value->bytes, 1u, value->length, stdout); }
}

static int graph_probe_keys(void) {
    uint32_t count = (uint32_t)graph_probe_number(4u), i;
    if (count > 4096u) { return 2; }
    for (i = 0u; i < count; ++i) {
        uint8_t prefix[32], output[32] = {0};
        fycs_graph_value_v1 key;
        uint8_t alias = (uint8_t)graph_probe_number(1u);
        size_t maximum;
        uint8_t *bytes, *result;
        int32_t status;
        key.codec = (uint8_t)graph_probe_number(1u);
        maximum = (size_t)graph_probe_number(8u);
        key.length = (size_t)graph_probe_number(4u);
        if (key.length > 1048576u || alias > 2u || fread(prefix, 1u, 32u, stdin) != 32u) { return 3; }
        bytes = (uint8_t *)malloc(key.length > 32u ? key.length : 32u);
        if (bytes == NULL) { return 4; }
        if (fread(bytes, 1u, key.length, stdin) != key.length) { free(bytes); return 5; }
        key.bytes = bytes;
        result = alias == 1u ? prefix : (alias == 2u ? bytes : output);
        status = fycs_graph_map_key_v1(prefix, &key, maximum, result);
        graph_probe_put((uint32_t)status, 1u);
        if (status == 0) { (void)fwrite(result, 1u, 32u, stdout); }
        else { (void)fwrite(output, 1u, 32u, stdout); }
        free(bytes);
    }
    return fgetc(stdin) == EOF && ferror(stdin) == 0 ? 0 : 6;
}

#define GRAPH_CHECK(value) do { if (!(value)) { return __LINE__; } } while (0)
static int graph_probe_boundaries(void) {
    fycs_overlay_v1 transaction = {0}, other = {0};
    fycs_overlay_entry_v1 writes[2], other_writes[2];
    uint8_t arena[256], scratch[256], other_arena[256], other_scratch[256];
    uint8_t values[8], other_values[8], a[32] = {1}, b[32] = {2}, c[32] = {3}, output[32];
    fycs_graph_epoch_v1 epochs[2], other_epochs[2];
    fycs_graph_v1 context = {0}, foreign = {0};
    fycs_graph_value_v1 value, saved;
    fycs_graph_borrow_v1 borrow, nested;
    uint8_t found;
    size_t used;
    GRAPH_CHECK(fycs_overlay_begin_v1(&transaction, NULL, 0u, writes, 2u, arena, scratch, sizeof(arena), 256u, 3u) == 0);
    GRAPH_CHECK(fycs_graph_begin_v1(NULL, &transaction, values, 8u, epochs, 2u) == 1);
    GRAPH_CHECK(fycs_graph_begin_v1(&context, &transaction, NULL, 8u, epochs, 2u) == 1);
    GRAPH_CHECK(fycs_graph_begin_v1(&context, &transaction, values, 8u, epochs, 1u) == 1);
    GRAPH_CHECK(fycs_graph_begin_v1(&context, &transaction, values, 8u, epochs, 2u) == 0);
    GRAPH_CHECK(fycs_graph_value_copy_v1(&context, FYCS_CODEC_BYTES, (const uint8_t *)"abc", 3u, &value) == 0);
    GRAPH_CHECK(fycs_graph_write_v1(&context, a, &value, 0u) == 0);
    GRAPH_CHECK(fycs_graph_borrow_v1_begin(&context, a, FYCS_CODEC_BYTES, &borrow, &found) == 0 && found == 1u);
    GRAPH_CHECK(fycs_graph_borrow_view_v1(&context, &borrow, 3u, 0u, &nested) == 0);
    GRAPH_CHECK(fycs_graph_borrow_copy_v1(&context, &nested, &saved) == 0 && saved.length == 0u);
    GRAPH_CHECK(fycs_graph_borrow_view_v1(&context, &borrow, SIZE_MAX, 1u, &nested) == 10);
    GRAPH_CHECK(fycs_graph_borrow_view_v1(&context, &borrow, 0u, SIZE_MAX, &nested) == 10);
    GRAPH_CHECK(fycs_graph_borrow_view_v1(&context, &borrow, 3u, 1u, &nested) == 10);
    GRAPH_CHECK(fycs_graph_value_copy_v1(&context, FYCS_CODEC_BYTES, (const uint8_t *)"xy", 2u, &saved) == 0);
    GRAPH_CHECK(context.used == 8u);
    GRAPH_CHECK(fycs_graph_value_copy_v1(&context, FYCS_CODEC_BYTES, (const uint8_t *)"z", 1u, &saved) == 8 && context.used == 8u);
    GRAPH_CHECK(saved.length == 2u && memcmp(saved.bytes, "xy", 2u) == 0);
    GRAPH_CHECK(fycs_graph_value_copy_v1(&context, FYCS_CODEC_BOOL, (const uint8_t *)"x", 1u, &saved) == 6 && context.used == 8u);
    GRAPH_CHECK(fycs_graph_write_v1(&context, b, &value, 0u) == 0 && context.epoch_count == 2u);
    GRAPH_CHECK(fycs_graph_write_v1(&context, c, &value, 0u) == 4 && context.generation == 2u);
    value.bytes = (const uint8_t *)"four"; value.length = 4u;
    GRAPH_CHECK(fycs_graph_write_v1(&context, a, &value, 0u) == 3 && context.generation == 2u);
    GRAPH_CHECK(fycs_graph_borrow_view_v1(&context, &borrow, 1u, 1u, &borrow) == 0 && borrow.value.length == 1u && borrow.value.bytes[0] == 'b');
    GRAPH_CHECK(fycs_overlay_begin_v1(&other, NULL, 0u, other_writes, 2u, other_arena, other_scratch, sizeof(other_arena), 256u, 3u) == 0);
    GRAPH_CHECK(fycs_graph_begin_v1(&foreign, &other, other_values, 8u, other_epochs, 2u) == 0);
    GRAPH_CHECK(fycs_graph_borrow_copy_v1(&foreign, &borrow, &saved) == 9);
    /* View/copy keeps the original codec even when a UTF-8 byte slice is not
     * canonical. Consumption by a typed write is the validation boundary. */
    value.bytes = (const uint8_t *)"\xc3\xa9"; value.length = 2u; value.codec = FYCS_CODEC_UTF8;
    GRAPH_CHECK(fycs_graph_write_v1(&foreign, a, &value, 0u) == 0);
    GRAPH_CHECK(fycs_graph_borrow_v1_begin(&foreign, a, FYCS_CODEC_UTF8, &nested, &found) == 0 && found == 1u);
    GRAPH_CHECK(fycs_graph_borrow_view_v1(&foreign, &nested, 1u, 1u, &nested) == 0);
    GRAPH_CHECK(fycs_graph_borrow_copy_v1(&foreign, &nested, &saved) == 0 && saved.codec == FYCS_CODEC_UTF8 && saved.length == 1u && saved.bytes[0] == 0xa9u);
    GRAPH_CHECK(fycs_graph_write_v1(&foreign, b, &saved, 0u) == 6 && foreign.epoch_count == 1u);
    GRAPH_CHECK(fycs_graph_borrow_v1_begin(&foreign, b, FYCS_CODEC_BYTES32, &nested, &found) == 0 && found == 0u && nested.value.length == 32u);
    GRAPH_CHECK(fycs_graph_borrow_copy_v1(&foreign, &nested, &saved) == 9);
    value.bytes = NULL; value.length = 0u; value.codec = FYCS_CODEC_BYTES;
    GRAPH_CHECK(fycs_graph_write_v1(&foreign, b, &value, 0u) == 0);
    GRAPH_CHECK(fycs_graph_borrow_v1_begin(&foreign, b, FYCS_CODEC_BYTES, &nested, &found) == 0 && found == 1u && nested.value.length == 0u);
    GRAPH_CHECK(fycs_graph_borrow_view_v1(&foreign, &nested, 0u, 0u, &nested) == 0);
    GRAPH_CHECK(fycs_graph_borrow_copy_v1(&foreign, &nested, &saved) == 0 && saved.length == 0u);
    GRAPH_CHECK(fycs_graph_write_v1(&foreign, b, NULL, 1u) == 0);
    GRAPH_CHECK(fycs_graph_borrow_copy_v1(&foreign, &nested, &saved) == 9);
    used = context.used;
    GRAPH_CHECK(fycs_graph_borrow_copy_v1(&context, &borrow, &saved) == 8 && context.used == used);
    context.generation = UINT64_MAX;
    GRAPH_CHECK(fycs_graph_write_v1(&context, a, NULL, 1u) == 11 && transaction.count == 2u);
    context.generation = 2u;
    GRAPH_CHECK(fycs_graph_step_v1(&context, 1u, 1u) == 0 && context.cost == 1u && context.operation == 1u);
    GRAPH_CHECK(fycs_graph_step_v1(&context, 1u, 0u) == 0 && context.cost == 2u && context.operation == 1u);
    GRAPH_CHECK(fycs_graph_step_v1(&context, 0u, 1u) == 0 && context.cost == 2u && context.operation == 2u);
    context.cost = UINT64_MAX;
    GRAPH_CHECK(fycs_graph_step_v1(&context, 1u, 1u) == 11 && context.operation == 2u);
    context.cost = 2u; context.operation = UINT64_MAX;
    GRAPH_CHECK(fycs_graph_step_v1(&context, 1u, 1u) == 11 && context.cost == 2u);
    GRAPH_CHECK(fycs_graph_step_v1(&context, 2u, 0u) == 1);
    value.bytes = (const uint8_t *)"x"; value.length = SIZE_MAX; value.codec = FYCS_CODEC_BYTES;
    GRAPH_CHECK(fycs_graph_map_key_v1(a, &value, SIZE_MAX, output) == 7);
    GRAPH_CHECK(fycs_graph_map_key_v1(NULL, &value, SIZE_MAX, output) == 1);
    GRAPH_CHECK(fycs_overlay_abort_v1(&transaction) == 0);
    GRAPH_CHECK(fycs_graph_read_v1(&context, a, FYCS_CODEC_BYTES, &saved, &found) == 2);
    GRAPH_CHECK(fycs_graph_write_v1(&context, a, NULL, 1u) == 2);
    GRAPH_CHECK(fycs_graph_borrow_copy_v1(&context, &borrow, &saved) == 2);
    GRAPH_CHECK(fycs_graph_step_v1(&context, 0u, 0u) == 2);
    return 0;
}

static int graph_probe_state(void) {
    fycs_overlay_v1 transaction = {0};
    fycs_overlay_entry_v1 writes[3];
    fycs_graph_epoch_v1 epochs[3];
    fycs_graph_v1 context = {0};
    fycs_graph_value_v1 alpha = {(const uint8_t *)"alpha",5u,FYCS_CODEC_BYTES};
    fycs_graph_value_v1 beta = {(const uint8_t *)"beta",4u,FYCS_CODEC_BYTES};
    fycs_graph_value_v1 next = {(const uint8_t *)"next",4u,FYCS_CODEC_BYTES};
    fycs_graph_value_v1 read = {0}, copy = {0};
    fycs_graph_borrow_v1 borrow, view, second;
    uint8_t arena[256], scratch[256], values[256], a[32], b[32], found = 0u, codec;
    int32_t status;
    if (fread(a, 1u, 32u, stdin) != 32u || fread(b, 1u, 32u, stdin) != 32u || fgetc(stdin) != EOF) { return 3; }
    GRAPH_CHECK(fycs_overlay_begin_v1(&transaction, NULL, 0u, writes, 3u, arena, scratch, sizeof(arena), 256u, 64u) == 0);
    GRAPH_CHECK(fycs_graph_begin_v1(&context, &transaction, values, sizeof(values), epochs, 3u) == 0);
    graph_probe_put((uint32_t)fycs_graph_write_v1(&context, a, &alpha, 0u), 1u);
    status = fycs_graph_read_v1(&context, a, FYCS_CODEC_BYTES, &read, &found);
    graph_probe_value(status, found, &read);
    graph_probe_put((uint32_t)fycs_graph_borrow_v1_begin(&context, a, FYCS_CODEC_BYTES, &borrow, &found), 1u);
    graph_probe_put(found, 1u);
    graph_probe_put((uint32_t)fycs_graph_borrow_view_v1(&context, &borrow, 1u, 3u, &view), 1u);
    graph_probe_put((uint32_t)fycs_graph_write_v1(&context, b, &beta, 0u), 1u);
    status = fycs_graph_borrow_copy_v1(&context, &borrow, &copy); graph_probe_value(status, 1u, &copy);
    status = fycs_graph_borrow_copy_v1(&context, &view, &copy); graph_probe_value(status, 1u, &copy);
    graph_probe_put((uint32_t)fycs_graph_write_v1(&context, a, &next, 0u), 1u);
    graph_probe_value(0, 1u, &read);
    graph_probe_put((uint32_t)fycs_graph_borrow_copy_v1(&context, &borrow, &copy), 1u);
    graph_probe_put((uint32_t)fycs_graph_borrow_copy_v1(&context, &view, &copy), 1u);
    graph_probe_put((uint32_t)fycs_graph_write_v1(&context, b, &read, 0u), 1u);
    graph_probe_put((uint32_t)fycs_graph_borrow_v1_begin(&context, b, FYCS_CODEC_BYTES, &second, &found), 1u);
    graph_probe_put((uint32_t)fycs_graph_write_v1(&context, a, NULL, 1u), 1u);
    for (codec = FYCS_CODEC_BOOL; codec <= FYCS_CODEC_BYTES32; ++codec) {
        status = fycs_graph_read_v1(&context, a, codec, &copy, &found);
        graph_probe_value(status, found, &copy);
    }
    status = fycs_graph_borrow_copy_v1(&context, &second, &copy); graph_probe_value(status, 1u, &copy);
    graph_probe_put((uint32_t)fycs_graph_write_v1(&context, b, &next, 0u), 1u);
    graph_probe_put((uint32_t)fycs_graph_borrow_copy_v1(&context, &second, &copy), 1u);
    GRAPH_CHECK(fycs_overlay_abort_v1(&transaction) == 0);
    return graph_probe_boundaries();
}

int main(void) {
    int mode, status;
    if (_setmode(_fileno(stdin), _O_BINARY) == -1 || _setmode(_fileno(stdout), _O_BINARY) == -1) { return 1; }
    mode = getchar();
    status = mode == 0 ? graph_probe_keys() : (mode == 1 ? graph_probe_state() : 2);
    if (fflush(stdout) != 0) { return 7; }
    return status;
}
