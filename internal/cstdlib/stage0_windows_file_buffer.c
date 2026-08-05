struct fy_std_windows_file_buffer_result_v0 {
    uint32_t file_status;
    uint32_t os_code;
    uint32_t allocation_status;
    size_t count;
};

static struct fy_std_windows_file_buffer_result_v0
fy_std_windows_file_buffer_result_v0(
    uint32_t file_status,
    DWORD os_code,
    uint32_t allocation_status,
    size_t count
) {
    struct fy_std_windows_file_buffer_result_v0 result = {
        file_status,
        (uint32_t)os_code,
        allocation_status,
        count
    };
    return result;
}

struct fy_std_windows_file_buffer_result_v0 fy_std_windows_file_read_append_v0(
    void *file_storage,
    void *buffer_storage,
    size_t maximum
) {
    struct fy_std_windows_file_result_v0 file_result = fy_std_windows_file_read_v0(
        file_storage,
        NULL,
        0
    );
    if (file_result.status != FY_STD_WINDOWS_FILE_OK_V0) {
        return fy_std_windows_file_buffer_result_v0(
            file_result.status,
            (DWORD)file_result.os_code,
            FY_STD_ALLOC_OK_V0,
            0
        );
    }

    struct fy_std_buffer_v0 *buffer = (struct fy_std_buffer_v0 *)buffer_storage;
    if (!fy_std_buffer_valid_v0(buffer)) {
        return fy_std_windows_file_buffer_result_v0(
            FY_STD_WINDOWS_FILE_OK_V0,
            ERROR_SUCCESS,
            FY_STD_ALLOC_INVALID_ARGUMENT_V0,
            0
        );
    }
    if (maximum > SIZE_MAX - buffer->length) {
        return fy_std_windows_file_buffer_result_v0(
            FY_STD_WINDOWS_FILE_OK_V0,
            ERROR_SUCCESS,
            FY_STD_ALLOC_SIZE_OVERFLOW_V0,
            0
        );
    }
    if (maximum == 0) {
        return fy_std_windows_file_buffer_result_v0(
            FY_STD_WINDOWS_FILE_OK_V0,
            ERROR_SUCCESS,
            FY_STD_ALLOC_OK_V0,
            0
        );
    }

    size_t old_length = buffer->length;
    uint32_t allocation_status = fy_std_buffer_reserve_v0(buffer, old_length + maximum);
    if (allocation_status != FY_STD_ALLOC_OK_V0) {
        return fy_std_windows_file_buffer_result_v0(
            FY_STD_WINDOWS_FILE_OK_V0,
            ERROR_SUCCESS,
            allocation_status,
            0
        );
    }

    file_result = fy_std_windows_file_read_v0(
        file_storage,
        (uint8_t *)buffer->block.data + old_length,
        maximum
    );
    if (file_result.status != FY_STD_WINDOWS_FILE_OK_V0) {
        return fy_std_windows_file_buffer_result_v0(
            file_result.status,
            (DWORD)file_result.os_code,
            FY_STD_ALLOC_OK_V0,
            0
        );
    }
    if (file_result.count > maximum) {
        return fy_std_windows_file_buffer_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            FY_STD_ALLOC_OK_V0,
            0
        );
    }
    buffer->length = old_length + file_result.count;
    return fy_std_windows_file_buffer_result_v0(
        FY_STD_WINDOWS_FILE_OK_V0,
        ERROR_SUCCESS,
        FY_STD_ALLOC_OK_V0,
        file_result.count
    );
}
