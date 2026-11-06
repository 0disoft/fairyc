package contractstatenative

// Private process framing; these bytes do not define an SDK or public runtime ABI.
const persistentTransportC = `
#include <stdio.h>
#include <string.h>
#include <io.h>
#include <fcntl.h>

#define FYCS_PERSISTENT_INPUT_SIZE_V1 137u
#define FYCS_PERSISTENT_OUTPUT_SIZE_V1 135u
#define FYCS_PERSISTENT_READ_SIZE_V1 13u
#define FYCS_PERSISTENT_WRITE_BYTES_V1 41u
#define FYCS_PERSISTENT_DELETE_BYTES_V1 33u

typedef struct fycs_persistent_read_v1 {
    uint32_t operation;
    uint8_t found;
    uint8_t value[8];
} fycs_persistent_read_v1;

static fycs_persistent_read_v1 fycs_persistent_reads_v1[FYCS_PERSISTENT_READ_CAPACITY_V1];
static uint32_t fycs_persistent_read_count_v1 = 0u;
static uint8_t fycs_persistent_dirty_v1 = 0u;
static uint64_t fycs_persistent_write_limit_v1 = 0u;
static uint64_t fycs_persistent_value_limit_v1 = 0u;

static uint64_t fycs_persistent_u64_v1(const uint8_t *value) {
    uint64_t result = 0u;
    for (size_t i = 0u; i < 8u; ++i) { result = (result << 8u) | value[i]; }
    return result;
}

static void fycs_persistent_put_u32_v1(uint8_t *output, uint32_t value) {
    for (size_t i = 0u; i < 4u; ++i) {
        output[3u-i] = (uint8_t)(value & 255u); value >>= 8u;
    }
}

int32_t fycs_persistent_before_v1(uint8_t kind) {
    if (kind == 2u) {
        if (fycs_persistent_value_limit_v1 < 8u) { return -25; }
        if (fycs_persistent_write_limit_v1 < FYCS_PERSISTENT_WRITE_BYTES_V1) { return -24; }
    } else if (kind == 3u && fycs_persistent_write_limit_v1 < FYCS_PERSISTENT_DELETE_BYTES_V1) {
        return -24;
    }
    return 0;
}

void fycs_persistent_written_v1(void) { fycs_persistent_dirty_v1 = 1u; }

int32_t fycs_persistent_observe_v1(uint32_t operation, const uint8_t *value, size_t length, uint8_t found) {
    if (found > 1u || length != (found != 0u ? 8u : 0u) ||
        (found != 0u && value == NULL) || fycs_persistent_read_count_v1 >= FYCS_PERSISTENT_READ_CAPACITY_V1) {
        return -1;
    }
    fycs_persistent_read_v1 *read = &fycs_persistent_reads_v1[fycs_persistent_read_count_v1++];
    read->operation = operation;
    read->found = found;
    if (found != 0u) { memcpy(read->value, value, 8u); }
    return 0;
}

int main(void) {
    uint8_t request[FYCS_PERSISTENT_INPUT_SIZE_V1];
    static uint8_t response[FYCS_PERSISTENT_OUTPUT_SIZE_V1 + FYCS_PERSISTENT_READ_SIZE_V1 * FYCS_PERSISTENT_READ_CAPACITY_V1] = {0};
    if (_setmode(_fileno(stdin), _O_BINARY) < 0 || _setmode(_fileno(stdout), _O_BINARY) < 0) { return 4; }
    if (fread(request, 1u, sizeof(request), stdin) != sizeof(request) || fgetc(stdin) != EOF || ferror(stdin)) { return 4; }
    if (memcmp(request, "FYSPIN01", 8u) != 0 || memcmp(request+8u, fycs_native_layout_hash_v0, 32u) != 0 ||
        memcmp(request+40u, fycs_native_body_hash_v0, 32u) != 0 ||
        memcmp(request+72u, fycs_persistent_program_hash_v1, 32u) != 0 || request[128] > 1u) { return 4; }
    if (request[128] == 0u) {
        for (size_t i = 129u; i < sizeof(request); ++i) { if (request[i] != 0u) { return 4; } }
    }
    fycs_persistent_write_limit_v1 = fycs_persistent_u64_v1(request+112u);
    fycs_persistent_value_limit_v1 = fycs_persistent_u64_v1(request+120u);
    if (fycs_persistent_write_limit_v1 == 0u || fycs_persistent_value_limit_v1 == 0u) { return 4; }
    fycs_native_store_v0 store;
    fycs_native_store_initialize_v0(&store, fycs_native_layout_hash_v0, fycs_native_body_hash_v0);
    store.present = request[128];
    if (store.present != 0u) { store.length = 8u; memcpy(store.value, request+129u, 8u); }
    const int32_t status = fycs_native_entry_v0(&store);
    uint8_t wire_status = 5u;
    if (status >= 0 && status <= 2) { wire_status = (uint8_t)status; }
    else if (status == -24) { wire_status = 3u; }
    else if (status == -25) { wire_status = 4u; }
    if (store.present > 1u || store.length != (store.present != 0u ? 8u : 0u) || fycs_native_live_resources_v0() != 0u) { return 4; }
    memcpy(response, "FYSPOU01", 8u);
    memcpy(response+8u, request+8u, 104u);
    response[112] = wire_status;
    response[113] = status == 0 ? fycs_persistent_dirty_v1 : 0u;
    response[114] = store.present;
    if (store.present != 0u) { memcpy(response+115u, store.value, 8u); }
    fycs_persistent_put_u32_v1(response+123u, fycs_persistent_read_count_v1);
    /* Legacy straight-line execution has zero reported cost, as in the Go oracle. */
    for (uint32_t i = 0u; i < fycs_persistent_read_count_v1; ++i) {
        uint8_t *out = response+FYCS_PERSISTENT_OUTPUT_SIZE_V1+(size_t)i*FYCS_PERSISTENT_READ_SIZE_V1;
        fycs_persistent_put_u32_v1(out, fycs_persistent_reads_v1[i].operation);
        out[4] = fycs_persistent_reads_v1[i].found;
        memcpy(out+5u, fycs_persistent_reads_v1[i].value, 8u);
    }
    const size_t length = FYCS_PERSISTENT_OUTPUT_SIZE_V1 + (size_t)fycs_persistent_read_count_v1 * FYCS_PERSISTENT_READ_SIZE_V1;
    return fwrite(response, 1u, length, stdout) == length && fflush(stdout) == 0 ? 0 : 4;
}
`
