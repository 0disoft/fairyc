static struct fy_std_file_buffer_result_v0 fy_std_file_buffer_read_v0(
    void *file_storage, uint8_t *data, size_t length
) {
    if (length == 0) {
        struct fy_std_linux_file_v0 file;
        if (!fy_std_file_storage_to_linux_v0(file_storage, &file)) {
            return fy_std_file_buffer_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0, 0);
        }
        return fy_std_file_buffer_result_v0(
            file.open != 0 ? FY_STD_FILE_OK_V0 : FY_STD_FILE_CLOSED_V0, 0, 0, 0
        );
    }
    struct fy_std_file_result_v0 result = fy_std_file_read_v0(file_storage, data, length);
    return fy_std_file_buffer_result_v0(result.status, (uint32_t)result.error_number, 0, result.count);
}
