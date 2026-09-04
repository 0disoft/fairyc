#include <stddef.h>
#include <stdint.h>
#include <string.h>

#define FYCS_NATIVE_VALUE_CAPACITY_V0 64u

typedef struct fycs_native_store_v0 {
    uint8_t value[FYCS_NATIVE_VALUE_CAPACITY_V0];
    size_t length;
    uint8_t present;
    uint8_t layout_hash[32];
    uint8_t body_hash[32];
} fycs_native_store_v0;

typedef struct fycs_native_transaction_v0 {
    fycs_native_store_v0 *owner;
    uint8_t value[FYCS_NATIVE_VALUE_CAPACITY_V0];
    size_t length;
    uint8_t present;
    uint8_t open;
} fycs_native_transaction_v0;

void fycs_native_store_initialize_v0(
    fycs_native_store_v0 *store,
    const uint8_t layout_hash[32],
    const uint8_t body_hash[32]
) {
    if (store == NULL || layout_hash == NULL || body_hash == NULL) {
        return;
    }
    memset(store, 0, sizeof(*store));
    memcpy(store->layout_hash, layout_hash, 32u);
    memcpy(store->body_hash, body_hash, 32u);
}

int32_t fycs_native_begin_v0(
    fycs_native_store_v0 *store,
    const uint8_t expected_layout_hash[32],
    const uint8_t expected_body_hash[32],
    fycs_native_transaction_v0 *transaction
) {
    if (store == NULL || expected_layout_hash == NULL || expected_body_hash == NULL || transaction == NULL) {
        return -1;
    }
    if (store->length > FYCS_NATIVE_VALUE_CAPACITY_V0 || store->present > 1u ||
        (store->present == 0u && store->length != 0u)) {
        return -3;
    }
    if (memcmp(store->layout_hash, expected_layout_hash, 32u) != 0 ||
        memcmp(store->body_hash, expected_body_hash, 32u) != 0) {
        return -2;
    }
    memset(transaction, 0, sizeof(*transaction));
    transaction->owner = store;
    transaction->length = store->length;
    transaction->present = store->present;
    if (store->present != 0u && store->length != 0u) {
        memcpy(transaction->value, store->value, store->length);
    }
    transaction->open = 1u;
    return 0;
}

int32_t fycs_native_cell_read_v0(
    const fycs_native_transaction_v0 *transaction,
    uint8_t output[FYCS_NATIVE_VALUE_CAPACITY_V0],
    size_t *output_length,
    uint8_t *found
) {
    if (transaction == NULL || output == NULL || output_length == NULL || found == NULL || transaction->open == 0u) {
        return -1;
    }
    *found = transaction->present;
    *output_length = transaction->present != 0u ? transaction->length : 0u;
    if (transaction->present != 0u && transaction->length != 0u) {
        memcpy(output, transaction->value, transaction->length);
    }
    return 0;
}

int32_t fycs_native_cell_write_v0(
    fycs_native_transaction_v0 *transaction,
    const uint8_t *value,
    size_t length
) {
    if (transaction == NULL || transaction->open == 0u || length > FYCS_NATIVE_VALUE_CAPACITY_V0 ||
        (value == NULL && length != 0u)) {
        return -1;
    }
    memset(transaction->value, 0, sizeof(transaction->value));
    if (length != 0u) {
        memcpy(transaction->value, value, length);
    }
    transaction->length = length;
    transaction->present = 1u;
    return 0;
}

int32_t fycs_native_cell_delete_v0(fycs_native_transaction_v0 *transaction) {
    if (transaction == NULL || transaction->open == 0u) {
        return -1;
    }
    memset(transaction->value, 0, sizeof(transaction->value));
    transaction->length = 0u;
    transaction->present = 0u;
    return 0;
}

int32_t fycs_native_commit_v0(fycs_native_transaction_v0 *transaction) {
    if (transaction == NULL || transaction->open == 0u || transaction->owner == NULL) {
        return -1;
    }
    memset(transaction->owner->value, 0, sizeof(transaction->owner->value));
    if (transaction->present != 0u && transaction->length != 0u) {
        memcpy(transaction->owner->value, transaction->value, transaction->length);
    }
    transaction->owner->length = transaction->length;
    transaction->owner->present = transaction->present;
    memset(transaction, 0, sizeof(*transaction));
    return 0;
}

void fycs_native_rollback_v0(fycs_native_transaction_v0 *transaction) {
    if (transaction != NULL) {
        memset(transaction, 0, sizeof(*transaction));
    }
}

size_t fycs_native_live_resources_v0(void) {
    return 0u;
}
