#include <stddef.h>
#include <stdint.h>

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

uint32_t fy_std_system_heap_allocate_v0(
    size_t size,
    size_t alignment,
    struct fy_std_system_heap_block_v0 *output
);
uint32_t fy_std_system_heap_resize_v0(
    struct fy_std_system_heap_block_v0 *block,
    size_t new_size,
    size_t alignment
);
uint32_t fy_std_system_heap_deallocate_v0(struct fy_std_system_heap_block_v0 *block);

static int block_is_empty(const struct fy_std_system_heap_block_v0 *block) {
    return block->data == NULL && block->size == 0 && block->alignment == 0;
}

int main(void) {
    struct fy_std_system_heap_block_v0 block = {NULL, 0, 0};
    if (fy_std_system_heap_allocate_v0(0, 1, &block) != FY_STD_ALLOC_OK_V0 || !block_is_empty(&block)) {
        return 1;
    }
    if (fy_std_system_heap_allocate_v0(32, 16, &block) != FY_STD_ALLOC_OK_V0 ||
        block.data == NULL || block.size != 32 || block.alignment != 16 ||
        ((uintptr_t)block.data % 16) != 0) {
        return 2;
    }
    uint8_t *bytes = (uint8_t *)block.data;
    for (size_t index = 0; index < block.size; ++index) {
        bytes[index] = (uint8_t)(index ^ 0x5au);
    }

    void *original_data = block.data;
    size_t original_size = block.size;
    size_t original_alignment = block.alignment;
    if (fy_std_system_heap_resize_v0(&block, 64, 3) != FY_STD_ALLOC_UNSUPPORTED_ALIGNMENT_V0 ||
        block.data != original_data || block.size != original_size || block.alignment != original_alignment) {
        return 3;
    }
    if (fy_std_system_heap_resize_v0(&block, 64, 16) != FY_STD_ALLOC_OK_V0 ||
        block.data == NULL || block.size != 64 || block.alignment != 16 ||
        ((uintptr_t)block.data % 16) != 0) {
        return 4;
    }
    bytes = (uint8_t *)block.data;
    for (size_t index = 0; index < 32; ++index) {
        if (bytes[index] != (uint8_t)(index ^ 0x5au)) {
            return 5;
        }
    }
    if (fy_std_system_heap_resize_v0(&block, 8, 8) != FY_STD_ALLOC_OK_V0 ||
        block.data == NULL || block.size != 8 || block.alignment != 8) {
        return 6;
    }
    bytes = (uint8_t *)block.data;
    for (size_t index = 0; index < 8; ++index) {
        if (bytes[index] != (uint8_t)(index ^ 0x5au)) {
            return 7;
        }
    }
    if (fy_std_system_heap_resize_v0(&block, 0, 1) != FY_STD_ALLOC_OK_V0 || !block_is_empty(&block)) {
        return 8;
    }
    if (fy_std_system_heap_deallocate_v0(&block) != FY_STD_ALLOC_OK_V0 || !block_is_empty(&block)) {
        return 9;
    }
    if (fy_std_system_heap_allocate_v0(1, 3, &block) != FY_STD_ALLOC_UNSUPPORTED_ALIGNMENT_V0 ||
        !block_is_empty(&block)) {
        return 10;
    }
    if (fy_std_system_heap_allocate_v0(1, 1, NULL) != FY_STD_ALLOC_INVALID_ARGUMENT_V0) {
        return 11;
    }
    struct fy_std_system_heap_block_v0 corrupt = {NULL, 1, 0};
    if (fy_std_system_heap_deallocate_v0(&corrupt) != FY_STD_ALLOC_INVALID_ARGUMENT_V0 ||
        corrupt.data != NULL || corrupt.size != 1 || corrupt.alignment != 0) {
        return 12;
    }
    return 0;
}
