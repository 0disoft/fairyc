//go:build ignore

#include "stage0_contract_state_graph.h"
#include <string.h>

static const uint8_t fycs_graph_zero_v1[32] = {0};

static int32_t fycs_graph_open_v1(const fycs_graph_v1 *context) {
    if (context == NULL || context->transaction == NULL) { return FYCS_OV_INVALID; }
    return context->transaction->state == 1u ? FYCS_OV_OK : FYCS_OV_CLOSED;
}

int32_t fycs_graph_begin_v1(fycs_graph_v1 *context, fycs_overlay_v1 *transaction,
    uint8_t *values, size_t capacity, fycs_graph_epoch_v1 *epochs, size_t epoch_capacity) {
    if (context == NULL || transaction == NULL || (values == NULL && capacity != 0u) ||
        epochs == NULL || epoch_capacity < transaction->max_writes) { return FYCS_OV_INVALID; }
    if (transaction->state != 1u) { return FYCS_OV_CLOSED; }
    memset(context, 0, sizeof(*context));
    context->transaction = transaction; context->values = values; context->capacity = capacity;
    context->epochs = epochs; context->epoch_capacity = epoch_capacity;
    return FYCS_OV_OK;
}

static int32_t fycs_graph_copy_v1(fycs_graph_v1 *context, uint8_t codec,
    const uint8_t *bytes, size_t length, fycs_graph_value_v1 *output) {
    fycs_graph_value_v1 value;
    int32_t status = fycs_graph_open_v1(context);
    if (status != FYCS_OV_OK) { return status; }
    if (output == NULL || (bytes == NULL && length != 0u)) { return FYCS_OV_INVALID; }
    if (context->used > context->capacity || length > context->capacity - context->used) { return FYCS_GRAPH_VALUE_SPACE; }
    value.codec = codec; value.length = length; value.bytes = fycs_graph_zero_v1;
    if (length != 0u) {
        value.bytes = context->values + context->used;
        memmove(context->values + context->used, bytes, length);
        context->used += length;
    }
    *output = value;
    return FYCS_OV_OK;
}

int32_t fycs_graph_value_copy_v1(fycs_graph_v1 *context, uint8_t codec,
    const uint8_t *bytes, size_t length, fycs_graph_value_v1 *output) {
    int32_t status = fycs_codec_valid_v1(codec, bytes, length);
    if (status != FYCS_OV_OK) { return status; }
    return fycs_graph_copy_v1(context, codec, bytes, length, output);
}

int32_t fycs_graph_read_v1(fycs_graph_v1 *context, const uint8_t key[32], uint8_t codec,
    fycs_graph_value_v1 *output, uint8_t *found) {
    const uint8_t *bytes = NULL;
    size_t length = 0u;
    uint8_t present = 0u;
    int32_t status = fycs_graph_open_v1(context);
    if (status != FYCS_OV_OK) { return status; }
    if (output == NULL || found == NULL) { return FYCS_OV_INVALID; }
    status = fycs_overlay_read_v1(context->transaction, key, &bytes, &length, &present);
    if (status != FYCS_OV_OK) { return status; }
    if (present == 0u) {
        switch (codec) {
        case FYCS_CODEC_BOOL: length = 1u; break;
        case FYCS_CODEC_U32: length = 4u; break;
        case FYCS_CODEC_U64: length = 8u; break;
        case FYCS_CODEC_BYTES20: length = 20u; break;
        case FYCS_CODEC_BYTES32: length = 32u; break;
        case FYCS_CODEC_BYTES: case FYCS_CODEC_UTF8: length = 0u; break;
        default: return FYCS_OV_CODEC;
        }
        output->bytes = fycs_graph_zero_v1; output->length = length; output->codec = codec;
    } else {
        status = fycs_graph_value_copy_v1(context, codec, bytes, length, output);
        if (status != FYCS_OV_OK) { return status; }
    }
    *found = present;
    return FYCS_OV_OK;
}

static size_t fycs_graph_epoch_index_v1(const fycs_graph_v1 *context, const uint8_t key[32]) {
    size_t low = 0u, high = context->epoch_count;
    while (low < high) {
        size_t middle = low + (high-low)/2u;
        if (memcmp(context->epochs[middle].key, key, 32u) < 0) { low = middle+1u; }
        else { high = middle; }
    }
    return low;
}

static uint64_t fycs_graph_generation_v1(const fycs_graph_v1 *context, const uint8_t key[32]) {
    size_t index = fycs_graph_epoch_index_v1(context, key);
    return index < context->epoch_count && memcmp(context->epochs[index].key, key, 32u) == 0 ? context->epochs[index].generation : 0u;
}

