struct fy_std_windows_file_attribute_result_v0 {
    uint32_t status;
    uint32_t os_code;
    uint32_t enabled;
};

static struct fy_std_windows_file_attribute_result_v0 fy_std_windows_file_attribute_result_v0(
    uint32_t status,
    DWORD os_code,
    int enabled
) {
    struct fy_std_windows_file_attribute_result_v0 result = {
        status,
        (uint32_t)os_code,
        enabled ? UINT32_C(1) : UINT32_C(0)
    };
    return result;
}

struct fy_std_windows_file_attribute_result_v0 fy_std_windows_file_read_only_v0(
    const void *file_storage
) {
    struct fy_std_windows_file_v0 file;
    if (!fy_std_windows_file_load_v0(file_storage, &file)) {
        return fy_std_windows_file_attribute_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            0
        );
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_windows_file_attribute_result_v0(
            FY_STD_WINDOWS_FILE_CLOSED_V0,
            ERROR_SUCCESS,
            0
        );
    }
    FILE_BASIC_INFO information;
    if (!GetFileInformationByHandleEx(
            (HANDLE)file.handle,
            FileBasicInfo,
            &information,
            sizeof information
        )) {
        return fy_std_windows_file_attribute_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            0
        );
    }
    return fy_std_windows_file_attribute_result_v0(
        FY_STD_WINDOWS_FILE_OK_V0,
        ERROR_SUCCESS,
        (information.FileAttributes & FILE_ATTRIBUTE_READONLY) != 0
    );
}

struct fy_std_windows_file_attribute_result_v0 fy_std_windows_file_set_readonly_v0(
    void *file_storage,
    uint32_t enabled
) {
    struct fy_std_windows_file_v0 file;
    if (!fy_std_windows_file_load_v0(file_storage, &file) || enabled > UINT32_C(1)) {
        return fy_std_windows_file_attribute_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            0
        );
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_windows_file_attribute_result_v0(
            FY_STD_WINDOWS_FILE_CLOSED_V0,
            ERROR_SUCCESS,
            0
        );
    }
    FILE_BASIC_INFO current;
    if (!GetFileInformationByHandleEx(
            (HANDLE)file.handle,
            FileBasicInfo,
            &current,
            sizeof current
        )) {
        return fy_std_windows_file_attribute_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            0
        );
    }
    DWORD attributes = current.FileAttributes;
    if (enabled != UINT32_C(0)) {
        attributes &= ~FILE_ATTRIBUTE_NORMAL;
        attributes |= FILE_ATTRIBUTE_READONLY;
    } else {
        attributes &= ~FILE_ATTRIBUTE_READONLY;
        if (attributes == 0) {
            attributes = FILE_ATTRIBUTE_NORMAL;
        }
    }
    FILE_BASIC_INFO update = {0};
    update.FileAttributes = attributes;
    if (!SetFileInformationByHandle(
            (HANDLE)file.handle,
            FileBasicInfo,
            &update,
            sizeof update
        )) {
        return fy_std_windows_file_attribute_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            0
        );
    }
    return fy_std_windows_file_attribute_result_v0(
        FY_STD_WINDOWS_FILE_OK_V0,
        ERROR_SUCCESS,
        enabled != UINT32_C(0)
    );
}

struct fy_std_windows_file_size_result_v0 fy_std_windows_file_link_count_v0(
    const void *file_storage
) {
    struct fy_std_windows_file_v0 file;
    if (!fy_std_windows_file_load_v0(file_storage, &file)) {
        return fy_std_windows_file_size_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            UINT64_C(0)
        );
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_windows_file_size_result_v0(
            FY_STD_WINDOWS_FILE_CLOSED_V0,
            ERROR_SUCCESS,
            UINT64_C(0)
        );
    }
    FILE_STANDARD_INFO information;
    if (!GetFileInformationByHandleEx(
            (HANDLE)file.handle,
            FileStandardInfo,
            &information,
            sizeof information
        )) {
        return fy_std_windows_file_size_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            UINT64_C(0)
        );
    }
    return fy_std_windows_file_size_result_v0(
        FY_STD_WINDOWS_FILE_OK_V0,
        ERROR_SUCCESS,
        (uint64_t)information.NumberOfLinks
    );
}

struct fy_std_windows_file_size_result_v0 fy_std_windows_file_alloc_size_v0(
    const void *file_storage
) {
    struct fy_std_windows_file_v0 file;
    if (!fy_std_windows_file_load_v0(file_storage, &file)) {
        return fy_std_windows_file_size_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            UINT64_C(0)
        );
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_windows_file_size_result_v0(
            FY_STD_WINDOWS_FILE_CLOSED_V0,
            ERROR_SUCCESS,
            UINT64_C(0)
        );
    }
    FILE_STANDARD_INFO information;
    if (!GetFileInformationByHandleEx(
            (HANDLE)file.handle,
            FileStandardInfo,
            &information,
            sizeof information
        )) {
        return fy_std_windows_file_size_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            UINT64_C(0)
        );
    }
    if (information.AllocationSize.QuadPart < 0) {
        return fy_std_windows_file_size_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            ERROR_ARITHMETIC_OVERFLOW,
            UINT64_C(0)
        );
    }
    return fy_std_windows_file_size_result_v0(
        FY_STD_WINDOWS_FILE_OK_V0,
        ERROR_SUCCESS,
        (uint64_t)information.AllocationSize.QuadPart
    );
}
