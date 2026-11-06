//go:build ignore

#ifndef FYCS_GRAPH_V1_H
#define FYCS_GRAPH_V1_H
#include "stage0_contract_state_overlay.h"

enum {
    FYCS_GRAPH_KEY_LIMIT = 7, FYCS_GRAPH_VALUE_SPACE = 8,
    FYCS_GRAPH_BORROW_INVALID = 9, FYCS_GRAPH_VIEW_RANGE = 10,
    FYCS_GRAPH_STEP_OVERFLOW = 11
};
typedef struct fycs_graph_value_v1 {
    const uint8_t *bytes;
    size_t length;
    uint8_t codec;
} fycs_graph_value_v1;
typedef struct fycs_graph_epoch_v1 {
    uint8_t key[32];
    uint64_t generation;
} fycs_graph_epoch_v1;
typedef struct fycs_graph_v1 {
    fycs_overlay_v1 *transaction;
    uint8_t *values;
    size_t capacity, used;
    fycs_graph_epoch_v1 *epochs;
    size_t epoch_capacity, epoch_count;
    uint64_t generation, operation, cost;
} fycs_graph_v1;
typedef struct fycs_graph_borrow_v1 {
    fycs_graph_v1 *owner;
    fycs_graph_value_v1 value;
    uint8_t key[32];
    uint64_t generation;
} fycs_graph_borrow_v1;

/* One context is initialized once for one open transaction. All supplied memory
 * is caller-owned and disjoint from snapshot, overlay and context storage.
 * SSA bytes are immutable copies in the value arena, never overlay read views.
 * Slot copies may share those immutable bytes. Borrow handles are block-local;
 * only a checked copy may outlive exact-key mutation. No allocator is hidden.
 */
int32_t fycs_graph_begin_v1(fycs_graph_v1 *, fycs_overlay_v1 *, uint8_t *, size_t,
    fycs_graph_epoch_v1 *, size_t);
int32_t fycs_graph_value_copy_v1(fycs_graph_v1 *, uint8_t, const uint8_t *, size_t, fycs_graph_value_v1 *);
int32_t fycs_graph_read_v1(fycs_graph_v1 *, const uint8_t[32], uint8_t, fycs_graph_value_v1 *, uint8_t *);
int32_t fycs_graph_write_v1(fycs_graph_v1 *, const uint8_t[32], const fycs_graph_value_v1 *, uint8_t);
int32_t fycs_graph_map_key_v1(const uint8_t[32], const fycs_graph_value_v1 *, size_t, uint8_t[32]);
int32_t fycs_graph_borrow_v1_begin(fycs_graph_v1 *, const uint8_t[32], uint8_t, fycs_graph_borrow_v1 *, uint8_t *);
int32_t fycs_graph_borrow_view_v1(fycs_graph_v1 *, const fycs_graph_borrow_v1 *, size_t, size_t, fycs_graph_borrow_v1 *);
int32_t fycs_graph_borrow_copy_v1(fycs_graph_v1 *, const fycs_graph_borrow_v1 *, fycs_graph_value_v1 *);
int32_t fycs_graph_step_v1(fycs_graph_v1 *, uint8_t, uint8_t);
#endif
