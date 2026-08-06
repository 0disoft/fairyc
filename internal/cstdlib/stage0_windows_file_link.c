struct fy_std_windows_file_result_v0 fy_std_windows_file_hard_link_v0(
    const uint8_t *source,
    const uint8_t *destination
) {
    WCHAR wide_source[FY_STD_WINDOWS_PATH_CAPACITY_V0];
    uint32_t source_status = fy_std_windows_path_v0(source, wide_source);
    if (source_status != FY_STD_WINDOWS_FILE_OK_V0) {
        return fy_std_windows_file_result_v0(source_status, ERROR_SUCCESS, 0);
    }
    WCHAR wide_destination[FY_STD_WINDOWS_PATH_CAPACITY_V0];
    uint32_t destination_status = fy_std_windows_path_v0(destination, wide_destination);
    if (destination_status != FY_STD_WINDOWS_FILE_OK_V0) {
        return fy_std_windows_file_result_v0(destination_status, ERROR_SUCCESS, 0);
    }
    if (!CreateHardLinkW(wide_destination, wide_source, NULL)) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            0
        );
    }
    return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS, 0);
}
