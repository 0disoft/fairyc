int fy_std_windows_file_directory_probe_v0(void) {
    static const uint8_t non_ascii_path[] = {UINT8_C(0x80), UINT8_C(0)};
    static const uint8_t directory_path[] = "fairyc-windows-directory-probe";
    static const uint8_t directory_child_path[] =
        "fairyc-windows-directory-probe\\child.tmp";
    static const uint8_t directory_missing_parent_path[] =
        "fairyc-windows-directory-missing-parent\\child";
    static const uint8_t directory_file_collision_path[] =
        "fairyc-windows-directory-file-collision.tmp";
    struct fy_std_windows_open_options_v0 write_options = {
        UINT8_C(0), UINT8_C(1), UINT8_C(1), UINT8_C(1), UINT8_C(0), UINT8_C(0)
    };
    struct fy_std_windows_open_options_v0 read_options = {
        UINT8_C(1), UINT8_C(0), UINT8_C(0), UINT8_C(0), UINT8_C(0), UINT8_C(0)
    };
    struct fy_std_windows_file_v0 file = {NULL, UINT8_C(0)};
    struct fy_std_windows_file_result_v0 result = fy_std_windows_directory_create_v0(NULL);
    if (result.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 132;
    result = fy_std_windows_directory_remove_v0(non_ascii_path);
    if (result.status != FY_STD_WINDOWS_FILE_PATH_ENCODING_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 133;
    result = fy_std_windows_directory_create_v0(directory_missing_parent_path);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_PATH_NOT_FOUND || result.count != 0) return 134;
    result = fy_std_windows_file_open_v0(directory_file_collision_path, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 135;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 136;
    result = fy_std_windows_directory_create_v0(directory_file_collision_path);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_ALREADY_EXISTS || result.count != 0) return 137;
    result = fy_std_windows_file_remove_v0(directory_file_collision_path);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 138;
    result = fy_std_windows_directory_create_v0(directory_path);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 139;
    result = fy_std_windows_directory_create_v0(directory_path);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_ALREADY_EXISTS || result.count != 0) return 140;
    result = fy_std_windows_file_open_v0(directory_child_path, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 141;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 142;
    result = fy_std_windows_directory_remove_v0(directory_path);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_DIR_NOT_EMPTY || result.count != 0) return 143;
    result = fy_std_windows_file_remove_v0(directory_child_path);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 144;
    result = fy_std_windows_directory_remove_v0(directory_path);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 145;
    result = fy_std_windows_directory_remove_v0(directory_path);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_FILE_NOT_FOUND || result.count != 0) return 146;

    static const uint8_t directory_rename_source[] =
        "fairyc-windows-directory-rename-source-probe";
    static const uint8_t directory_rename_destination[] =
        "fairyc-windows-directory-rename-destination-probe";
    static const uint8_t directory_rename_missing[] =
        "fairyc-windows-directory-rename-missing-probe";
    static const uint8_t directory_rename_child_source[] =
        "fairyc-windows-directory-rename-source-probe\\child.tmp";
    static const uint8_t directory_rename_child_destination[] =
        "fairyc-windows-directory-rename-destination-probe\\child.tmp";
    static const uint8_t directory_rename_file[] =
        "fairyc-windows-directory-rename-file-probe.tmp";
    result = fy_std_windows_directory_rename_v0(NULL, directory_rename_destination);
    if (result.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 147;
    result = fy_std_windows_directory_rename_v0(directory_rename_missing, non_ascii_path);
    if (result.status != FY_STD_WINDOWS_FILE_PATH_ENCODING_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 148;
    result = fy_std_windows_directory_rename_v0(
        directory_rename_missing,
        directory_rename_destination
    );
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_FILE_NOT_FOUND || result.count != 0) return 149;
    result = fy_std_windows_file_open_v0(directory_rename_file, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 150;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 151;
    result = fy_std_windows_directory_rename_v0(
        directory_rename_file,
        directory_rename_destination
    );
    if (result.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 152;
    result = fy_std_windows_file_remove_v0(directory_rename_file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 153;
    result = fy_std_windows_directory_create_v0(directory_rename_source);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 154;
    result = fy_std_windows_file_open_v0(directory_rename_child_source, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 155;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 156;
    result = fy_std_windows_directory_create_v0(directory_rename_destination);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 157;
    result = fy_std_windows_directory_rename_v0(
        directory_rename_source,
        directory_rename_destination
    );
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_ALREADY_EXISTS || result.count != 0) return 158;
    result = fy_std_windows_directory_remove_v0(directory_rename_destination);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 159;
    result = fy_std_windows_directory_rename_v0(
        directory_rename_source,
        directory_rename_destination
    );
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 160;
    result = fy_std_windows_directory_remove_v0(directory_rename_source);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_FILE_NOT_FOUND || result.count != 0) return 161;
    result = fy_std_windows_file_open_v0(directory_rename_child_destination, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 162;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 163;
    result = fy_std_windows_file_remove_v0(directory_rename_child_destination);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 164;
    result = fy_std_windows_directory_remove_v0(directory_rename_destination);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 165;
    result = fy_std_windows_directory_rename_v0(
        directory_rename_source,
        directory_rename_destination
    );
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_FILE_NOT_FOUND || result.count != 0) return 166;

    static const uint8_t directory_all_leaf[] = "fy-dir-all\\n\\leaf";
    static const uint8_t directory_all_dot[] = "fy-dir-dot\\..\\escape";
    static const uint8_t directory_all_partial[] = "fy-dir-partial\\created\\bad?name";
    result = fy_std_windows_directory_ensure_v0(NULL);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0, ERROR_SUCCESS)) return 167;
    result = fy_std_windows_directory_ensure_v0(directory_all_dot);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0, ERROR_SUCCESS)) return 169;
    result = fy_std_windows_directory_ensure_v0(directory_all_leaf);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS)) return 170;
    if (!RemoveDirectoryW(L"fy-dir-all\\n\\leaf")) return 172;
    if (!RemoveDirectoryW(L"fy-dir-all\\n")) return 173;
    if (!RemoveDirectoryW(L"fy-dir-all")) return 174;
    result = fy_std_windows_directory_ensure_v0(directory_all_partial);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_IO_ERROR_V0, ERROR_INVALID_NAME)) return 175;
    if (!RemoveDirectoryW(L"fy-dir-partial\\created")) return 176;
    if (!RemoveDirectoryW(L"fy-dir-partial")) return 177;
    return 0;
}
