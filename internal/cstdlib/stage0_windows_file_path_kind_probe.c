struct fy_std_windows_file_kind_result_v0 fy_std_windows_file_path_kind_v0(
    const uint8_t *
);

enum fy_std_windows_path_kind_probe_v0 {
    FY_STD_WINDOWS_PATH_KIND_FILE_V0 = 1,
    FY_STD_WINDOWS_PATH_KIND_DIRECTORY_V0 = 2
};

int fy_std_windows_file_path_kind_probe_v0(void) {
    static const uint8_t non_ascii_path[] = {UINT8_C(0x80), UINT8_C(0)};
    static const uint8_t path_kind_file[] = "fairyc-windows-path-kind-file-probe.tmp";
    static const uint8_t path_kind_directory[] =
        "fairyc-windows-path-kind-directory-probe";
    static const uint8_t path_kind_missing[] = "fairyc-windows-path-kind-missing-probe";

    struct fy_std_windows_file_kind_result_v0 kind =
        fy_std_windows_file_path_kind_v0(NULL);
    if (kind.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        kind.os_code != (uint32_t)ERROR_SUCCESS || kind.kind != 0) return 196;
    kind = fy_std_windows_file_path_kind_v0(non_ascii_path);
    if (kind.status != FY_STD_WINDOWS_FILE_PATH_ENCODING_V0 ||
        kind.os_code != (uint32_t)ERROR_SUCCESS || kind.kind != 0) return 197;

    HANDLE file = CreateFileW(
        L"fairyc-windows-path-kind-file-probe.tmp",
        GENERIC_WRITE,
        0,
        NULL,
        CREATE_ALWAYS,
        FILE_ATTRIBUTE_NORMAL,
        NULL
    );
    if (file == INVALID_HANDLE_VALUE) return 198;
    if (!CloseHandle(file)) return 199;
    if (!CreateDirectoryW(L"fairyc-windows-path-kind-directory-probe", NULL)) return 200;

    kind = fy_std_windows_file_path_kind_v0(path_kind_file);
    if (kind.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        kind.os_code != (uint32_t)ERROR_SUCCESS ||
        kind.kind != FY_STD_WINDOWS_PATH_KIND_FILE_V0) return 201;
    kind = fy_std_windows_file_path_kind_v0(path_kind_directory);
    if (kind.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        kind.os_code != (uint32_t)ERROR_SUCCESS ||
        kind.kind != FY_STD_WINDOWS_PATH_KIND_DIRECTORY_V0) return 202;
    kind = fy_std_windows_file_path_kind_v0(path_kind_missing);
    if (kind.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        kind.os_code != (uint32_t)ERROR_FILE_NOT_FOUND || kind.kind != 0) return 203;

    if (!RemoveDirectoryW(L"fairyc-windows-path-kind-directory-probe")) return 204;
    if (!DeleteFileW(L"fairyc-windows-path-kind-file-probe.tmp")) return 205;
    return 0;
}
