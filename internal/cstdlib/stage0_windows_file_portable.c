struct fy_std_windows_file_result_v0 fy_std_file_open_v0(
    const uint8_t *path,
    const struct fy_std_windows_open_options_v0 *options,
    void *output_storage
) {
    return fy_std_windows_file_open_v0(path, options, output_storage);
}

struct fy_std_windows_file_result_v0 fy_std_file_open_utf8_v0(
    const uint8_t *path,
    size_t length,
    const struct fy_std_windows_open_options_v0 *options,
    void *output_storage
) {
    if (path == NULL || length == 0 || length >= (size_t)FY_STD_WINDOWS_PATH_CAPACITY_V0) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0, ERROR_SUCCESS, 0
        );
    }
    for (size_t index = 0; index < length; ++index) {
        if (path[index] == UINT8_C(0)) {
            return fy_std_windows_file_result_v0(
                FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0, ERROR_SUCCESS, 0
            );
        }
    }
    uint8_t terminated[FY_STD_WINDOWS_PATH_CAPACITY_V0];
    memcpy(terminated, path, length);
    terminated[length] = UINT8_C(0);
    return fy_std_windows_file_open_v0(terminated, options, output_storage);
}

struct fy_std_windows_file_result_v0 fy_std_file_read_v0(
    void *file_storage,
    uint8_t *buffer,
    size_t length
) {
    return fy_std_windows_file_read_v0(file_storage, buffer, length);
}

struct fy_std_windows_file_result_v0 fy_std_file_read_exact_v0(void *file, uint8_t *buffer, size_t length) {
    return fy_std_windows_file_read_exact_v0(file, buffer, length);
}
struct fy_std_windows_file_result_v0 fy_std_file_remove_v0(const uint8_t *path) {
    return fy_std_windows_file_remove_v0(path);
}
struct fy_std_windows_file_result_v0 fy_std_file_replace_v0(const uint8_t *replacement, const uint8_t *replaced) {
    return fy_std_windows_file_replace_v0(replacement, replaced);
}

struct fy_std_windows_file_position_result_v0 fy_std_file_position_v0(void *file_storage) {
    return fy_std_windows_file_position_v0(file_storage);
}

struct fy_std_windows_file_size_result_v0 fy_std_file_size_v0(const void *file_storage) {
    return fy_std_windows_file_size_v0(file_storage);
}

struct fy_std_windows_file_position_result_v0 fy_std_file_seek_start_v0(
    void *file_storage,
    uint64_t offset
) {
    return fy_std_windows_file_seek_start_v0(file_storage, offset);
}

struct fy_std_windows_file_position_result_v0 fy_std_file_seek_current_v0(
    void *file_storage,
    int32_t offset
) {
    return fy_std_windows_file_seek_current_v0(file_storage, offset);
}

struct fy_std_windows_file_position_result_v0 fy_std_file_seek_end_v0(
    void *file_storage,
    int32_t offset
) {
    return fy_std_windows_file_seek_end_v0(file_storage, offset);
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
