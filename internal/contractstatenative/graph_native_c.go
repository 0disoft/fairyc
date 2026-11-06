package contractstatenative

const graphNativeFrameC = `
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>
#include <io.h>
#ifndef FYGR_ALLOC
#define FYGR_ALLOC malloc
#define FYGR_RELEASE free
#endif
#define FYGR_INPUT_BYTES 152u
#define FYGR_OUTPUT_BYTES 186u
#define FYGR_SNAPSHOT_LIMIT 16777216u
#define FYGR_ENTRY_LIMIT 65536u
#define FYGR_RESPONSE_LIMIT 67108864u

typedef struct fygr_trace {
    uint8_t *bytes;
    size_t used, capacity;
    uint32_t count;
} fygr_trace;

static uint64_t fygr_number(const uint8_t *value, size_t size) {
    uint64_t result = 0u; size_t i;
    for (i = 0u; i < size; ++i) { result = (result<<8u) | value[i]; }
    return result;
}
static void fygr_put(uint8_t *output, uint64_t value, size_t size) {
    size_t i = size;
    while (i != 0u) { --i; output[i] = (uint8_t)(value & 255u); value >>= 8u; }
}
static uint64_t fygr_min(uint64_t a, uint64_t b) { return a < b ? a : b; }
static int32_t fygr_observe(void *user, uint64_t operation, uint32_t site,
    const fycs_graph_value_v1 *key, const fycs_graph_value_v1 *value, uint8_t found) {
    fygr_trace *trace = (fygr_trace *)user;
    size_t key_size = key != NULL ? key->length : 0u;
    size_t value_size = value != NULL ? value->length : 0u;
    uint8_t *header;
    if (trace == NULL || found > 1u || (key_size != 0u && key->bytes == NULL) ||
        (value_size != 0u && value->bytes == NULL) ||
        key_size > trace->capacity-trace->used ||
        value_size > trace->capacity-trace->used-key_size ||
        trace->capacity-trace->used-key_size-value_size < 21u || trace->count == UINT32_MAX) { return FYCS_OV_INVALID; }
    header = trace->bytes+trace->used;
    fygr_put(header,operation,8u); fygr_put(header+8u,site,4u);
    fygr_put(header+12u,key_size,4u); fygr_put(header+16u,value_size,4u); header[20] = found;
    if (key_size != 0u) { memcpy(header+21u,key->bytes,key_size); }
    if (value_size != 0u) { memcpy(header+21u+key_size,value->bytes,value_size); }
    trace->used += 21u+key_size+value_size; ++trace->count;
    return FYCS_OV_OK;
}

int main(void) {
    uint8_t request[FYGR_INPUT_BYTES], response[FYGR_OUTPUT_BYTES] = {0};
    uint8_t *payload = NULL, *arena = NULL, *scratch = NULL, *values = NULL;
    fycs_overlay_entry_v1 *snapshot = NULL, *writes = NULL;
    fycs_graph_epoch_v1 *epochs = NULL;
    fycs_overlay_v1 transaction = {0}; fycs_graph_v1 context = {0};
    fygr_trace trace = {0};
    uint32_t count, payload_size, i, trap_kind = 0u;
    uint64_t policy_count, policy_bytes, policy_value, policy_key, bound, write_payload = 0u;
    size_t at = 0u, maximum_value = FYGR_LITERAL > 32u ? FYGR_LITERAL : 32u;
    size_t write_count, workspace, value_space;
    uint8_t exit_kind = 0u, begun = 0u;
    int32_t status;
    int result = 4;
    if (_setmode(_fileno(stdin),_O_BINARY) == -1 || _setmode(_fileno(stdout),_O_BINARY) == -1 ||
        fread(request,1u,sizeof(request),stdin) != sizeof(request)) { return 4; }
    if (memcmp(request,"FYGRIN01",8u) != 0 || memcmp(request+8u,fygr_identity,96u) != 0) { return 4; }
    policy_count = fygr_number(request+112u,8u); policy_bytes = fygr_number(request+120u,8u);
    policy_value = fygr_number(request+128u,8u); policy_key = fygr_number(request+136u,8u);
    count = (uint32_t)fygr_number(request+144u,4u); payload_size = (uint32_t)fygr_number(request+148u,4u);
    if (policy_count == 0u || policy_bytes == 0u || policy_value == 0u || policy_key == 0u ||
        count > FYGR_ENTRY_LIMIT || payload_size > FYGR_SNAPSHOT_LIMIT ||
        (uint64_t)count*36u > payload_size) { return 4; }
    payload = (uint8_t *)FYGR_ALLOC(payload_size != 0u ? payload_size : 1u);
    if (payload == NULL) { goto done; }
    snapshot = (fycs_overlay_entry_v1 *)FYGR_ALLOC((count != 0u ? count : 1u)*sizeof(*snapshot));
    if (snapshot == NULL) { goto done; }
    if (fread(payload,1u,payload_size,stdin) != payload_size || fgetc(stdin) != EOF || ferror(stdin) != 0) { goto done; }
    memset(snapshot,0,(count != 0u ? count : 1u)*sizeof(*snapshot));
    for (i = 0u; i < count; ++i) {
        size_t length; uint8_t codec;
        if (payload_size-at < 36u) { goto done; }
        memcpy(snapshot[i].key,payload+at,32u);
        length = (size_t)fygr_number(payload+at+32u,4u); at += 36u;
        if (length > payload_size-at || (i != 0u && memcmp(snapshot[i-1u].key,snapshot[i].key,32u) >= 0)) { goto done; }
        snapshot[i].value = payload+at; snapshot[i].length = length;
        codec = fygr_codec(snapshot[i].key);
        if (codec != 0u && fycs_codec_valid_v1(codec,snapshot[i].value,length) != FYCS_OV_OK) { goto done; }
        if (length > maximum_value) { maximum_value = length; }
        at += length;
    }
    if (at != payload_size || fycs_graph_digest_v1(request,sizeof(request),payload,payload_size,response+112u) != FYCS_OV_OK) { goto done; }
    write_count = (size_t)fygr_min(policy_count,fygr_min(FYGR_WORK,FYGR_RESPONSE_LIMIT/33u));
    bound = (uint64_t)FYGR_WORK*(33u+(uint64_t)maximum_value);
    workspace = (size_t)fygr_min(policy_bytes,fygr_min(bound,FYGR_RESPONSE_LIMIT));
    value_space = (size_t)fygr_min((uint64_t)FYGR_WORK*maximum_value,FYGR_RESPONSE_LIMIT);
    trace.capacity = (size_t)fygr_min((uint64_t)FYGR_WORK*(21u+2u*(uint64_t)maximum_value),FYGR_RESPONSE_LIMIT-FYGR_OUTPUT_BYTES);
    writes = (fycs_overlay_entry_v1 *)FYGR_ALLOC(write_count*sizeof(*writes));
    epochs = (fycs_graph_epoch_v1 *)FYGR_ALLOC(write_count*sizeof(*epochs));
    arena = (uint8_t *)FYGR_ALLOC(workspace); scratch = (uint8_t *)FYGR_ALLOC(workspace);
    values = (uint8_t *)FYGR_ALLOC(value_space); trace.bytes = (uint8_t *)FYGR_ALLOC(trace.capacity);
    if (writes == NULL || epochs == NULL || arena == NULL || scratch == NULL || values == NULL || trace.bytes == NULL) { goto done; }
    status = fycs_overlay_begin_v1(&transaction,snapshot,count,writes,write_count,arena,scratch,
        workspace,fygr_min(policy_bytes,workspace),(size_t)policy_value);
    if (status != FYCS_OV_OK) { goto done; }
    begun = 1u;
    status = fycs_graph_begin_v1(&context,&transaction,values,value_space,epochs,write_count);
    if (status != FYCS_OV_OK) { goto done; }
    status = fycs_graph_execute_v1(&context,(size_t)policy_key,fygr_observe,&trace,&exit_kind,&trap_kind);
    if (status < 0 || status > 11 || (status == 0 && (exit_kind > 2u || trap_kind > FYGR_TRAP_COUNT)) ||
        (status == FYCS_OV_WRITE_LIMIT && policy_count > write_count) ||
        (status == FYCS_OV_BYTE_LIMIT && policy_bytes > workspace)) { goto done; }
    if (status != 0 || exit_kind != 0u) {
        if (transaction.count != 0u || transaction.write_bytes != 0u) { goto done; }
    }
    for (i = 0u; i < transaction.count; ++i) { write_payload += 37u+transaction.writes[i].length; }
    if (write_payload > FYGR_RESPONSE_LIMIT-FYGR_OUTPUT_BYTES || trace.used > FYGR_RESPONSE_LIMIT-FYGR_OUTPUT_BYTES-write_payload) { goto done; }
    memcpy(response,"FYGROUT1",8u); memcpy(response+8u,request+8u,104u);
    response[144] = (uint8_t)status; response[145] = exit_kind;
    fygr_put(response+146u,trap_kind,4u); fygr_put(response+150u,context.cost,8u);
    fygr_put(response+158u,context.operation,8u); fygr_put(response+166u,trace.count,4u);
    fygr_put(response+170u,transaction.count,4u); fygr_put(response+174u,transaction.write_bytes,8u);
    fygr_put(response+182u,trace.used+write_payload,4u);
    if (fwrite(response,1u,sizeof(response),stdout) != sizeof(response) || fwrite(trace.bytes,1u,trace.used,stdout) != trace.used) { goto done; }
    for (i = 0u; i < transaction.count; ++i) {
        const fycs_overlay_entry_v1 *entry = &transaction.writes[i]; uint8_t header[37];
        memcpy(header,entry->key,32u); header[32] = entry->deleted;
        fygr_put(header+33u,entry->length,4u);
        if (fwrite(header,1u,sizeof(header),stdout) != sizeof(header) || fwrite(entry->value,1u,entry->length,stdout) != entry->length) { goto done; }
    }
    if (fflush(stdout) == 0) { result = 0; }
done:
    if (begun != 0u && transaction.state == 1u) { (void)fycs_overlay_abort_v1(&transaction); }
    FYGR_RELEASE(trace.bytes); FYGR_RELEASE(values); FYGR_RELEASE(scratch); FYGR_RELEASE(arena);
    FYGR_RELEASE(epochs); FYGR_RELEASE(writes); FYGR_RELEASE(snapshot); FYGR_RELEASE(payload);
    return result;
}
`
