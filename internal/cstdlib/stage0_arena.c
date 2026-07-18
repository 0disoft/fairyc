struct fy_std_arena_node_v0 {
    struct fy_std_system_heap_block_v0 storage;
    struct fy_std_arena_node_v0 *next;
};

struct fy_std_arena_v0 {
    struct fy_std_arena_node_v0 *head;
    uint64_t generation;
    uint8_t active_region;
};

struct fy_std_region_v0 {
    struct fy_std_arena_v0 *arena;
    struct fy_std_arena_node_v0 *mark;
    uint64_t generation;
    uint8_t active;
};

static int fy_std_arena_valid_v0(const struct fy_std_arena_v0 *arena) {
    return arena != NULL && arena->generation != 0 && arena->active_region <= 1;
}

static void fy_std_arena_advance_generation_v0(struct fy_std_arena_v0 *arena) {
    arena->generation = arena->generation == UINT64_MAX ? 1 : arena->generation + 1;
}

static int fy_std_region_valid_v0(const struct fy_std_region_v0 *region) {
    return region != NULL && region->active == 1 && region->arena != NULL &&
           fy_std_arena_valid_v0(region->arena) && region->arena->active_region == 1 &&
           region->generation == region->arena->generation;
}

uint32_t fy_std_arena_initialize_v0(struct fy_std_arena_v0 *output) {
    if (output == NULL) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    output->head = NULL;
    output->generation = 1;
    output->active_region = 0;
    return FY_STD_ALLOC_OK_V0;
}

uint32_t fy_std_region_begin_v0(struct fy_std_arena_v0 *arena, struct fy_std_region_v0 *output) {
    if (!fy_std_arena_valid_v0(arena) || output == NULL || arena->active_region != 0) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    output->arena = arena;
    output->mark = arena->head;
    output->generation = arena->generation;
    output->active = 1;
    arena->active_region = 1;
    return FY_STD_ALLOC_OK_V0;
}

static uint32_t fy_std_region_allocate_node_v0(
    struct fy_std_region_v0 *region,
    size_t size,
    size_t alignment,
    void **output
) {
    if (!fy_std_region_valid_v0(region) || output == NULL) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    if (size == 0) {
        if (!fy_std_system_heap_alignment_supported_v0(alignment)) {
            return FY_STD_ALLOC_UNSUPPORTED_ALIGNMENT_V0;
        }
        *output = NULL;
        return FY_STD_ALLOC_OK_V0;
    }

    struct fy_std_system_heap_block_v0 storage = {NULL, 0, 0};
    uint32_t status = fy_std_system_heap_allocate_v0(size, alignment, &storage);
    if (status != FY_STD_ALLOC_OK_V0) {
        return status;
    }
    struct fy_std_system_heap_block_v0 node_block = {NULL, 0, 0};
    status = fy_std_system_heap_allocate_v0(
        sizeof(struct fy_std_arena_node_v0),
        _Alignof(struct fy_std_arena_node_v0),
        &node_block
    );
    if (status != FY_STD_ALLOC_OK_V0) {
        fy_std_system_heap_deallocate_v0(&storage);
        return status;
    }
    struct fy_std_arena_node_v0 *node = (struct fy_std_arena_node_v0 *)node_block.data;
    node->storage = storage;
    node->next = region->arena->head;
    region->arena->head = node;
    *output = storage.data;
    return FY_STD_ALLOC_OK_V0;
}

uint32_t fy_std_region_allocate_zeroed_v0(
    struct fy_std_region_v0 *region,
    size_t size,
    size_t alignment,
    void **output
) {
    uint32_t status = fy_std_region_allocate_node_v0(region, size, alignment, output);
    if (status == FY_STD_ALLOC_OK_V0 && size != 0) {
        memset(*output, 0, size);
    }
    return status;
}

uint32_t fy_std_region_copy_bytes_v0(
    struct fy_std_region_v0 *region,
    const void *source,
    size_t size,
    size_t alignment,
    void **output
) {
    if (size != 0 && source == NULL) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    uint32_t status = fy_std_region_allocate_node_v0(region, size, alignment, output);
    if (status == FY_STD_ALLOC_OK_V0 && size != 0) {
        memmove(*output, source, size);
    }
    return status;
}

static uint32_t fy_std_arena_release_until_v0(
    struct fy_std_arena_v0 *arena,
    struct fy_std_arena_node_v0 *mark
) {
    while (arena->head != mark) {
        struct fy_std_arena_node_v0 *node = arena->head;
        if (node == NULL) {
            return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
        }
        arena->head = node->next;
        struct fy_std_system_heap_block_v0 node_block = {
            node,
            sizeof(struct fy_std_arena_node_v0),
            _Alignof(struct fy_std_arena_node_v0)
        };
        uint32_t status = fy_std_system_heap_deallocate_v0(&node->storage);
        if (status != FY_STD_ALLOC_OK_V0) {
            return status;
        }
        status = fy_std_system_heap_deallocate_v0(&node_block);
        if (status != FY_STD_ALLOC_OK_V0) {
            return status;
        }
    }
    return FY_STD_ALLOC_OK_V0;
}

uint32_t fy_std_region_reset_v0(struct fy_std_region_v0 *region) {
    if (!fy_std_region_valid_v0(region)) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    struct fy_std_arena_v0 *arena = region->arena;
    uint32_t status = fy_std_arena_release_until_v0(arena, region->mark);
    if (status != FY_STD_ALLOC_OK_V0) {
        return status;
    }
    arena->active_region = 0;
    fy_std_arena_advance_generation_v0(arena);
    region->arena = NULL;
    region->mark = NULL;
    region->generation = 0;
    region->active = 0;
    return FY_STD_ALLOC_OK_V0;
}

uint32_t fy_std_arena_drop_v0(struct fy_std_arena_v0 *arena) {
    if (!fy_std_arena_valid_v0(arena) || arena->active_region != 0) {
        return FY_STD_ALLOC_INVALID_ARGUMENT_V0;
    }
    uint32_t status = fy_std_arena_release_until_v0(arena, NULL);
    if (status != FY_STD_ALLOC_OK_V0) {
        return status;
    }
    fy_std_arena_advance_generation_v0(arena);
    return FY_STD_ALLOC_OK_V0;
}
