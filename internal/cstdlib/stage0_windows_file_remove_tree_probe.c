int fy_std_windows_file_remove_tree_probe_v0(void) {
    static const uint8_t root[] = "fy-remove-tree";
    static const uint8_t nested[] = "fy-remove-tree\\nested";
    static const uint8_t root_file[] = "fy-remove-tree\\a.tmp";
    static const uint8_t nested_file[] = "fy-remove-tree\\nested\\b.tmp";
    static const uint8_t missing[] = "fy-remove-tree-missing";
    static const uint8_t collision[] = "fy-remove-tree-file.tmp";
    static const uint8_t blocked[] = "fy-remove-tree-blocked";
    static const uint8_t blocked_file[] = "fy-remove-tree-blocked\\child.tmp";
    static const uint8_t readonly[] = "fy-remove-tree-readonly";
    static const uint8_t readonly_file[] = "fy-remove-tree-readonly\\child.tmp";
    static const uint8_t trailing[] = "fy-remove-tree\\";
    static const uint8_t dot[] = "fy-remove-tree-dot\\..\\escape";
    static const uint8_t non_ascii[] = {UINT8_C(0x80), UINT8_C(0)};
    struct fy_std_windows_open_options_v0 write_options = {
        UINT8_C(0), UINT8_C(1), UINT8_C(1), UINT8_C(1), UINT8_C(0), UINT8_C(0)
    };
    struct fy_std_windows_file_v0 file = {NULL, UINT8_C(0)};
    struct fy_std_windows_file_result_v0 result =
        fy_std_windows_dir_remove_all_v0(NULL);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0, ERROR_SUCCESS)) return 275;
    result = fy_std_windows_dir_remove_all_v0(non_ascii);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_PATH_ENCODING_V0, ERROR_SUCCESS)) return 276;
    result = fy_std_windows_dir_remove_all_v0((const uint8_t *)"/");
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0, ERROR_SUCCESS)) return 277;
    result = fy_std_windows_dir_remove_all_v0(trailing);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0, ERROR_SUCCESS)) return 278;
    result = fy_std_windows_dir_remove_all_v0(dot);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0, ERROR_SUCCESS)) return 279;
    result = fy_std_windows_dir_remove_all_v0(missing);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_IO_ERROR_V0, ERROR_FILE_NOT_FOUND)) return 280;

    result = fy_std_windows_file_open_v0(collision, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 281;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 282;
    result = fy_std_windows_dir_remove_all_v0(collision);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_IO_ERROR_V0, ERROR_DIRECTORY)) return 283;
    if (fy_std_windows_file_remove_v0(collision).status != FY_STD_WINDOWS_FILE_OK_V0) return 284;

    if (fy_std_windows_directory_ensure_v0(nested).status != FY_STD_WINDOWS_FILE_OK_V0) return 285;
    result = fy_std_windows_file_open_v0(root_file, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 286;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 287;
    result = fy_std_windows_file_open_v0(nested_file, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 288;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 289;
    result = fy_std_windows_dir_remove_all_v0(root);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 4) return 290;
    if (GetFileAttributesW(L"fy-remove-tree") != INVALID_FILE_ATTRIBUTES) return 291;

    if (fy_std_windows_directory_ensure_v0(blocked).status != FY_STD_WINDOWS_FILE_OK_V0) return 292;
    result = fy_std_windows_file_open_v0(blocked_file, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 293;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 294;
    HANDLE root_handle = CreateFileW(
        L"fy-remove-tree-blocked",
        FILE_LIST_DIRECTORY,
        FILE_SHARE_READ | FILE_SHARE_WRITE,
        NULL,
        OPEN_EXISTING,
        FILE_FLAG_BACKUP_SEMANTICS,
        NULL
    );
    if (root_handle == INVALID_HANDLE_VALUE) return 295;
    result = fy_std_windows_dir_remove_all_v0(blocked);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_SHARING_VIOLATION || result.count != 1) return 296;
    if (GetFileAttributesW(L"fy-remove-tree-blocked\\child.tmp") != INVALID_FILE_ATTRIBUTES ||
        GetFileAttributesW(L"fy-remove-tree-blocked") == INVALID_FILE_ATTRIBUTES) return 297;
    if (!CloseHandle(root_handle)) return 298;
    result = fy_std_windows_dir_remove_all_v0(blocked);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != 1) return 299;

    if (fy_std_windows_directory_ensure_v0(readonly).status != FY_STD_WINDOWS_FILE_OK_V0) return 300;
    result = fy_std_windows_file_open_v0(readonly_file, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 301;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 302;
    if (!SetFileAttributesW(L"fy-remove-tree-readonly\\child.tmp", FILE_ATTRIBUTE_READONLY)) {
        return 303;
    }
    result = fy_std_windows_dir_remove_all_v0(readonly);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_ACCESS_DENIED || result.count != 0) return 304;
    if (!SetFileAttributesW(L"fy-remove-tree-readonly\\child.tmp", FILE_ATTRIBUTE_NORMAL)) {
        return 305;
    }
    result = fy_std_windows_dir_remove_all_v0(readonly);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != 2) return 306;
    return 0;
}
