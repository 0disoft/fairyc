#include <stddef.h>
#include <stdint.h>
#include <string.h>

enum fy_std_alloc_status_v0 {
    FY_STD_ALLOC_OK_V0 = 0,
    FY_STD_ALLOC_OUT_OF_MEMORY_V0 = 1,
    FY_STD_ALLOC_SIZE_OVERFLOW_V0 = 2,
    FY_STD_ALLOC_UNSUPPORTED_ALIGNMENT_V0 = 3,
    FY_STD_ALLOC_INVALID_ARGUMENT_V0 = 4
};

struct fy_std_system_heap_block_v0 {
    void *data;
    size_t size;
    size_t alignment;
};

struct fy_std_buffer_v0 {
    struct fy_std_system_heap_block_v0 block;
    size_t length;
};

uint32_t fy_std_buffer_initialize_v0(size_t capacity, struct fy_std_buffer_v0 *output);
uint32_t fy_std_buffer_reserve_v0(struct fy_std_buffer_v0 *buffer, size_t minimum_capacity);
uint32_t fy_std_buffer_append_v0(struct fy_std_buffer_v0 *buffer, const uint8_t *source, size_t count);
uint32_t fy_std_buffer_resize_zeroed_v0(struct fy_std_buffer_v0 *buffer, size_t new_length);
uint32_t fy_std_buffer_truncate_v0(struct fy_std_buffer_v0 *buffer, size_t new_length);
uint32_t fy_std_buffer_clear_v0(struct fy_std_buffer_v0 *buffer);
uint32_t fy_std_buffer_drop_v0(struct fy_std_buffer_v0 *buffer);

int main(void) {
    struct fy_std_buffer_v0 buffer = {{NULL, 0, 0}, 0};
    if (fy_std_buffer_initialize_v0(0, &buffer) != FY_STD_ALLOC_OK_V0 ||
        buffer.block.data != NULL || buffer.block.size != 0 || buffer.length != 0) {
        return 1;
    }
    static const uint8_t prefix[] = {'a', 'b', 'c', 'd', 'e', 'f'};
    if (fy_std_buffer_append_v0(&buffer, prefix, sizeof prefix) != FY_STD_ALLOC_OK_V0 ||
        buffer.length != sizeof prefix || buffer.block.size < buffer.length ||
        memcmp(buffer.block.data, prefix, sizeof prefix) != 0) {
        return 2;
    }
    void *reserved_data = buffer.block.data;
    if (fy_std_buffer_reserve_v0(&buffer, buffer.block.size) != FY_STD_ALLOC_OK_V0 ||
        buffer.block.data != reserved_data) {
        return 3;
    }
    const uint8_t *overlap = (const uint8_t *)buffer.block.data + 1;
    if (fy_std_buffer_append_v0(&buffer, overlap, 4) != FY_STD_ALLOC_OK_V0 ||
        buffer.length != 10 || memcmp(buffer.block.data, "abcdefbcde", 10) != 0) {
        return 4;
    }

    void *before_overflow_data = buffer.block.data;
    size_t before_overflow_size = buffer.block.size;
    size_t before_overflow_length = buffer.length;
    if (fy_std_buffer_append_v0(&buffer, prefix, SIZE_MAX) != FY_STD_ALLOC_SIZE_OVERFLOW_V0 ||
        buffer.block.data != before_overflow_data || buffer.block.size != before_overflow_size ||
        buffer.length != before_overflow_length || memcmp(buffer.block.data, "abcdefbcde", 10) != 0) {
        return 5;
    }
    if (fy_std_buffer_resize_zeroed_v0(&buffer, 16) != FY_STD_ALLOC_OK_V0 || buffer.length != 16 ||
        memcmp(buffer.block.data, "abcdefbcde", 10) != 0) {
        return 6;
    }
    const uint8_t *bytes = (const uint8_t *)buffer.block.data;
    for (size_t index = 10; index < 16; ++index) {
        if (bytes[index] != 0) {
            return 7;
        }
    }
    if (fy_std_buffer_truncate_v0(&buffer, 3) != FY_STD_ALLOC_OK_V0 ||
        buffer.length != 3 || memcmp(buffer.block.data, "abc", 3) != 0) {
        return 8;
    }
    if (fy_std_buffer_clear_v0(&buffer) != FY_STD_ALLOC_OK_V0 || buffer.length != 0) {
        return 9;
    }
    struct fy_std_buffer_v0 corrupt = buffer;
    corrupt.length = corrupt.block.size + 1;
    if (fy_std_buffer_clear_v0(&corrupt) != FY_STD_ALLOC_INVALID_ARGUMENT_V0 ||
        corrupt.length != corrupt.block.size + 1) {
        return 10;
    }
    if (fy_std_buffer_drop_v0(&buffer) != FY_STD_ALLOC_OK_V0 ||
        buffer.block.data != NULL || buffer.block.size != 0 || buffer.length != 0) {
        return 11;
    }
    return 0;
}
