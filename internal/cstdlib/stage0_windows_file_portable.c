struct fy_std_windows_file_result_v0 fy_std_file_open_v0(
    const uint8_t *path,
    const struct fy_std_windows_open_options_v0 *options,
    void *output_storage
) {
    return fy_std_windows_file_open_v0(path, options, output_storage);
}

struct fy_std_windows_file_result_v0 fy_std_file_read_v0(
    void *file_storage,
    uint8_t *buffer,
    size_t length
) {
    return fy_std_windows_file_read_v0(file_storage, buffer, length);
}

struct fy_std_windows_file_position_result_v0 fy_std_file_position_v0(void *file_storage) {
    return fy_std_windows_file_position_v0(file_storage);
}

struct fy_std_windows_file_position_result_v0 fy_std_file_seek_start_v0(
    void *file_storage,
    uint64_t offset
) {
    return fy_std_windows_file_seek_start_v0(file_storage, offset);
}

struct fy_std_windows_file_result_v0 fy_std_file_write_v0(
    void *file_storage,
    const uint8_t *buffer,
    size_t length
) {
    return fy_std_windows_file_write_v0(file_storage, buffer, length);
}

struct fy_std_windows_file_result_v0 fy_std_file_write_all_v0(
    void *file_storage,
    const uint8_t *buffer,
    size_t length
) {
    return fy_std_windows_file_write_all_v0(file_storage, buffer, length);
}

struct fy_std_windows_file_result_v0 fy_std_file_sync_v0(void *file_storage) {
    return fy_std_windows_file_sync_v0(file_storage);
}

struct fy_std_windows_file_result_v0 fy_std_file_close_v0(void *file_storage) {
    return fy_std_windows_file_close_v0(file_storage);
}

uint32_t fy_std_file_drop_v0(void *file_storage) {
    return fy_std_windows_file_drop_v0(file_storage);
}
