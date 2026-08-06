static int fy_std_windows_directory_entry_name_v0(const WCHAR *name) {
    return name != NULL &&
           !((name[0] == L'.' && name[1] == L'\0') ||
             (name[0] == L'.' && name[1] == L'.' && name[2] == L'\0'));
}

static struct fy_std_windows_file_result_v0 fy_std_windows_directory_removed_v0(
    size_t *completed
) {
    if (*completed == SIZE_MAX) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            ERROR_ARITHMETIC_OVERFLOW,
            *completed
        );
    }
    ++*completed;
    return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS, *completed);
}

static struct fy_std_windows_file_result_v0 fy_std_windows_directory_remove_tree_wide_v0(
    WCHAR *path,
    size_t length,
    size_t *completed
) {
    DWORD attributes = GetFileAttributesW(path);
    if (attributes == INVALID_FILE_ATTRIBUTES) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            *completed
        );
    }
    if ((attributes & FILE_ATTRIBUTE_DIRECTORY) == 0) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            ERROR_DIRECTORY,
            *completed
        );
    }
    if ((attributes & FILE_ATTRIBUTE_REPARSE_POINT) != 0) {
        if (!RemoveDirectoryW(path)) {
            return fy_std_windows_file_result_v0(
                FY_STD_WINDOWS_FILE_IO_ERROR_V0,
                GetLastError(),
                *completed
            );
        }
        return fy_std_windows_directory_removed_v0(completed);
    }
    if (length + 2 >= (size_t)FY_STD_WINDOWS_PATH_CAPACITY_V0) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_PATH_TOO_LONG_V0,
            ERROR_SUCCESS,
            *completed
        );
    }
    path[length] = L'\\';
    path[length + 1] = L'*';
    path[length + 2] = L'\0';
    WIN32_FIND_DATAW entry;
    HANDLE search = FindFirstFileW(path, &entry);
    DWORD search_error = search == INVALID_HANDLE_VALUE ? GetLastError() : ERROR_SUCCESS;
    path[length] = L'\0';
    if (search == INVALID_HANDLE_VALUE && search_error != ERROR_FILE_NOT_FOUND) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            search_error,
            *completed
        );
    }
    if (search != INVALID_HANDLE_VALUE) {
        for (;;) {
            if (fy_std_windows_directory_entry_name_v0(entry.cFileName)) {
                size_t name_length = 0;
                while (entry.cFileName[name_length] != L'\0') {
                    ++name_length;
                }
                if (length + 1 + name_length >=
                    (size_t)FY_STD_WINDOWS_PATH_CAPACITY_V0) {
                    FindClose(search);
                    return fy_std_windows_file_result_v0(
                        FY_STD_WINDOWS_FILE_PATH_TOO_LONG_V0,
                        ERROR_SUCCESS,
                        *completed
                    );
                }
                path[length] = L'\\';
                memcpy(
                    &path[length + 1],
                    entry.cFileName,
                    (name_length + 1) * sizeof entry.cFileName[0]
                );
                struct fy_std_windows_file_result_v0 child_result;
                if ((entry.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY) != 0) {
                    if ((entry.dwFileAttributes & FILE_ATTRIBUTE_REPARSE_POINT) != 0) {
                        if (!RemoveDirectoryW(path)) {
                            child_result = fy_std_windows_file_result_v0(
                                FY_STD_WINDOWS_FILE_IO_ERROR_V0,
                                GetLastError(),
                                *completed
                            );
                        } else {
                            child_result = fy_std_windows_directory_removed_v0(completed);
                        }
                    } else {
                        child_result = fy_std_windows_directory_remove_tree_wide_v0(
                            path,
                            length + 1 + name_length,
                            completed
                        );
                    }
                } else if (!DeleteFileW(path)) {
                    child_result = fy_std_windows_file_result_v0(
                        FY_STD_WINDOWS_FILE_IO_ERROR_V0,
                        GetLastError(),
                        *completed
                    );
                } else {
                    child_result = fy_std_windows_directory_removed_v0(completed);
                }
                path[length] = L'\0';
                if (child_result.status != FY_STD_WINDOWS_FILE_OK_V0) {
                    FindClose(search);
                    return child_result;
                }
            }
            if (!FindNextFileW(search, &entry)) {
                DWORD next_error = GetLastError();
                if (!FindClose(search)) {
                    return fy_std_windows_file_result_v0(
                        FY_STD_WINDOWS_FILE_IO_ERROR_V0,
                        GetLastError(),
                        *completed
                    );
                }
                if (next_error != ERROR_NO_MORE_FILES) {
                    return fy_std_windows_file_result_v0(
                        FY_STD_WINDOWS_FILE_IO_ERROR_V0,
                        next_error,
                        *completed
                    );
                }
                break;
            }
        }
    }
    if (!RemoveDirectoryW(path)) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            *completed
        );
    }
    return fy_std_windows_directory_removed_v0(completed);
}

struct fy_std_windows_file_result_v0 fy_std_windows_dir_remove_all_v0(
    const uint8_t *path
) {
    WCHAR wide_path[FY_STD_WINDOWS_PATH_CAPACITY_V0];
    uint32_t status = fy_std_windows_path_v0(path, wide_path);
    if (status != FY_STD_WINDOWS_FILE_OK_V0) {
        return fy_std_windows_file_result_v0(status, ERROR_SUCCESS, 0);
    }
    size_t length = 0;
    while (wide_path[length] != L'\0') {
        ++length;
    }
    size_t root_length = 0;
    status = fy_std_windows_directory_root_length_v0(wide_path, length, &root_length);
    if (status != FY_STD_WINDOWS_FILE_OK_V0 || length == root_length ||
        fy_std_windows_path_separator_v0(wide_path[length - 1])) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            0
        );
    }
    status = fy_std_windows_directory_components_valid_v0(wide_path, length, root_length);
    if (status != FY_STD_WINDOWS_FILE_OK_V0) {
        return fy_std_windows_file_result_v0(status, ERROR_SUCCESS, 0);
    }
    size_t component_start = root_length;
    while (component_start < length) {
        size_t component_end = component_start;
        while (component_end < length &&
               !fy_std_windows_path_separator_v0(wide_path[component_end])) {
            ++component_end;
        }
        WCHAR separator = wide_path[component_end];
        wide_path[component_end] = L'\0';
        DWORD attributes = GetFileAttributesW(wide_path);
        DWORD attribute_error = attributes == INVALID_FILE_ATTRIBUTES ? GetLastError() : ERROR_SUCCESS;
        wide_path[component_end] = separator;
        if (attributes == INVALID_FILE_ATTRIBUTES) {
            return fy_std_windows_file_result_v0(
                FY_STD_WINDOWS_FILE_IO_ERROR_V0,
                attribute_error,
                0
            );
        }
        if (component_end < length &&
            ((attributes & FILE_ATTRIBUTE_DIRECTORY) == 0 ||
             (attributes & FILE_ATTRIBUTE_REPARSE_POINT) != 0)) {
            return fy_std_windows_file_result_v0(
                FY_STD_WINDOWS_FILE_IO_ERROR_V0,
                ERROR_ACCESS_DENIED,
                0
            );
        }
        component_start = component_end + 1;
    }
    size_t completed = 0;
    return fy_std_windows_directory_remove_tree_wide_v0(wide_path, length, &completed);
}
