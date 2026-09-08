/* Compiler-private storage transport. The caller owns element lifetimes and
 * supplies a live, exclusively borrowed descriptor of this exact layout.
 * A descriptor may move, but must never be copied into a second live owner. */
struct fy_private_vector_storage_v0 {
    struct fy_std_system_heap_block_v0 block;
    size_t length;
    size_t element_size;
    size_t element_alignment;
};

static int fy_private_vector_storage_valid_v0(const struct fy_private_vector_storage_v0 *value) {
    return value->element_size != 0 &&
           fy_std_system_heap_alignment_supported_v0(value->element_alignment) &&
           value->element_size % value->element_alignment == 0 &&
           fy_std_system_heap_block_valid_v0(&value->block) &&
           value->block.size % value->element_size == 0 &&
           (value->block.data == NULL || value->block.alignment == value->element_alignment) &&
           value->length <= value->block.size / value->element_size;
}

/* memcpy avoids accessing a generated-Fairy descriptor through a different C
 * struct lvalue. No raw pointer validation or public FFI safety is implied. */
uint32_t fy_private_vector_initialize_v0(void *storage, size_t element_size, size_t element_alignment) {
    if (storage == NULL || element_size == 0) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    if (!fy_std_system_heap_alignment_supported_v0(element_alignment)) {
        return FY_STD_ALLOC_UNSUPPORTED_ALIGNMENT_V0;
    }
    if (element_size % element_alignment != 0) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    struct fy_private_vector_storage_v0 value;
    memcpy(&value, storage, sizeof value);
    if (value.block.data != NULL || value.block.size != 0 || value.block.alignment != 0 ||
        value.length != 0 || value.element_size != 0 || value.element_alignment != 0) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    value.element_size = element_size;
    value.element_alignment = element_alignment;
    memcpy(storage, &value, sizeof value);
    return FY_STD_ALLOC_OK_V0;
}

uint32_t fy_private_vector_reserve_v0(void *storage, size_t minimum_capacity) {
    if (storage == NULL) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    struct fy_private_vector_storage_v0 value;
    memcpy(&value, storage, sizeof value);
    if (!fy_private_vector_storage_valid_v0(&value)) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    size_t maximum_capacity = SIZE_MAX / value.element_size;
    if (minimum_capacity > maximum_capacity) {
        return FY_STD_ALLOC_SIZE_OVERFLOW_V0;
    }
    size_t capacity = value.block.size / value.element_size;
    if (minimum_capacity <= capacity) {
        return FY_STD_ALLOC_OK_V0;
    }
    size_t next_capacity = capacity;
    if (next_capacity == 0) {
        next_capacity = maximum_capacity < 8 ? maximum_capacity : 8;
    }
    while (next_capacity < minimum_capacity) {
        if (next_capacity > maximum_capacity / 2) {
            next_capacity = minimum_capacity;
            break;
        }
        next_capacity *= 2;
    }
    struct fy_std_system_heap_block_v0 replacement = {NULL, 0, 0};
    uint32_t status = fy_std_system_heap_allocate_v0(
        next_capacity * value.element_size, value.element_alignment, &replacement);
    if (status != FY_STD_ALLOC_OK_V0) {
        return status;
    }
    if (value.length != 0) {
        memcpy(replacement.data, value.block.data, value.length * value.element_size);
    }
    status = fy_std_system_heap_deallocate_v0(&value.block);
    if (status != FY_STD_ALLOC_OK_V0) {
        fy_std_system_heap_deallocate_v0(&replacement);
        return status;
    }
    value.block = replacement;
    memcpy(storage, &value, sizeof value);
    return FY_STD_ALLOC_OK_V0;
}

/* The caller must destroy or transfer every initialized element first. */
uint32_t fy_private_vector_release_empty_v0(void *storage) {
    if (storage == NULL) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    struct fy_private_vector_storage_v0 value;
    memcpy(&value, storage, sizeof value);
    if (!fy_private_vector_storage_valid_v0(&value) || value.length != 0) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    uint32_t status = fy_std_system_heap_deallocate_v0(&value.block);
    if (status != FY_STD_ALLOC_OK_V0) {
        return status;
    }
    value.element_size = 0;
    value.element_alignment = 0;
    memcpy(storage, &value, sizeof value);
    return FY_STD_ALLOC_OK_V0;
}
