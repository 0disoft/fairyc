struct fy_std_file_buffer_result_v0 {
    uint32_t file_status;
    uint32_t os_code;
    uint32_t allocation_status;
    size_t count;
};

/* The target adapter validates a zero-length request without an OS read. */
static struct fy_std_file_buffer_result_v0 fy_std_file_buffer_read_v0(
    void *file_storage, uint8_t *data, size_t length
);

static struct fy_std_file_buffer_result_v0 fy_std_file_buffer_result_v0(
    uint32_t file_status, uint32_t os_code, uint32_t allocation_status, size_t count
) {
    struct fy_std_file_buffer_result_v0 result = {
        file_status, os_code, allocation_status, count
    };
    return result;
}

struct fy_std_file_buffer_result_v0 fy_std_file_read_append_v0(
    void *file_storage, void *buffer_storage, size_t maximum
) {
    struct fy_std_file_buffer_result_v0 result = fy_std_file_buffer_read_v0(file_storage, NULL, 0);
    if (result.file_status != 0) {
        return result;
    }
    struct fy_std_buffer_v0 *buffer = (struct fy_std_buffer_v0 *)buffer_storage;
    if (!fy_std_buffer_valid_v0(buffer)) {
        return fy_std_file_buffer_result_v0(0, 0, FY_STD_ALLOC_INVALID_ARGUMENT_V0, 0);
    }
    if (maximum > SIZE_MAX - buffer->length) {
        return fy_std_file_buffer_result_v0(0, 0, FY_STD_ALLOC_SIZE_OVERFLOW_V0, 0);
    }
    if (maximum == 0) {
        return fy_std_file_buffer_result_v0(0, 0, FY_STD_ALLOC_OK_V0, 0);
    }
    size_t old_length = buffer->length;
    uint32_t allocation_status = fy_std_buffer_reserve_v0(buffer, old_length + maximum);
    if (allocation_status != FY_STD_ALLOC_OK_V0) {
        return fy_std_file_buffer_result_v0(0, 0, allocation_status, 0);
    }
    result = fy_std_file_buffer_read_v0(file_storage, (uint8_t *)buffer->block.data + old_length, maximum);
    if (result.file_status != 0) {
        result.count = 0;
        return result;
    }
    if (result.count > maximum) {
        return fy_std_file_buffer_result_v0(UINT32_C(2), 0, FY_STD_ALLOC_OK_V0, 0);
    }
    buffer->length = old_length + result.count;
    return result;
}
