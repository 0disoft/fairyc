//go:build ignore

#include <stdio.h>
#include <string.h>
#ifdef _WIN32
#include <fcntl.h>
#include <io.h>
#endif

static int probe_read(void *value, size_t size) { return fread(value, 1u, size, stdin) == size; }
static uint32_t probe_uint(size_t width) {
    uint32_t value = 0u;
    size_t i;
    for (i = 0u; i < width; ++i) {
        int next = getchar();
        if (next == EOF) { return UINT32_MAX; }
        value = (value << 8u) | (uint32_t)next;
    }
    return value;
}
static void probe_put(uint64_t value, size_t width) {
    size_t i = width;
    while (i != 0u) { --i; (void)putchar((int)((value >> (i * 8u)) & 255u)); }
}
static int probe_codecs(void) {
    static uint8_t value[65536];
    int codec;
    while ((codec = getchar()) != EOF) {
        uint32_t length = probe_uint(4u);
        if (length > sizeof(value) || !probe_read(value, length)) { return 10; }
        (void)putchar(fycs_codec_valid_v1((uint8_t)codec, value, length) == FYCS_OV_OK ? 1 : 0);
    }
    return ferror(stdin) != 0 || fflush(stdout) != 0 ? 11 : 0;
}

static int probe_overlay(void) {
    uint32_t cases = probe_uint(2u), trial;
    if (cases > 1000u) { return 20; }
    for (trial = 0u; trial < cases; ++trial) {
        fycs_overlay_v1 transaction;
        fycs_overlay_entry_v1 snapshot[16], writes[16], retained[16];
        uint8_t values[16][256], original[16][256], arena[4096], scratch[4096];
        uint32_t max_writes = probe_uint(1u), max_bytes = probe_uint(4u);
        uint32_t max_value = probe_uint(2u), count = probe_uint(1u), operations = probe_uint(1u), i;
        const uint8_t *last = NULL;
        size_t last_length = 0u;
        if (max_writes > 16u || count > 16u || operations > 255u || max_bytes > sizeof(arena)) { return 21; }
        memset(snapshot, 0, sizeof(snapshot));
        memset(values, 0, sizeof(values));
        for (i = 0u; i < count; ++i) {
            size_t size;
            if (!probe_read(snapshot[i].key, 32u)) { return 22; }
            size = probe_uint(2u);
            if (size > sizeof(values[i]) || !probe_read(values[i], size)) { return 23; }
            snapshot[i].value = values[i];
            snapshot[i].length = size;
        }
        memcpy(retained, snapshot, sizeof(snapshot));
        memcpy(original, values, sizeof(values));
        if (fycs_overlay_begin_v1(&transaction, snapshot, count, writes, max_writes,
            arena, scratch, sizeof(arena), max_bytes, max_value) != FYCS_OV_OK) { return 24; }
        for (i = 0u; i < operations; ++i) {
            uint32_t kind = probe_uint(1u);
            uint8_t key[32], value[256], found = 0u;
            size_t size, observed_length = 0u;
            const uint8_t *observed = NULL;
            int32_t status;
            if (!probe_read(key, sizeof(key))) { return 25; }
            size = probe_uint(2u);
            if (size > sizeof(value) || !probe_read(value, size)) { return 26; }
            if (kind == 1u) {
                status = fycs_overlay_read_v1(&transaction, key, &observed, &observed_length, &found);
                last = observed;
                last_length = observed_length;
            } else if (kind == 2u || kind == 3u) {
                status = fycs_overlay_write_v1(&transaction, key, value, size, kind == 3u ? 1u : 0u);
            } else if (kind == 4u) {
                status = fycs_overlay_write_v1(&transaction, key, last, last_length, 0u);
            } else { return 27; }
            probe_put((uint32_t)status, 1u);
            probe_put(found, 1u);
            probe_put(observed_length, 2u);
            if (observed_length != 0u && fwrite(observed, 1u, observed_length, stdout) != observed_length) { return 28; }
            probe_put(transaction.count, 2u);
            probe_put(transaction.write_bytes, 4u);
        }
        if (memcmp(retained, snapshot, sizeof(snapshot)) != 0 || memcmp(original, values, sizeof(values)) != 0) { return 29; }
        if (fycs_overlay_commit_v1(&transaction) != FYCS_OV_OK) { return 30; }
        probe_put(transaction.count, 2u);
        probe_put(transaction.write_bytes, 4u);
        for (i = 0u; i < transaction.count; ++i) {
            const fycs_overlay_entry_v1 *entry = &transaction.writes[i];
            if (fwrite(entry->key, 1u, 32u, stdout) != 32u) { return 31; }
            probe_put(entry->deleted, 1u);
            probe_put(entry->length, 2u);
            if (entry->length != 0u && fwrite(entry->value, 1u, entry->length, stdout) != entry->length) { return 32; }
        }
        {
            uint8_t key[32] = {0}, found;
            const uint8_t *value;
            size_t length;
            if (fycs_overlay_read_v1(&transaction, key, &value, &length, &found) != FYCS_OV_CLOSED ||
                fycs_overlay_write_v1(&transaction, key, NULL, 0u, 1u) != FYCS_OV_CLOSED ||
                fycs_overlay_commit_v1(&transaction) != FYCS_OV_CLOSED ||
                fycs_overlay_abort_v1(&transaction) != FYCS_OV_CLOSED) { return 33; }
        }
    }
    return getchar() != EOF || ferror(stdin) != 0 || fflush(stdout) != 0 ? 34 : 0;
}

