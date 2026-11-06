//go:build ignore

#include "stage0_contract_state_overlay.h"
#include <string.h>

static int32_t fycs_utf8_valid_v1(const uint8_t *bytes, size_t length) {
    size_t i = 0u;
    while (i < length) {
        uint8_t lead = bytes[i++];
        uint8_t low = 0x80u, high = 0xbfu;
        size_t rest;
        if (lead < 0x80u) { continue; }
        if (lead >= 0xc2u && lead <= 0xdfu) { rest = 1u; }
        else if (lead >= 0xe0u && lead <= 0xefu) {
            rest = 2u;
            if (lead == 0xe0u) { low = 0xa0u; }
            if (lead == 0xedu) { high = 0x9fu; }
        } else if (lead >= 0xf0u && lead <= 0xf4u) {
            rest = 3u;
            if (lead == 0xf0u) { low = 0x90u; }
            if (lead == 0xf4u) { high = 0x8fu; }
        } else { return 0; }
        if (rest > length - i || bytes[i] < low || bytes[i] > high) { return 0; }
        ++i;
        --rest;
        while (rest != 0u) {
            if (bytes[i] < 0x80u || bytes[i] > 0xbfu) { return 0; }
            ++i;
            --rest;
        }
    }
    return 1;
}

int32_t fycs_codec_valid_v1(uint8_t codec, const uint8_t *bytes, size_t length) {
    if (bytes == NULL && length != 0u) { return FYCS_OV_INVALID; }
    switch (codec) {
    case FYCS_CODEC_BOOL:
        return length == 1u && bytes[0] <= 1u ? FYCS_OV_OK : FYCS_OV_CODEC;
    case FYCS_CODEC_U32: return length == 4u ? FYCS_OV_OK : FYCS_OV_CODEC;
    case FYCS_CODEC_U64: return length == 8u ? FYCS_OV_OK : FYCS_OV_CODEC;
    case FYCS_CODEC_BYTES: return FYCS_OV_OK;
    case FYCS_CODEC_UTF8:
        return fycs_utf8_valid_v1(bytes, length) != 0 ? FYCS_OV_OK : FYCS_OV_CODEC;
    case FYCS_CODEC_BYTES20: return length == 20u ? FYCS_OV_OK : FYCS_OV_CODEC;
    case FYCS_CODEC_BYTES32: return length == 32u ? FYCS_OV_OK : FYCS_OV_CODEC;
    default: return FYCS_OV_CODEC;
    }
}

/* One shared lower_bound retains full binary-key ordering and avoids repeated
 * linear lookup. Sorted insertion and arena repacking remain bounded by writes.
 */
static size_t fycs_overlay_index_v1(const fycs_overlay_entry_v1 *entries,
    size_t count, const uint8_t key[32]) {
    size_t low = 0u, high = count;
    while (low < high) {
        size_t middle = low + (high - low) / 2u;
        if (memcmp(entries[middle].key, key, 32u) < 0) { low = middle + 1u; }
        else { high = middle; }
    }
    return low;
}

int32_t fycs_overlay_begin_v1(fycs_overlay_v1 *transaction,
    const fycs_overlay_entry_v1 *snapshot, size_t snapshot_count,
    fycs_overlay_entry_v1 *writes, size_t max_writes, uint8_t *arena,
    uint8_t *scratch, size_t capacity, uint64_t max_bytes, size_t max_value) {
    size_t i;
    if (transaction == NULL || writes == NULL || arena == NULL || scratch == NULL ||
        arena == scratch || max_writes == 0u || max_value == 0u || max_bytes == 0u ||
        max_bytes > (uint64_t)capacity ||
        (snapshot == NULL && snapshot_count != 0u)) { return FYCS_OV_INVALID; }
    for (i = 0u; i < snapshot_count; ++i) {
        if (snapshot[i].deleted != 0u ||
            (snapshot[i].value == NULL && snapshot[i].length != 0u) ||
            (i != 0u && memcmp(snapshot[i-1u].key, snapshot[i].key, 32u) >= 0)) {
            return FYCS_OV_INVALID;
        }
    }
    memset(transaction, 0, sizeof(*transaction));
    transaction->snapshot = snapshot;
    transaction->snapshot_count = snapshot_count;
    transaction->writes = writes;
    transaction->max_writes = max_writes;
    transaction->max_value = max_value;
    transaction->max_bytes = max_bytes;
    transaction->active = arena;
    transaction->scratch = scratch;
    transaction->arena_capacity = capacity;
    transaction->state = 1u;
    return FYCS_OV_OK;
}

