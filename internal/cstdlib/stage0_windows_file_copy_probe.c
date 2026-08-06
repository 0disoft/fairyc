struct fy_std_windows_file_result_v0 fy_std_windows_file_copy_bytes_v0(
    const uint8_t *,
    const uint8_t *
);

int fy_std_windows_file_copy_probe_v0(void) {
    static const uint8_t non_ascii_path[] = {UINT8_C(0x80), UINT8_C(0)};
    static const uint8_t copy_source[] = "fairyc-windows-file-copy-source-probe.tmp";
    static const uint8_t copy_destination[] = "fairyc-windows-file-copy-destination-probe.tmp";
    static const uint8_t copy_locked_destination[] =
        "fairyc-windows-file-copy-locked-destination-probe.tmp";
    static const uint8_t copy_missing_source[] =
        "fairyc-windows-file-copy-missing-source-probe.tmp";
    static const uint8_t copy_missing_destination[] =
        "fairyc-windows-file-copy-missing-destination-probe.tmp";
    struct fy_std_windows_open_options_v0 write_options = {
        UINT8_C(0), UINT8_C(1), UINT8_C(1), UINT8_C(1), UINT8_C(0), UINT8_C(0)
    };
    struct fy_std_windows_open_options_v0 read_options = {
        UINT8_C(1), UINT8_C(0), UINT8_C(0), UINT8_C(0), UINT8_C(0), UINT8_C(0)
    };
    struct fy_std_windows_file_v0 file = {NULL, UINT8_C(0)};
    struct fy_std_windows_file_result_v0 result = fy_std_windows_file_copy_bytes_v0(
        NULL,
        copy_destination
    );
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0, ERROR_SUCCESS)) return 178;
    result = fy_std_windows_file_copy_bytes_v0(copy_missing_source, non_ascii_path);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_PATH_ENCODING_V0, ERROR_SUCCESS)) return 179;
    uint8_t copy_payload[UINT32_C(65537)];
    for (size_t index = 0; index < sizeof copy_payload; ++index) {
        copy_payload[index] = (uint8_t)(index % UINT8_C(251));
    }
    result = fy_std_windows_file_open_v0(copy_source, &write_options, &file);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS)) return 180;
    result = fy_std_windows_file_write_all_v0(&file, copy_payload, sizeof copy_payload);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != sizeof copy_payload) return 181;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 182;
    result = fy_std_windows_file_copy_bytes_v0(copy_source, copy_destination);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS ||
        result.count != sizeof copy_payload) return 183;
    result = fy_std_windows_file_copy_bytes_v0(copy_source, copy_destination);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_IO_ERROR_V0, ERROR_FILE_EXISTS)) return 184;
    result = fy_std_windows_file_open_v0(copy_destination, &read_options, &file);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS)) return 185;
    uint8_t copy_received[UINT32_C(65537)] = {0};
    result = fy_std_windows_file_read_exact_v0(&file, copy_received, sizeof copy_received);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.count != sizeof copy_received ||
        memcmp(copy_received, copy_payload, sizeof copy_payload) != 0) return 186;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 187;
    result = fy_std_windows_file_copy_bytes_v0(copy_missing_source, copy_missing_destination);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_IO_ERROR_V0, ERROR_FILE_NOT_FOUND)) return 188;
    result = fy_std_windows_file_remove_v0(copy_missing_destination);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_IO_ERROR_V0, ERROR_FILE_NOT_FOUND)) return 189;
    result = fy_std_windows_file_open_v0(copy_source, &read_options, &file);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS)) return 190;
    result = fy_std_windows_file_copy_bytes_v0(copy_source, copy_locked_destination);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_IO_ERROR_V0, ERROR_SHARING_VIOLATION)) return 191;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 192;
    result = fy_std_windows_file_remove_v0(copy_locked_destination);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_IO_ERROR_V0, ERROR_FILE_NOT_FOUND)) return 193;
    result = fy_std_windows_file_remove_v0(copy_destination);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS)) return 194;
    result = fy_std_windows_file_remove_v0(copy_source);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS)) return 195;
    return 0;
}