static int probe_failures(void) {
    fycs_overlay_v1 transaction;
    fycs_overlay_entry_v1 snapshot[2] = {{{1}, NULL, 0u, 0u}, {{1}, NULL, 0u, 0u}}, writes[2];
    uint8_t arena[128], scratch[128], key[32] = {0}, value[3] = {1u, 2u, 3u}, found;
    const uint8_t *read_value;
    size_t read_length;
    memset(&transaction, 0x5a, sizeof(transaction));
    if (fycs_overlay_begin_v1(&transaction, snapshot, 2u, writes, 2u, arena, scratch, 128u, 128u, 3u) != FYCS_OV_INVALID || transaction.state != 0x5au) { return 40; }
    snapshot[1].key[0] = 0u;
    if (fycs_overlay_begin_v1(&transaction, snapshot, 2u, writes, 2u, arena, scratch, 128u, 128u, 3u) != FYCS_OV_INVALID) { return 41; }
    snapshot[1].key[0] = 2u;
    snapshot[0].deleted = 1u;
    if (fycs_overlay_begin_v1(&transaction, snapshot, 2u, writes, 2u, arena, scratch, 128u, 128u, 3u) != FYCS_OV_INVALID) { return 42; }
    if (fycs_overlay_begin_v1(&transaction, NULL, 0u, writes, 2u, arena, arena, 128u, 128u, 3u) != FYCS_OV_INVALID ||
        fycs_overlay_begin_v1(&transaction, NULL, 0u, writes, 2u, arena, scratch, 128u, 129u, 3u) != FYCS_OV_INVALID) { return 43; }
    if (fycs_codec_valid_v1(FYCS_CODEC_BYTES, NULL, 1u) != FYCS_OV_INVALID ||
        fycs_codec_valid_v1(FYCS_CODEC_UTF8, NULL, 0u) != FYCS_OV_OK) { return 44; }
    if (fycs_overlay_begin_v1(&transaction, NULL, 0u, writes, 2u, arena, scratch, 128u, 128u, 3u) != FYCS_OV_OK) { return 45; }
    if (fycs_overlay_write_v1(&transaction, key, value, 3u, 0u) != FYCS_OV_OK ||
        fycs_overlay_write_v1(&transaction, key, value, 1u, 1u) != FYCS_OV_INVALID || transaction.write_bytes != 36u) { return 46; }
    /* Key and value inputs can refer to the current overlay itself. */
    if (fycs_overlay_write_v1(&transaction, transaction.writes[0].key, transaction.writes[0].value, 3u, 0u) != FYCS_OV_OK) { return 47; }
    if (fycs_overlay_read_v1(&transaction, key, &read_value, &read_length, &found) != FYCS_OV_OK || found != 1u || read_length != 3u || memcmp(read_value, value, 3u) != 0) { return 48; }
    transaction.max_value = SIZE_MAX;
    if (fycs_overlay_write_v1(&transaction, key, value, SIZE_MAX, 0u) != FYCS_OV_BYTE_LIMIT || transaction.write_bytes != 36u) { return 52; }
    if (fycs_overlay_abort_v1(&transaction) != FYCS_OV_OK || transaction.count != 0u || transaction.write_bytes != 0u) { return 49; }
    {
        size_t i;
        for (i = 0u; i < sizeof(arena); ++i) { if (arena[i] != 0u || scratch[i] != 0u) { return 50; } }
    }
    if (fycs_overlay_abort_v1(&transaction) != FYCS_OV_CLOSED || fycs_overlay_read_v1(&transaction, key, &read_value, &read_length, &found) != FYCS_OV_CLOSED) { return 51; }
    return 0;
}

int main(void) {
    int mode;
#ifdef _WIN32
    if (_setmode(_fileno(stdin), _O_BINARY) == -1 || _setmode(_fileno(stdout), _O_BINARY) == -1) { return 2; }
#endif
    mode = getchar();
    if (mode == 0) { return probe_codecs(); }
    if (mode == 1) { return probe_overlay(); }
    if (mode == 2) { return probe_failures(); }
    return 3;
}
