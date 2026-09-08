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

/* Byte transport requires representation-relocatable values.
 * Element/output spans must be live, at least element_size bytes, and disjoint
 * from both the descriptor and its backing allocation. No element borrow may
 * survive a mutating call. These are unsafe caller obligations, not pointer checks. */
static uint32_t fy_private_vector_push_bytes_v0(void *storage, const void *element, size_t element_size) {
    if (storage == NULL || element == NULL) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    struct fy_private_vector_storage_v0 value;
    memcpy(&value, storage, sizeof value);
    if (!fy_private_vector_storage_valid_v0(&value) || element_size != value.element_size) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    if (value.length == SIZE_MAX / value.element_size) {
        return FY_STD_ALLOC_SIZE_OVERFLOW_V0;
    }
    uint32_t status = fy_private_vector_reserve_v0(storage, value.length + 1);
    if (status != FY_STD_ALLOC_OK_V0) {
        return status;
    }
    memcpy(&value, storage, sizeof value);
    memcpy((unsigned char *)value.block.data + value.length * value.element_size, element, element_size);
    value.length++;
    memcpy(storage, &value, sizeof value);
    return FY_STD_ALLOC_OK_V0;
}

/* Copy wrappers may only duplicate copyable values, never resource ownership. */
uint32_t fy_private_vector_push_copy_v0(void *storage, const void *element, size_t element_size) {
    return fy_private_vector_push_bytes_v0(storage, element, element_size);
}

/* Only elements without destruction obligations may be discarded this way.
 * The caller proves that property and exclusive access; byte size is not a type
 * capability check. Capacity is retained and old payload bytes are not erased. */
uint32_t fy_private_vector_clear_copy_v0(void *storage, size_t element_size) {
    if (storage == NULL) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    struct fy_private_vector_storage_v0 value;
    memcpy(&value, storage, sizeof value);
    if (!fy_private_vector_storage_valid_v0(&value) || element_size != value.element_size) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    value.length = 0;
    memcpy(storage, &value, sizeof value);
    return FY_STD_ALLOC_OK_V0;
}

/* The declaring owner supplies a valid non-owning empty representation, in a
 * separate live span disjoint from every other argument/allocation. Success
 * transfers ownership into the prefix and resets the donor before returning.
 * Failure neither consumes nor changes the donor. */
uint32_t fy_private_vector_push_take_v0(void *storage, void *element, const void *empty, size_t element_size) {
    if (empty == NULL) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    uint32_t status = fy_private_vector_push_bytes_v0(storage, element, element_size);
    if (status == FY_STD_ALLOC_OK_V0) {
        memcpy(element, empty, element_size);
    }
    return status;
}

uint32_t fy_private_vector_read_copy_v0(const void *storage, size_t index, void *output, size_t element_size) {
    if (storage == NULL || output == NULL) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    struct fy_private_vector_storage_v0 value;
    memcpy(&value, storage, sizeof value);
    if (!fy_private_vector_storage_valid_v0(&value) || element_size != value.element_size || index >= value.length) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    memcpy(output, (const unsigned char *)value.block.data + index * value.element_size, element_size);
    return FY_STD_ALLOC_OK_V0;
}

static uint32_t fy_private_vector_pop_bytes_v0(void *storage, void *output, size_t element_size) {
    if (storage == NULL || output == NULL) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    struct fy_private_vector_storage_v0 value;
    memcpy(&value, storage, sizeof value);
    if (!fy_private_vector_storage_valid_v0(&value) || element_size != value.element_size || value.length == 0) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    memcpy(output, (const unsigned char *)value.block.data + (value.length - 1) * value.element_size, element_size);
    value.length--;
    memcpy(storage, &value, sizeof value);
    return FY_STD_ALLOC_OK_V0;
}

uint32_t fy_private_vector_pop_copy_v0(void *storage, void *output, size_t element_size) {
    return fy_private_vector_pop_bytes_v0(storage, output, element_size);
}

/* Output must be a valid non-owning empty value of the exact element type.
 * Success transfers the tail obligation to output; no read-copy resource API
 * is supplied. Removed tail bytes are outside the initialized prefix. */
uint32_t fy_private_vector_pop_take_v0(void *storage, void *output, size_t element_size) {
    return fy_private_vector_pop_bytes_v0(storage, output, element_size);
}

/* Retire the last initialized slot and hand its sole element obligation to the
 * unsafe caller. The caller must immediately take the typed value before any
 * storage mutation/release; the returned address is not an element borrow.
 * NULL means invalid input or empty prefix, with no descriptor change. */
void *fy_private_vector_pop_slot_v0(void *storage, size_t element_size) {
    if (storage == NULL) return NULL;
    struct fy_private_vector_storage_v0 value;
    memcpy(&value, storage, sizeof value);
    if (!fy_private_vector_storage_valid_v0(&value) || element_size != value.element_size || value.length == 0) {
        return NULL;
    }
    value.length--;
    void *slot = (unsigned char *)value.block.data + value.length * value.element_size;
    memcpy(storage, &value, sizeof value);
    return slot;
}

/* Prepare one uninitialized tail slot without changing length. slot_output
 * points to a live pointer-sized output disjoint from the descriptor/backing
 * allocation. On failure both descriptor and output are unchanged. */
uint32_t fy_private_vector_prepare_push_v0(void *storage, size_t element_size, void *slot_output) {
    if (storage == NULL || slot_output == NULL) return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    struct fy_private_vector_storage_v0 value;
    memcpy(&value, storage, sizeof value);
    if (!fy_private_vector_storage_valid_v0(&value) || element_size != value.element_size) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    if (value.length == SIZE_MAX / value.element_size) return FY_STD_ALLOC_SIZE_OVERFLOW_V0;
    uint32_t status = fy_private_vector_reserve_v0(storage, value.length + 1);
    if (status != FY_STD_ALLOC_OK_V0) return status;
    memcpy(&value, storage, sizeof value);
    void *slot = (unsigned char *)value.block.data + value.length * value.element_size;
    memcpy(slot_output, &slot, sizeof slot);
    return FY_STD_ALLOC_OK_V0;
}

/* The caller must have written exactly one valid owned element into the prepared
 * slot, with no intervening descriptor mutation, alias access or callback. A
 * matching address is not proof of initialization/ownership. This publishes the
 * initialized-prefix obligation; it neither writes nor destroys the payload. */
uint32_t fy_private_vector_commit_push_v0(void *storage, void *slot, size_t element_size) {
    if (storage == NULL || slot == NULL) return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    struct fy_private_vector_storage_v0 value;
    memcpy(&value, storage, sizeof value);
    if (!fy_private_vector_storage_valid_v0(&value) || element_size != value.element_size ||
        value.length == value.block.size / value.element_size) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    void *expected = (unsigned char *)value.block.data + value.length * value.element_size;
    if (slot != expected) return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    value.length++;
    memcpy(storage, &value, sizeof value);
    return FY_STD_ALLOC_OK_V0;
}