int32_t fycs_overlay_read_v1(const fycs_overlay_v1 *transaction,
    const uint8_t key[32], const uint8_t **value, size_t *length, uint8_t *found) {
    size_t at;
    const fycs_overlay_entry_v1 *entry = NULL;
    if (transaction == NULL || key == NULL || value == NULL || length == NULL || found == NULL) {
        return FYCS_OV_INVALID;
    }
    if (transaction->state != 1u) { return FYCS_OV_CLOSED; }
    at = fycs_overlay_index_v1(transaction->writes, transaction->count, key);
    if (at < transaction->count && memcmp(transaction->writes[at].key, key, 32u) == 0) {
        entry = &transaction->writes[at];
    } else {
        at = fycs_overlay_index_v1(transaction->snapshot, transaction->snapshot_count, key);
        if (at < transaction->snapshot_count && memcmp(transaction->snapshot[at].key, key, 32u) == 0) {
            entry = &transaction->snapshot[at];
        }
    }
    *found = entry != NULL && entry->deleted == 0u ? 1u : 0u;
    *length = *found != 0u ? entry->length : 0u;
    *value = *found != 0u ? entry->value : NULL;
    return FYCS_OV_OK;
}

int32_t fycs_overlay_write_v1(fycs_overlay_v1 *transaction, const uint8_t key[32],
    const uint8_t *value, size_t length, uint8_t deleted) {
    size_t at, count, i, used = 0u;
    uint64_t old_cost = 0u, new_cost, next_bytes;
    uint8_t replacing;
    uint8_t saved_key[32];
    uint8_t *previous;
    if (transaction == NULL || key == NULL || deleted > 1u ||
        (value == NULL && length != 0u) || (deleted != 0u && length != 0u)) {
        return FYCS_OV_INVALID;
    }
    if (transaction->state != 1u) { return FYCS_OV_CLOSED; }
    if (deleted == 0u && length > transaction->max_value) { return FYCS_OV_VALUE_LIMIT; }
    at = fycs_overlay_index_v1(transaction->writes, transaction->count, key);
    replacing = at < transaction->count && memcmp(transaction->writes[at].key, key, 32u) == 0 ? 1u : 0u;
    count = transaction->count;
    if (replacing != 0u) { old_cost = UINT64_C(33) + (uint64_t)transaction->writes[at].length; }
    else {
        if (count >= transaction->max_writes) { return FYCS_OV_WRITE_LIMIT; }
        ++count;
    }
    if ((uint64_t)length > UINT64_MAX - UINT64_C(33)) { return FYCS_OV_BYTE_LIMIT; }
    new_cost = UINT64_C(33) + (uint64_t)length;
    if (transaction->write_bytes < old_cost) { return FYCS_OV_INVALID; }
    next_bytes = transaction->write_bytes - old_cost;
    if (new_cost > UINT64_MAX - next_bytes || new_cost > transaction->max_bytes ||
        next_bytes > transaction->max_bytes - new_cost) { return FYCS_OV_BYTE_LIMIT; }
    next_bytes += new_cost;
    memcpy(saved_key, key, 32u);
    /* All rejecting gates precede workspace mutation. The exact write-byte bound
     * also bounds packed value bytes, without accumulating replaced values.
     */
    if (replacing == 0u) {
        memmove(&transaction->writes[at+1u], &transaction->writes[at],
            (transaction->count - at) * sizeof(*transaction->writes));
    }
    for (i = 0u; i < count; ++i) {
        fycs_overlay_entry_v1 *entry = &transaction->writes[i];
        const uint8_t *source = i == at ? value : entry->value;
        size_t size = i == at ? length : entry->length;
        if (size != 0u) { memcpy(transaction->scratch + used, source, size); }
        entry->value = transaction->scratch + used;
        entry->length = size;
        used += size;
    }
    memcpy(transaction->writes[at].key, saved_key, 32u);
    transaction->writes[at].deleted = deleted;
    transaction->count = count;
    transaction->write_bytes = next_bytes;
    previous = transaction->active;
    transaction->active = transaction->scratch;
    transaction->scratch = previous;
    return FYCS_OV_OK;
}

int32_t fycs_overlay_commit_v1(fycs_overlay_v1 *transaction) {
    if (transaction == NULL) { return FYCS_OV_INVALID; }
    if (transaction->state != 1u) { return FYCS_OV_CLOSED; }
    /* Proposed sorted overlay only. The persistence port owns durable commit. */
    transaction->state = 2u;
    return FYCS_OV_OK;
}

int32_t fycs_overlay_abort_v1(fycs_overlay_v1 *transaction) {
    if (transaction == NULL) { return FYCS_OV_INVALID; }
    if (transaction->state != 1u) { return FYCS_OV_CLOSED; }
    memset(transaction->active, 0, transaction->arena_capacity);
    memset(transaction->scratch, 0, transaction->arena_capacity);
    memset(transaction->writes, 0, transaction->count * sizeof(*transaction->writes));
    transaction->count = 0u;
    transaction->write_bytes = 0u;
    transaction->state = 3u;
    return FYCS_OV_OK;
}
