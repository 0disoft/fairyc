static struct fy_std_file_buffer_result_v0 fy_std_file_buffer_read_v0(
    void *file_storage, uint8_t *data, size_t length
) {
    struct fy_std_windows_file_result_v0 result = fy_std_file_read_v0(file_storage, data, length);
    return fy_std_file_buffer_result_v0(result.status, result.os_code, 0, result.count);
}

/* Retain the old private Windows symbol and record for existing callers. */
struct fy_std_windows_file_buffer_result_v0 {
    uint32_t file_status;
    uint32_t os_code;
    uint32_t allocation_status;
    size_t count;
};

struct fy_std_windows_file_buffer_result_v0 fy_std_windows_file_read_append_v0(
    void *file_storage, void *buffer_storage, size_t maximum
) {
    struct fy_std_file_buffer_result_v0 result = fy_std_file_read_append_v0(file_storage, buffer_storage, maximum);
    struct fy_std_windows_file_buffer_result_v0 legacy = {
        result.file_status, result.os_code, result.allocation_status, result.count
    };
    return legacy;
}
