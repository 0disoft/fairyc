struct fy_std_windows_file_position_result_v0 fy_std_windows_file_seek_current_v0(
    void *,
    int32_t
);
struct fy_std_windows_file_position_result_v0 fy_std_windows_file_seek_end_v0(
    void *,
    int32_t
);

int fy_std_windows_file_seek_probe_v0(void) {
    static const uint8_t path[] = "fairyc-windows-file-seek-probe.tmp";
    static const uint8_t payload[] = {
        UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121),
        UINT8_C(70), UINT8_C(105), UINT8_C(108), UINT8_C(101)
    };
    struct fy_std_windows_open_options_v0 write_options = {
        UINT8_C(0), UINT8_C(1), UINT8_C(1), UINT8_C(1), UINT8_C(0), UINT8_C(0)
    };
    struct fy_std_windows_file_v0 file = {NULL, UINT8_C(0)};
    struct fy_std_windows_file_result_v0 result =
        fy_std_windows_file_open_v0(path, &write_options, &file);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS)) return 206;
    result = fy_std_windows_file_write_all_v0(&file, payload, sizeof payload);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != sizeof payload) return 207;

    struct fy_std_windows_file_position_result_v0 position =
        fy_std_windows_file_seek_current_v0(&file, INT32_C(-4));
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(5)) return 208;
    position = fy_std_windows_file_seek_end_v0(&file, INT32_C(-9));
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(0)) return 209;
    position = fy_std_windows_file_seek_current_v0(&file, INT32_C(-1));
    if (position.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        position.os_code != (uint32_t)ERROR_NEGATIVE_SEEK ||
        position.position != UINT64_C(0)) return 210;
    position = fy_std_windows_file_position_v0(&file);
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(0)) return 211;
    position = fy_std_windows_file_seek_end_v0(&file, INT32_C(3));
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(12)) return 212;
    struct fy_std_windows_file_size_result_v0 size = fy_std_windows_file_size_v0(&file);
    if (size.status != FY_STD_WINDOWS_FILE_OK_V0 || size.size != sizeof payload) return 213;
    position = fy_std_windows_file_seek_end_v0(&file, INT32_C(-9));
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(0)) return 214;

    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 215;
    if (fy_std_windows_file_seek_current_v0(&file, INT32_C(0)).status !=
        FY_STD_WINDOWS_FILE_CLOSED_V0) return 216;
    if (fy_std_windows_file_seek_end_v0(&file, INT32_C(0)).status !=
        FY_STD_WINDOWS_FILE_CLOSED_V0) return 217;
    result = fy_std_windows_file_remove_v0(path);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS)) return 218;
    return 0;
}
