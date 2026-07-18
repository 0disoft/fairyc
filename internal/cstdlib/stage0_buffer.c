struct fy_std_buffer_v0 {
    struct fy_std_system_heap_block_v0 block;
    size_t length;
};

static int fy_std_buffer_valid_v0(const struct fy_std_buffer_v0 *buffer) {
    return buffer != NULL &&
           fy_std_system_heap_block_valid_v0(&buffer->block) &&
           buffer->length <= buffer->block.size;
}

uint32_t fy_std_buffer_initialize_v0(size_t capacity, struct fy_std_buffer_v0 *output) {
    if (output == NULL) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    struct fy_std_system_heap_block_v0 block = {NULL, 0, 0};
    uint32_t status = fy_std_system_heap_allocate_v0(capacity, 1, &block);
    if (status != FY_STD_ALLOC_OK_V0) {
        return status;
    }
    output->block = block;
    output->length = 0;
    return FY_STD_ALLOC_OK_V0;
}

uint32_t fy_std_buffer_reserve_v0(struct fy_std_buffer_v0 *buffer, size_t minimum_capacity) {
    if (!fy_std_buffer_valid_v0(buffer)) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    if (minimum_capacity <= buffer->block.size) {
        return FY_STD_ALLOC_OK_V0;
    }
    size_t next_capacity = buffer->block.size < 8 ? 8 : buffer->block.size;
    while (next_capacity < minimum_capacity) {
        if (next_capacity > SIZE_MAX / 2) {
            next_capacity = minimum_capacity;
            break;
        }
        next_capacity *= 2;
    }
    struct fy_std_system_heap_block_v0 replacement = {NULL, 0, 0};
    uint32_t status = fy_std_system_heap_allocate_v0(next_capacity, 1, &replacement);
    if (status != FY_STD_ALLOC_OK_V0) {
        return status;
    }
    if (buffer->length != 0) {
        memcpy(replacement.data, buffer->block.data, buffer->length);
    }
    status = fy_std_system_heap_deallocate_v0(&buffer->block);
    if (status != FY_STD_ALLOC_OK_V0) {
        fy_std_system_heap_deallocate_v0(&replacement);
        return status;
    }
    buffer->block = replacement;
    return FY_STD_ALLOC_OK_V0;
}

static int fy_std_buffer_internal_source_v0(
    const struct fy_std_buffer_v0 *buffer,
    const uint8_t *source,
    size_t count,
    size_t *offset
) {
    if (buffer->block.data == NULL || source == NULL) {
        return 0;
    }
    uintptr_t data_address = (uintptr_t)buffer->block.data;
    uintptr_t source_address = (uintptr_t)source;
    if (source_address < data_address) {
        return 0;
    }
    uintptr_t distance = source_address - data_address;
    if (distance >= buffer->block.size) {
        return 0;
    }
    if (distance > buffer->length || count > buffer->length - (size_t)distance) {
        return -1;
    }
    *offset = (size_t)distance;
    return 1;
}

uint32_t fy_std_buffer_append_v0(
    struct fy_std_buffer_v0 *buffer,
    const uint8_t *source,
    size_t count
) {
    if (!fy_std_buffer_valid_v0(buffer) || (count != 0 && source == NULL)) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    if (count > SIZE_MAX - buffer->length) {
        return FY_STD_ALLOC_SIZE_OVERFLOW_V0;
    }
    if (count == 0) {
        return FY_STD_ALLOC_OK_V0;
    }
    size_t new_length = buffer->length + count;
    size_t internal_offset = 0;
    int internal_source = fy_std_buffer_internal_source_v0(buffer, source, count, &internal_offset);
    if (internal_source < 0) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    if (new_length > buffer->block.size) {
        size_t old_length = buffer->length;
        uint32_t status = fy_std_buffer_reserve_v0(buffer, new_length);
        if (status != FY_STD_ALLOC_OK_V0) {
            return status;
        }
        if (internal_source) {
            source = (const uint8_t *)buffer->block.data + internal_offset;
        }
        memmove((uint8_t *)buffer->block.data + old_length, source, count);
    } else {
        memmove((uint8_t *)buffer->block.data + buffer->length, source, count);
    }
    buffer->length = new_length;
    return FY_STD_ALLOC_OK_V0;
}

uint32_t fy_std_buffer_resize_zeroed_v0(struct fy_std_buffer_v0 *buffer, size_t new_length) {
    if (!fy_std_buffer_valid_v0(buffer)) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    if (new_length > buffer->length) {
        size_t old_length = buffer->length;
        uint32_t status = fy_std_buffer_reserve_v0(buffer, new_length);
        if (status != FY_STD_ALLOC_OK_V0) {
            return status;
        }
        memset((uint8_t *)buffer->block.data + old_length, 0, new_length - old_length);
    }
    buffer->length = new_length;
    return FY_STD_ALLOC_OK_V0;
}

uint32_t fy_std_buffer_truncate_v0(struct fy_std_buffer_v0 *buffer, size_t new_length) {
    if (!fy_std_buffer_valid_v0(buffer) || new_length > buffer->length) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    buffer->length = new_length;
    return FY_STD_ALLOC_OK_V0;
}

uint32_t fy_std_buffer_clear_v0(struct fy_std_buffer_v0 *buffer) {
    return fy_std_buffer_truncate_v0(buffer, 0);
}

uint32_t fy_std_buffer_drop_v0(struct fy_std_buffer_v0 *buffer) {
    if (!fy_std_buffer_valid_v0(buffer)) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    uint32_t status = fy_std_system_heap_deallocate_v0(&buffer->block);
    if (status != FY_STD_ALLOC_OK_V0) {
        return status;
    }
    buffer->length = 0;
    return FY_STD_ALLOC_OK_V0;
}
