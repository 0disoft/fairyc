#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#ifndef FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0
#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 _Alignof(max_align_t)
#endif

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

static int fy_std_system_heap_alignment_supported_v0(size_t alignment) {
    return alignment != 0 &&
           (alignment & (alignment - 1)) == 0 &&
           alignment <= FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0;
}

static int fy_std_system_heap_block_valid_v0(const struct fy_std_system_heap_block_v0 *block) {
    if (block == NULL) {
        return 0;
    }
    if (block->data == NULL) {
        return block->size == 0 && block->alignment == 0;
    }
    return block->size != 0 && fy_std_system_heap_alignment_supported_v0(block->alignment);
}

uint32_t fy_std_system_heap_allocate_v0(
    size_t size,
    size_t alignment,
    struct fy_std_system_heap_block_v0 *output
) {
    if (output == NULL) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    if (!fy_std_system_heap_alignment_supported_v0(alignment)) {
        return FY_STD_ALLOC_UNSUPPORTED_ALIGNMENT_V0;
    }
    if (size == 0) {
        output->data = NULL;
        output->size = 0;
        output->alignment = 0;
        return FY_STD_ALLOC_OK_V0;
    }
    void *data = malloc(size);
    if (data == NULL) {
        return FY_STD_ALLOC_OUT_OF_MEMORY_V0;
    }
    output->data = data;
    output->size = size;
    output->alignment = alignment;
    return FY_STD_ALLOC_OK_V0;
}

uint32_t fy_std_system_heap_resize_v0(
    struct fy_std_system_heap_block_v0 *block,
    size_t new_size,
    size_t alignment
) {
    if (!fy_std_system_heap_block_valid_v0(block)) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    if (!fy_std_system_heap_alignment_supported_v0(alignment)) {
        return FY_STD_ALLOC_UNSUPPORTED_ALIGNMENT_V0;
    }
    if (new_size == 0) {
        free(block->data);
        block->data = NULL;
        block->size = 0;
        block->alignment = 0;
        return FY_STD_ALLOC_OK_V0;
    }

    struct fy_std_system_heap_block_v0 replacement = {NULL, 0, 0};
    uint32_t status = fy_std_system_heap_allocate_v0(new_size, alignment, &replacement);
    if (status != FY_STD_ALLOC_OK_V0) {
        return status;
    }
    if (block->data != NULL) {
        size_t copy_size = block->size < new_size ? block->size : new_size;
        memcpy(replacement.data, block->data, copy_size);
    }
    free(block->data);
    *block = replacement;
    return FY_STD_ALLOC_OK_V0;
}

uint32_t fy_std_system_heap_deallocate_v0(struct fy_std_system_heap_block_v0 *block) {
    if (!fy_std_system_heap_block_valid_v0(block)) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    free(block->data);
    block->data = NULL;
    block->size = 0;
    block->alignment = 0;
    return FY_STD_ALLOC_OK_V0;
}
