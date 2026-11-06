package contractstatenative

const overlayNativeFrameC = `
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>
#include <io.h>
#ifndef FYOV_ALLOC
#define FYOV_ALLOC malloc
#define FYOV_RELEASE free
#endif
#define FYOV_INPUT_BYTES 152u
#define FYOV_OUTPUT_BYTES 142u
#define FYOV_SNAPSHOT_LIMIT 16777216u
#define FYOV_ENTRY_LIMIT 65536u
#define FYOV_RESPONSE_LIMIT 67108864u

static uint8_t *fyov_trace;
static size_t fyov_trace_size, fyov_trace_capacity;
static uint32_t fyov_read_count;

static uint64_t fyov_number(const uint8_t *value, size_t size) {
    uint64_t result = 0u;
    size_t i;
    for (i = 0u; i < size; ++i) { result = (result << 8u) | value[i]; }
    return result;
}
static void fyov_put(uint8_t *output, uint64_t value, size_t size) {
    size_t i = size;
    while (i != 0u) { --i; output[i] = (uint8_t)(value & 255u); value >>= 8u; }
}
static int32_t fyov_observe(void *context, uint32_t operation, uint8_t codec,
    const uint8_t *value, size_t length, uint8_t found) {
    (void)context;
#if FYOV_READ_COUNT == 0
    (void)operation; (void)codec; (void)value; (void)length; (void)found;
    return FYCS_OV_INVALID;
#else
    uint8_t *header;
    if (fyov_read_count >= FYOV_READ_COUNT || operation != fyov_operations[fyov_read_count] ||
        codec != fyov_codecs[fyov_read_count] || found > 1u ||
        (found == 0u && length != 0u) || (value == NULL && length != 0u) ||
        length > fyov_trace_capacity - fyov_trace_size ||
        fyov_trace_capacity - fyov_trace_size - length < 10u) { return FYCS_OV_INVALID; }
    header = fyov_trace + fyov_trace_size;
    fyov_put(header, operation, 4u); header[4] = codec; header[5] = found;
    fyov_put(header + 6u, length, 4u);
    if (length != 0u) { memcpy(header + 10u, value, length); }
    fyov_trace_size += 10u + length;
    ++fyov_read_count;
    return FYCS_OV_OK;
#endif
}

int main(void) {
    uint8_t request[FYOV_INPUT_BYTES], response[FYOV_OUTPUT_BYTES] = {0};
    uint8_t *payload = NULL, *arena = NULL, *scratch = NULL;
    fycs_overlay_entry_v1 *snapshot = NULL, *writes = NULL;
    fycs_overlay_v1 transaction = {0};
    uint32_t count, payload_size, i;
    uint64_t policy_count, policy_bytes, policy_value, policy_key, trace_bound, write_payload = 0u;
    size_t at = 0u, maximum_value = FYOV_LITERAL_SIZE;
    uint8_t exit_kind = 0u, begun = 0u;
    int32_t status;
    int result = 4;
    if (_setmode(_fileno(stdin), _O_BINARY) == -1 || _setmode(_fileno(stdout), _O_BINARY) == -1 ||
        fread(request, 1u, sizeof(request), stdin) != sizeof(request)) { return 4; }
    if (memcmp(request, "FYOVIN01", 8u) != 0 || memcmp(request + 8u, fyov_identity, 96u) != 0) { return 4; }
    policy_count = fyov_number(request + 112u, 8u); policy_bytes = fyov_number(request + 120u, 8u);
    policy_value = fyov_number(request + 128u, 8u); policy_key = fyov_number(request + 136u, 8u);
    count = (uint32_t)fyov_number(request + 144u, 4u); payload_size = (uint32_t)fyov_number(request + 148u, 4u);
    if (policy_count == 0u || policy_bytes == 0u || policy_value == 0u || policy_key == 0u ||
        count > FYOV_ENTRY_LIMIT || payload_size > FYOV_SNAPSHOT_LIMIT ||
        (uint64_t)count * 36u > payload_size) { return 4; }
    payload = (uint8_t *)FYOV_ALLOC(payload_size != 0u ? payload_size : 1u);
    if (payload == NULL) { goto done; }
    snapshot = (fycs_overlay_entry_v1 *)FYOV_ALLOC((count != 0u ? count : 1u) * sizeof(*snapshot));
    if (snapshot == NULL) { goto done; }
    if (fread(payload, 1u, payload_size, stdin) != payload_size || fgetc(stdin) != EOF || ferror(stdin) != 0) { goto done; }
    memset(snapshot, 0, (count != 0u ? count : 1u) * sizeof(*snapshot));
    for (i = 0u; i < count; ++i) {
        size_t length;
        uint8_t codec;
        if (payload_size - at < 36u) { goto done; }
        memcpy(snapshot[i].key, payload + at, 32u);
        length = (size_t)fyov_number(payload + at + 32u, 4u); at += 36u;
        if (length > payload_size - at ||
            (i != 0u && memcmp(snapshot[i-1u].key, snapshot[i].key, 32u) >= 0)) { goto done; }
        snapshot[i].value = payload + at; snapshot[i].length = length;
        codec = fyov_codec(snapshot[i].key);
        if (codec != 0u) {
            if (fycs_codec_valid_v1(codec, snapshot[i].value, length) != FYCS_OV_OK) { goto done; }
            if (length > maximum_value) { maximum_value = length; }
        }
        at += length;
    }
    if (at != payload_size) { goto done; }
    writes = (fycs_overlay_entry_v1 *)FYOV_ALLOC(FYOV_WRITE_COUNT * sizeof(*writes));
    arena = (uint8_t *)FYOV_ALLOC(FYOV_WORKSPACE);
    scratch = (uint8_t *)FYOV_ALLOC(FYOV_WORKSPACE);
    if (writes == NULL || arena == NULL || scratch == NULL) { goto done; }
    trace_bound = (uint64_t)FYOV_READ_COUNT * (10u + (uint64_t)maximum_value);
    fyov_trace_capacity = trace_bound < FYOV_RESPONSE_LIMIT - FYOV_OUTPUT_BYTES ? (size_t)trace_bound : FYOV_RESPONSE_LIMIT - FYOV_OUTPUT_BYTES;
    fyov_trace = (uint8_t *)FYOV_ALLOC(fyov_trace_capacity != 0u ? fyov_trace_capacity : 1u);
    if (fyov_trace == NULL) { goto done; }
    /* The immutable literal plan bounds every possible overlay cost. Capping a
     * larger caller policy at that bound cannot reject any represented attempt.
     */
    status = fycs_overlay_begin_v1(&transaction, snapshot, count, writes,
        policy_count < FYOV_WRITE_COUNT ? (size_t)policy_count : FYOV_WRITE_COUNT,
        arena, scratch, FYOV_WORKSPACE, policy_bytes < FYOV_WORKSPACE ? policy_bytes : FYOV_WORKSPACE,
        (size_t)policy_value);
    if (status != FYCS_OV_OK) { goto done; }
    begun = 1u;
    status = fycs_overlay_execute_v1(&transaction, (size_t)policy_key, fyov_observe, NULL, &exit_kind);
    if (status < 0 || status > 7 || (status == 0 && (exit_kind != FYOV_EXPECTED_EXIT || fyov_read_count != FYOV_READ_COUNT))) { goto done; }
    if (status != 0 || exit_kind != 0u) {
        if (transaction.count != 0u || transaction.write_bytes != 0u) { goto done; }
    }
    for (i = 0u; i < transaction.count; ++i) { write_payload += 37u + transaction.writes[i].length; }
    if (write_payload > FYOV_RESPONSE_LIMIT - FYOV_OUTPUT_BYTES || fyov_trace_size > FYOV_RESPONSE_LIMIT - FYOV_OUTPUT_BYTES - write_payload) { goto done; }
    memcpy(response, "FYOVOUT1", 8u); memcpy(response + 8u, request + 8u, 104u);
    response[112] = (uint8_t)status; response[113] = exit_kind;
    fyov_put(response + 114u, fyov_read_count, 4u); fyov_put(response + 118u, transaction.count, 4u);
    fyov_put(response + 122u, transaction.write_bytes, 8u);
    fyov_put(response + 138u, fyov_trace_size + write_payload, 4u);
    if (fwrite(response, 1u, sizeof(response), stdout) != sizeof(response) ||
        fwrite(fyov_trace, 1u, fyov_trace_size, stdout) != fyov_trace_size) { goto done; }
    for (i = 0u; i < transaction.count; ++i) {
        const fycs_overlay_entry_v1 *entry = &transaction.writes[i];
        uint8_t header[37];
        memcpy(header, entry->key, 32u); header[32] = entry->deleted;
        fyov_put(header + 33u, entry->length, 4u);
        if (fwrite(header, 1u, sizeof(header), stdout) != sizeof(header) ||
            fwrite(entry->value, 1u, entry->length, stdout) != entry->length) { goto done; }
    }
    if (fflush(stdout) == 0) { result = 0; }
done:
    if (begun != 0u && transaction.state == 1u) { (void)fycs_overlay_abort_v1(&transaction); }
    FYOV_RELEASE(fyov_trace); FYOV_RELEASE(scratch); FYOV_RELEASE(arena);
    FYOV_RELEASE(writes); FYOV_RELEASE(snapshot); FYOV_RELEASE(payload);
    return result;
}
`