int32_t fycs_graph_write_v1(fycs_graph_v1 *context, const uint8_t key[32],
    const fycs_graph_value_v1 *value, uint8_t deleted) {
    size_t index;
    uint8_t fresh;
    int32_t status = fycs_graph_open_v1(context);
    if (status != FYCS_OV_OK) { return status; }
    if (key == NULL || deleted > 1u || (deleted == 0u && value == NULL)) { return FYCS_OV_INVALID; }
    if (deleted == 0u) {
        status = fycs_codec_valid_v1(value->codec, value->bytes, value->length);
        if (status != FYCS_OV_OK) { return status; }
    }
    if (context->generation == UINT64_MAX) { return FYCS_GRAPH_STEP_OVERFLOW; }
    index = fycs_graph_epoch_index_v1(context, key);
    fresh = (uint8_t)(index == context->epoch_count || memcmp(context->epochs[index].key, key, 32u) != 0);
    if (fresh != 0u && context->epoch_count == context->epoch_capacity) { return FYCS_OV_WRITE_LIMIT; }
    status = fycs_overlay_write_v1(context->transaction, key,
        deleted != 0u ? NULL : value->bytes, deleted != 0u ? 0u : value->length, deleted);
    if (status != FYCS_OV_OK) { return status; }
    if (fresh != 0u) {
        memmove(context->epochs + index+1u, context->epochs + index,
            (context->epoch_count-index) * sizeof(*context->epochs));
        ++context->epoch_count;
        memcpy(context->epochs[index].key, key, 32u);
    }
    context->epochs[index].generation = ++context->generation;
    return FYCS_OV_OK;
}

int32_t fycs_graph_borrow_v1_begin(fycs_graph_v1 *context, const uint8_t key[32], uint8_t codec,
    fycs_graph_borrow_v1 *output, uint8_t *found) {
    fycs_graph_borrow_v1 borrow;
    uint8_t present;
    int32_t status;
    if (output == NULL || found == NULL) { return FYCS_OV_INVALID; }
    status = fycs_graph_read_v1(context, key, codec, &borrow.value, &present);
    if (status != FYCS_OV_OK) { return status; }
    borrow.owner = present != 0u ? context : NULL;
    memcpy(borrow.key, key, 32u);
    borrow.generation = fycs_graph_generation_v1(context, key);
    *output = borrow; *found = present;
    return FYCS_OV_OK;
}

static int32_t fycs_graph_borrow_valid_v1(fycs_graph_v1 *context, const fycs_graph_borrow_v1 *borrow) {
    int32_t status = fycs_graph_open_v1(context);
    if (status != FYCS_OV_OK) { return status; }
    if (borrow == NULL || borrow->owner != context ||
        borrow->generation != fycs_graph_generation_v1(context, borrow->key)) { return FYCS_GRAPH_BORROW_INVALID; }
    return FYCS_OV_OK;
}

int32_t fycs_graph_borrow_view_v1(fycs_graph_v1 *context, const fycs_graph_borrow_v1 *input,
    size_t offset, size_t length, fycs_graph_borrow_v1 *output) {
    fycs_graph_borrow_v1 next;
    int32_t status = fycs_graph_borrow_valid_v1(context, input);
    if (status != FYCS_OV_OK) { return status; }
    if (output == NULL) { return FYCS_OV_INVALID; }
    if (offset > input->value.length || length > input->value.length-offset) { return FYCS_GRAPH_VIEW_RANGE; }
    next = *input; next.value.bytes += offset; next.value.length = length;
    *output = next;
    return FYCS_OV_OK;
}

int32_t fycs_graph_borrow_copy_v1(fycs_graph_v1 *context, const fycs_graph_borrow_v1 *borrow,
    fycs_graph_value_v1 *output) {
    int32_t status = fycs_graph_borrow_valid_v1(context, borrow);
    if (status != FYCS_OV_OK) { return status; }
    /* A byte view can be noncanonical for the original codec until consumed by
     * a typed write/key. Match the Go borrow copy instead of validating early.
     */
    return fycs_graph_copy_v1(context, borrow->value.codec, borrow->value.bytes, borrow->value.length, output);
}

int32_t fycs_graph_step_v1(fycs_graph_v1 *context, uint8_t charge, uint8_t operation) {
    int32_t status = fycs_graph_open_v1(context);
    if (status != FYCS_OV_OK) { return status; }
    if (charge > 1u || operation > 1u) { return FYCS_OV_INVALID; }
    if ((charge != 0u && context->cost == UINT64_MAX) ||
        (operation != 0u && context->operation == UINT64_MAX)) { return FYCS_GRAPH_STEP_OVERFLOW; }
    context->cost += charge; context->operation += operation;
    return FYCS_OV_OK;
}
