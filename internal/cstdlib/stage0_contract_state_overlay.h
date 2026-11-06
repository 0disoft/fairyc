//go:build ignore

#ifndef FYCS_OVERLAY_V1_H
#define FYCS_OVERLAY_V1_H
#include <stddef.h>
#include <stdint.h>

enum {
    FYCS_OV_OK = 0,
    FYCS_OV_INVALID = 1,
    FYCS_OV_CLOSED = 2,
    FYCS_OV_VALUE_LIMIT = 3,
    FYCS_OV_WRITE_LIMIT = 4,
    FYCS_OV_BYTE_LIMIT = 5,
    FYCS_OV_CODEC = 6
};
enum {
    FYCS_CODEC_BOOL = 1, FYCS_CODEC_U32 = 2, FYCS_CODEC_U64 = 3,
    FYCS_CODEC_BYTES = 4, FYCS_CODEC_UTF8 = 5,
    FYCS_CODEC_BYTES20 = 6, FYCS_CODEC_BYTES32 = 7
};

typedef struct fycs_overlay_entry_v1 {
    uint8_t key[32];
    const uint8_t *value;
    size_t length;
    uint8_t deleted;
} fycs_overlay_entry_v1;

/* All memory belongs to the caller. Snapshot entries/bytes are immutable;
 * workspace buffers, entries and transaction must be distinct live allocations.
 * The compiler supplies canonical physical keys and values. Codec validation is
 * explicit; this physical overlay does not infer a field from its hashed key.
 * Read views expire at the next successful mutation or close. A write may use
 * the current view: copying into the other arena preserves its source bytes.
 */
typedef struct fycs_overlay_v1 {
    const fycs_overlay_entry_v1 *snapshot;
    size_t snapshot_count;
    fycs_overlay_entry_v1 *writes;
    size_t count;
    size_t max_writes;
    size_t max_value;
    uint64_t max_bytes;
    uint64_t write_bytes;
    uint8_t *active;
    uint8_t *scratch;
    size_t arena_capacity;
    uint8_t state;
} fycs_overlay_v1;

int32_t fycs_codec_valid_v1(uint8_t, const uint8_t *, size_t);
int32_t fycs_overlay_begin_v1(fycs_overlay_v1 *, const fycs_overlay_entry_v1 *, size_t,
    fycs_overlay_entry_v1 *, size_t, uint8_t *, uint8_t *, size_t, uint64_t, size_t);
int32_t fycs_overlay_read_v1(const fycs_overlay_v1 *, const uint8_t[32],
    const uint8_t **, size_t *, uint8_t *);
int32_t fycs_overlay_write_v1(fycs_overlay_v1 *, const uint8_t[32],
    const uint8_t *, size_t, uint8_t);
int32_t fycs_overlay_commit_v1(fycs_overlay_v1 *);
int32_t fycs_overlay_abort_v1(fycs_overlay_v1 *);

#endif
