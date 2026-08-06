#include <winioctl.h>

struct fy_std_windows_mount_point_buffer_v0 {
    ULONG reparse_tag;
    USHORT reparse_data_length;
    USHORT reserved;
    USHORT substitute_name_offset;
    USHORT substitute_name_length;
    USHORT print_name_offset;
    USHORT print_name_length;
    WCHAR path_buffer[512];
};

static int fy_std_windows_create_junction_probe_v0(
    const WCHAR *junction,
    const WCHAR *target
) {
    WCHAR full_target[256];
    DWORD full_length = GetFullPathNameW(
        target,
        (DWORD)(sizeof full_target / sizeof full_target[0]),
        full_target,
        NULL
    );
    if (full_length == 0 || full_length >= sizeof full_target / sizeof full_target[0] ||
        (size_t)full_length * 2 + 6 > 512) {
        return 0;
    }
    if (!CreateDirectoryW(junction, NULL)) {
        return 0;
    }
    HANDLE handle = CreateFileW(
        junction,
        GENERIC_WRITE,
        0,
        NULL,
        OPEN_EXISTING,
        FILE_FLAG_OPEN_REPARSE_POINT | FILE_FLAG_BACKUP_SEMANTICS,
        NULL
    );
    if (handle == INVALID_HANDLE_VALUE) {
        return 0;
    }
    struct fy_std_windows_mount_point_buffer_v0 buffer;
    memset(&buffer, 0, sizeof buffer);
    buffer.reparse_tag = IO_REPARSE_TAG_MOUNT_POINT;
    buffer.path_buffer[0] = L'\\';
    buffer.path_buffer[1] = L'?';
    buffer.path_buffer[2] = L'?';
    buffer.path_buffer[3] = L'\\';
    memcpy(&buffer.path_buffer[4], full_target, (full_length + 1) * sizeof(WCHAR));
    size_t substitute_characters = (size_t)full_length + 4;
    size_t print_start = substitute_characters + 1;
    memcpy(
        &buffer.path_buffer[print_start],
        full_target,
        (full_length + 1) * sizeof(WCHAR)
    );
    buffer.substitute_name_offset = 0;
    buffer.substitute_name_length = (USHORT)(substitute_characters * sizeof(WCHAR));
    buffer.print_name_offset = (USHORT)(print_start * sizeof(WCHAR));
    buffer.print_name_length = (USHORT)(full_length * sizeof(WCHAR));
    size_t path_bytes = (print_start + full_length + 1) * sizeof(WCHAR);
    buffer.reparse_data_length = (USHORT)(8 + path_bytes);
    DWORD returned = 0;
    BOOL set = DeviceIoControl(
        handle,
        FSCTL_SET_REPARSE_POINT,
        &buffer,
        (DWORD)(8 + buffer.reparse_data_length),
        NULL,
        0,
        &returned,
        NULL
    );
    BOOL closed = CloseHandle(handle);
    return set && closed;
}

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
    static const uint8_t reparse_root[] = "fy-remove-tree-reparse";
    static const uint8_t reparse_target[] = "fy-remove-tree-target";
    static const uint8_t reparse_target_file[] = "fy-remove-tree-target\\marker.tmp";
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

    if (fy_std_windows_directory_ensure_v0(reparse_root).status !=
        FY_STD_WINDOWS_FILE_OK_V0) return 307;
    if (fy_std_windows_directory_ensure_v0(reparse_target).status !=
        FY_STD_WINDOWS_FILE_OK_V0) return 308;
    result = fy_std_windows_file_open_v0(reparse_target_file, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 309;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 310;
    if (!fy_std_windows_create_junction_probe_v0(
            L"fy-remove-tree-reparse\\junction",
            L"fy-remove-tree-target"
        )) return 311;
    DWORD junction_attributes =
        GetFileAttributesW(L"fy-remove-tree-reparse\\junction");
    if (junction_attributes == INVALID_FILE_ATTRIBUTES ||
        (junction_attributes & FILE_ATTRIBUTE_DIRECTORY) == 0 ||
        (junction_attributes & FILE_ATTRIBUTE_REPARSE_POINT) == 0) return 312;
    result = fy_std_windows_dir_remove_all_v0(reparse_root);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != 2) return 313;
    if (GetFileAttributesW(L"fy-remove-tree-reparse") != INVALID_FILE_ATTRIBUTES ||
        GetFileAttributesW(L"fy-remove-tree-target") == INVALID_FILE_ATTRIBUTES ||
        GetFileAttributesW(L"fy-remove-tree-target\\marker.tmp") ==
            INVALID_FILE_ATTRIBUTES) return 314;
    result = fy_std_windows_dir_remove_all_v0(reparse_target);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != 2) return 315;
    return 0;
}
