int fy_std_windows_file_link_probe_v0(void) {
    static const uint8_t source[] = "fairyc-windows-file-link-source-probe.tmp";
    static const uint8_t destination[] = "fairyc-windows-file-link-destination-probe.tmp";
    static const uint8_t missing_source[] = "fairyc-windows-file-link-missing-probe.tmp";
    static const uint8_t missing_destination[] =
        "fairyc-windows-file-link-missing-destination-probe.tmp";
    static const uint8_t non_ascii_path[] = {UINT8_C(0x80), UINT8_C(0)};
    static const uint8_t payload[] = {
        UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121),
        UINT8_C(76), UINT8_C(105), UINT8_C(110), UINT8_C(107)
    };
    struct fy_std_windows_open_options_v0 write_options = {
        UINT8_C(0), UINT8_C(1), UINT8_C(1), UINT8_C(1), UINT8_C(0), UINT8_C(0)
    };
    struct fy_std_windows_open_options_v0 read_options = {
        UINT8_C(1), UINT8_C(0), UINT8_C(0), UINT8_C(0), UINT8_C(0), UINT8_C(0)
    };
    DeleteFileW(L"fairyc-windows-file-link-destination-probe.tmp");
    DeleteFileW(L"fairyc-windows-file-link-source-probe.tmp");
    DeleteFileW(L"fairyc-windows-file-link-missing-destination-probe.tmp");

    struct fy_std_windows_file_result_v0 result =
        fy_std_windows_file_hard_link_v0(NULL, destination);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0, ERROR_SUCCESS)) return 238;
    result = fy_std_windows_file_hard_link_v0(source, NULL);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0, ERROR_SUCCESS)) return 239;
    result = fy_std_windows_file_hard_link_v0(non_ascii_path, destination);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_PATH_ENCODING_V0, ERROR_SUCCESS)) return 240;
    result = fy_std_windows_file_hard_link_v0(missing_source, non_ascii_path);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_PATH_ENCODING_V0, ERROR_SUCCESS)) return 241;
    result = fy_std_windows_file_hard_link_v0(missing_source, missing_destination);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_IO_ERROR_V0, ERROR_FILE_NOT_FOUND)) return 242;
    if (GetFileAttributesW(L"fairyc-windows-file-link-missing-destination-probe.tmp") !=
        INVALID_FILE_ATTRIBUTES) return 243;

    struct fy_std_windows_file_v0 file = {NULL, UINT8_C(0)};
    result = fy_std_windows_file_open_v0(source, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 244;
    result = fy_std_windows_file_write_all_v0(&file, payload, sizeof payload);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != sizeof payload) return 245;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 246;

    result = fy_std_windows_file_hard_link_v0(source, destination);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS)) return 247;
    result = fy_std_windows_file_hard_link_v0(source, destination);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_IO_ERROR_V0, ERROR_ALREADY_EXISTS)) return 248;

    struct fy_std_windows_file_v0 source_file = {NULL, UINT8_C(0)};
    struct fy_std_windows_file_v0 destination_file = {NULL, UINT8_C(0)};
    result = fy_std_windows_file_open_v0(source, &read_options, &source_file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 249;
    result = fy_std_windows_file_open_v0(destination, &read_options, &destination_file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 250;
    struct fy_std_windows_file_id_v0 source_id = {0};
    struct fy_std_windows_file_id_v0 destination_id = {0};
    result = fy_std_windows_file_id_v0(&source_file, &source_id);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 251;
    result = fy_std_windows_file_id_v0(&destination_file, &destination_id);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        !fy_std_windows_file_ids_equal_v0(&source_id, &destination_id)) return 252;
    if (fy_std_windows_file_close_v0(&source_file).status != FY_STD_WINDOWS_FILE_OK_V0) return 253;
    if (fy_std_windows_file_close_v0(&destination_file).status != FY_STD_WINDOWS_FILE_OK_V0) {
        return 254;
    }

    result = fy_std_windows_file_remove_v0(source);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 255;
    result = fy_std_windows_file_open_v0(destination, &read_options, &destination_file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 256;
    uint8_t received[sizeof payload] = {0};
    result = fy_std_windows_file_read_exact_v0(&destination_file, received, sizeof received);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != sizeof received ||
        memcmp(received, payload, sizeof payload) != 0) return 257;
    if (fy_std_windows_file_close_v0(&destination_file).status != FY_STD_WINDOWS_FILE_OK_V0) {
        return 258;
    }
    result = fy_std_windows_file_remove_v0(destination);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 259;
    return 0;
}
