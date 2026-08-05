#include <stddef.h>
#include <stdint.h>
#include <string.h>
#include <windows.h>

enum fy_std_windows_file_status_v0 {
    FY_STD_WINDOWS_FILE_OK_V0 = 0,
    FY_STD_WINDOWS_FILE_IO_ERROR_V0 = 1,
    FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 = 2,
    FY_STD_WINDOWS_FILE_CLOSED_V0 = 3,
    FY_STD_WINDOWS_FILE_NO_PROGRESS_V0 = 4,
    FY_STD_WINDOWS_FILE_PATH_ENCODING_V0 = 5,
    FY_STD_WINDOWS_FILE_PATH_TOO_LONG_V0 = 6,
    FY_STD_WINDOWS_FILE_UNEXPECTED_EOF_V0 = 7
};

struct fy_std_windows_file_result_v0 {
    uint32_t status;
    uint32_t os_code;
    size_t count;
};

struct fy_std_windows_file_position_result_v0 {
    uint32_t status;
    uint32_t os_code;
    uint64_t position;
};

struct fy_std_windows_file_size_result_v0 {
    uint32_t status;
    uint32_t os_code;
    uint64_t size;
};

struct fy_std_windows_file_kind_result_v0 {
    uint32_t status;
    uint32_t os_code;
    uint32_t kind;
};

struct fy_std_windows_file_id_v0 {
    uint32_t volume_low;
    uint32_t volume_high;
    uint32_t file_low_0;
    uint32_t file_low_1;
    uint32_t file_high_0;
    uint32_t file_high_1;
};

struct fy_std_windows_file_v0 {
    void *handle;
    uint8_t open;
};

struct fy_std_windows_open_options_v0 {
    uint8_t read;
    uint8_t write;
    uint8_t create;
    uint8_t truncate;
    uint8_t exclusive;
    uint8_t append;
};

enum { FY_STD_WINDOWS_PATH_CAPACITY_V0 = 4096 };

static struct fy_std_windows_file_result_v0 fy_std_windows_file_result_v0(
    uint32_t status,
    DWORD os_code,
    size_t count
) {
    struct fy_std_windows_file_result_v0 result = {status, (uint32_t)os_code, count};
    return result;
}

static struct fy_std_windows_file_position_result_v0 fy_std_windows_file_position_result_v0(
    uint32_t status,
    DWORD os_code,
    uint64_t position
) {
    struct fy_std_windows_file_position_result_v0 result = {
        status,
        (uint32_t)os_code,
        position
    };
    return result;
}

static struct fy_std_windows_file_size_result_v0 fy_std_windows_file_size_result_v0(
    uint32_t status,
    DWORD os_code,
    uint64_t size
) {
    struct fy_std_windows_file_size_result_v0 result = {
        status,
        (uint32_t)os_code,
        size
    };
    return result;
}

static struct fy_std_windows_file_kind_result_v0 fy_std_windows_file_kind_result_v0(
    uint32_t status,
    DWORD os_code,
    DWORD kind
) {
    struct fy_std_windows_file_kind_result_v0 result = {
        status,
        (uint32_t)os_code,
        (uint32_t)kind
    };
    return result;
}

static int fy_std_windows_file_valid_v0(const struct fy_std_windows_file_v0 *file) {
    return file != NULL &&
           ((file->open == UINT8_C(0) && file->handle == NULL) ||
            (file->open == UINT8_C(1) && file->handle != NULL &&
             file->handle != INVALID_HANDLE_VALUE));
}

static int fy_std_windows_file_load_v0(
    const void *storage,
    struct fy_std_windows_file_v0 *file
) {
    if (storage == NULL || file == NULL) {
        return 0;
    }
    memcpy(file, storage, sizeof *file);
    return fy_std_windows_file_valid_v0(file);
}

static void fy_std_windows_file_store_v0(
    void *storage,
    const struct fy_std_windows_file_v0 *file
) {
    memcpy(storage, file, sizeof *file);
}

static uint32_t fy_std_windows_path_v0(const uint8_t *path, WCHAR *wide_path) {
    if (path == NULL || wide_path == NULL) {
        return FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0;
    }
    for (size_t index = 0; index < (size_t)FY_STD_WINDOWS_PATH_CAPACITY_V0; ++index) {
        uint8_t value = path[index];
        if (value == UINT8_C(0)) {
            if (index == 0) {
                return FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0;
            }
            wide_path[index] = L'\0';
            return FY_STD_WINDOWS_FILE_OK_V0;
        }
        if (value > UINT8_C(0x7f)) {
            return FY_STD_WINDOWS_FILE_PATH_ENCODING_V0;
        }
        wide_path[index] = (WCHAR)value;
    }
    return FY_STD_WINDOWS_FILE_PATH_TOO_LONG_V0;
}

static int fy_std_windows_open_options_valid_v0(
    const struct fy_std_windows_open_options_v0 *options
) {
    return options != NULL && options->read <= UINT8_C(1) && options->write <= UINT8_C(1) &&
           options->create <= UINT8_C(1) && options->truncate <= UINT8_C(1) &&
           options->exclusive <= UINT8_C(1) && options->append <= UINT8_C(1) &&
           (options->read != UINT8_C(0) || options->write != UINT8_C(0)) &&
           (options->truncate == UINT8_C(0) || options->write != UINT8_C(0)) &&
           (options->exclusive == UINT8_C(0) ||
            (options->write != UINT8_C(0) && options->create != UINT8_C(0) &&
             options->truncate == UINT8_C(0))) &&
           (options->append == UINT8_C(0) ||
            (options->write != UINT8_C(0) && options->truncate == UINT8_C(0) &&
             options->exclusive == UINT8_C(0)));
}

static DWORD fy_std_windows_creation_disposition_v0(
    const struct fy_std_windows_open_options_v0 *options
) {
    if (options->create != UINT8_C(0)) {
        if (options->exclusive != UINT8_C(0)) {
            return CREATE_NEW;
        }
        return options->truncate != UINT8_C(0) ? CREATE_ALWAYS : OPEN_ALWAYS;
    }
    return options->truncate != UINT8_C(0) ? TRUNCATE_EXISTING : OPEN_EXISTING;
}

struct fy_std_windows_file_result_v0 fy_std_windows_file_open_v0(
    const uint8_t *path,
    const struct fy_std_windows_open_options_v0 *options,
    void *output_storage
) {
    WCHAR wide_path[FY_STD_WINDOWS_PATH_CAPACITY_V0];
    uint32_t path_status = fy_std_windows_path_v0(path, wide_path);
    if (path_status != FY_STD_WINDOWS_FILE_OK_V0) {
        return fy_std_windows_file_result_v0(path_status, ERROR_SUCCESS, 0);
    }
    struct fy_std_windows_file_v0 output;
    if (!fy_std_windows_open_options_valid_v0(options) ||
        !fy_std_windows_file_load_v0(output_storage, &output) || output.open != UINT8_C(0)) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            0
        );
    }
    DWORD access = 0;
    if (options->read != UINT8_C(0)) {
        access |= GENERIC_READ;
    }
    if (options->write != UINT8_C(0)) {
        access |= options->append != UINT8_C(0) ? FILE_APPEND_DATA : GENERIC_WRITE;
    }
    HANDLE handle = CreateFileW(
        wide_path,
        access,
        FILE_SHARE_READ,
        NULL,
        fy_std_windows_creation_disposition_v0(options),
        FILE_ATTRIBUTE_NORMAL,
        NULL
    );
    if (handle == INVALID_HANDLE_VALUE) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            0
        );
    }
    output.handle = handle;
    output.open = UINT8_C(1);
    fy_std_windows_file_store_v0(output_storage, &output);
    return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS, 0);
}

struct fy_std_windows_file_result_v0 fy_std_windows_file_remove_v0(
    const uint8_t *path
) {
    WCHAR wide_path[FY_STD_WINDOWS_PATH_CAPACITY_V0];
    uint32_t path_status = fy_std_windows_path_v0(path, wide_path);
    if (path_status != FY_STD_WINDOWS_FILE_OK_V0) {
        return fy_std_windows_file_result_v0(path_status, ERROR_SUCCESS, 0);
    }
    if (!DeleteFileW(wide_path)) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            0
        );
    }
    return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS, 0);
}

struct fy_std_windows_file_result_v0 fy_std_windows_file_rename_v0(
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
    DWORD attributes = GetFileAttributesW(wide_source);
    if (attributes == INVALID_FILE_ATTRIBUTES) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            0
        );
    }
    if ((attributes & FILE_ATTRIBUTE_DIRECTORY) != 0) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            0
        );
    }
    if (!MoveFileExW(wide_source, wide_destination, 0)) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            0
        );
    }
    return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS, 0);
}

struct fy_std_windows_file_result_v0 fy_std_windows_file_read_v0(
    void *file_storage,
    uint8_t *buffer,
    size_t length
) {
    struct fy_std_windows_file_v0 file;
    if (!fy_std_windows_file_load_v0(file_storage, &file)) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            0
        );
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_CLOSED_V0, ERROR_SUCCESS, 0);
    }
    if (length != 0 && buffer == NULL) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            0
        );
    }
    if (length == 0) {
        return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS, 0);
    }
    DWORD requested = length > (size_t)MAXDWORD ? MAXDWORD : (DWORD)length;
    DWORD completed = 0;
    if (!ReadFile((HANDLE)file.handle, buffer, requested, &completed, NULL)) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            0
        );
    }
    return fy_std_windows_file_result_v0(
        FY_STD_WINDOWS_FILE_OK_V0,
        ERROR_SUCCESS,
        (size_t)completed
    );
}

struct fy_std_windows_file_result_v0 fy_std_windows_file_read_exact_v0(
    void *file_storage,
    uint8_t *buffer,
    size_t length
) {
    struct fy_std_windows_file_result_v0 result = fy_std_windows_file_read_v0(
        file_storage,
        buffer,
        length
    );
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || length == 0) {
        return result;
    }
    if (result.count == 0) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_UNEXPECTED_EOF_V0,
            ERROR_SUCCESS,
            0
        );
    }
    if (result.count > length) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            0
        );
    }
    size_t total = result.count;
    while (total < length) {
        result = fy_std_windows_file_read_v0(
            file_storage,
            buffer + total,
            length - total
        );
        if (result.status != FY_STD_WINDOWS_FILE_OK_V0) {
            return fy_std_windows_file_result_v0(
                result.status,
                (DWORD)result.os_code,
                total + result.count
            );
        }
        if (result.count == 0) {
            return fy_std_windows_file_result_v0(
                FY_STD_WINDOWS_FILE_UNEXPECTED_EOF_V0,
                ERROR_SUCCESS,
                total
            );
        }
        if (result.count > length - total) {
            return fy_std_windows_file_result_v0(
                FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
                ERROR_SUCCESS,
                total
            );
        }
        total += result.count;
    }
    return fy_std_windows_file_result_v0(
        FY_STD_WINDOWS_FILE_OK_V0,
        ERROR_SUCCESS,
        total
    );
}

struct fy_std_windows_file_position_result_v0 fy_std_windows_file_position_v0(
    void *file_storage
) {
    struct fy_std_windows_file_v0 file;
    if (!fy_std_windows_file_load_v0(file_storage, &file)) {
        return fy_std_windows_file_position_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            UINT64_C(0)
        );
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_windows_file_position_result_v0(
            FY_STD_WINDOWS_FILE_CLOSED_V0,
            ERROR_SUCCESS,
            UINT64_C(0)
        );
    }
    LARGE_INTEGER distance;
    LARGE_INTEGER position;
    distance.QuadPart = 0;
    if (!SetFilePointerEx((HANDLE)file.handle, distance, &position, FILE_CURRENT)) {
        return fy_std_windows_file_position_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            UINT64_C(0)
        );
    }
    return fy_std_windows_file_position_result_v0(
        FY_STD_WINDOWS_FILE_OK_V0,
        ERROR_SUCCESS,
        (uint64_t)position.QuadPart
    );
}

struct fy_std_windows_file_size_result_v0 fy_std_windows_file_size_v0(
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
    LARGE_INTEGER size;
    if (!GetFileSizeEx((HANDLE)file.handle, &size)) {
        return fy_std_windows_file_size_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            UINT64_C(0)
        );
    }
    return fy_std_windows_file_size_result_v0(
        FY_STD_WINDOWS_FILE_OK_V0,
        ERROR_SUCCESS,
        (uint64_t)size.QuadPart
    );
}

struct fy_std_windows_file_kind_result_v0 fy_std_windows_file_kind_v0(
    const void *file_storage
) {
    struct fy_std_windows_file_v0 file;
    if (!fy_std_windows_file_load_v0(file_storage, &file)) {
        return fy_std_windows_file_kind_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            FILE_TYPE_UNKNOWN
        );
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_windows_file_kind_result_v0(
            FY_STD_WINDOWS_FILE_CLOSED_V0,
            ERROR_SUCCESS,
            FILE_TYPE_UNKNOWN
        );
    }
    SetLastError(ERROR_SUCCESS);
    DWORD kind = GetFileType((HANDLE)file.handle);
    if (kind == FILE_TYPE_UNKNOWN) {
        DWORD error = GetLastError();
        if (error != ERROR_SUCCESS) {
            return fy_std_windows_file_kind_result_v0(
                FY_STD_WINDOWS_FILE_IO_ERROR_V0,
                error,
                FILE_TYPE_UNKNOWN
            );
        }
    }
    return fy_std_windows_file_kind_result_v0(
        FY_STD_WINDOWS_FILE_OK_V0,
        ERROR_SUCCESS,
        kind
    );
}

struct fy_std_windows_file_result_v0 fy_std_windows_file_id_v0(
    const void *file_storage,
    struct fy_std_windows_file_id_v0 *output
) {
    struct fy_std_windows_file_v0 file;
    if (!fy_std_windows_file_load_v0(file_storage, &file) || output == NULL) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            0
        );
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_CLOSED_V0, ERROR_SUCCESS, 0);
    }
    FILE_ID_INFO information;
    if (!GetFileInformationByHandleEx(
            (HANDLE)file.handle,
            FileIdInfo,
            &information,
            sizeof information
        )) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            0
        );
    }
    struct fy_std_windows_file_id_v0 value = {
        (uint32_t)(information.VolumeSerialNumber & UINT64_C(0xffffffff)),
        (uint32_t)(information.VolumeSerialNumber >> 32),
        UINT32_C(0),
        UINT32_C(0),
        UINT32_C(0),
        UINT32_C(0)
    };
    memcpy(&value.file_low_0, &information.FileId.Identifier[0], sizeof value.file_low_0);
    memcpy(&value.file_low_1, &information.FileId.Identifier[4], sizeof value.file_low_1);
    memcpy(&value.file_high_0, &information.FileId.Identifier[8], sizeof value.file_high_0);
    memcpy(&value.file_high_1, &information.FileId.Identifier[12], sizeof value.file_high_1);
    memcpy(output, &value, sizeof value);
    return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS, 0);
}

struct fy_std_windows_file_size_result_v0 fy_std_windows_file_set_len_v0(
    void *file_storage,
    uint64_t size
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
    if (size > (uint64_t)INT64_MAX) {
        return fy_std_windows_file_size_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            UINT64_C(0)
        );
    }
    FILE_END_OF_FILE_INFO information;
    information.EndOfFile.QuadPart = (LONGLONG)size;
    if (!SetFileInformationByHandle(
            (HANDLE)file.handle,
            FileEndOfFileInfo,
            &information,
            (DWORD)sizeof information
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
        size
    );
}

struct fy_std_windows_file_position_result_v0 fy_std_windows_file_seek_start_v0(
    void *file_storage,
    uint64_t offset
) {
    struct fy_std_windows_file_v0 file;
    if (!fy_std_windows_file_load_v0(file_storage, &file)) {
        return fy_std_windows_file_position_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            UINT64_C(0)
        );
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_windows_file_position_result_v0(
            FY_STD_WINDOWS_FILE_CLOSED_V0,
            ERROR_SUCCESS,
            UINT64_C(0)
        );
    }
    if (offset > (uint64_t)INT64_MAX) {
        return fy_std_windows_file_position_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            UINT64_C(0)
        );
    }
    LARGE_INTEGER distance;
    LARGE_INTEGER position;
    distance.QuadPart = (LONGLONG)offset;
    if (!SetFilePointerEx((HANDLE)file.handle, distance, &position, FILE_BEGIN)) {
        return fy_std_windows_file_position_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            UINT64_C(0)
        );
    }
    return fy_std_windows_file_position_result_v0(
        FY_STD_WINDOWS_FILE_OK_V0,
        ERROR_SUCCESS,
        (uint64_t)position.QuadPart
    );
}

struct fy_std_windows_file_result_v0 fy_std_windows_file_write_v0(
    void *file_storage,
    const uint8_t *buffer,
    size_t length
) {
    struct fy_std_windows_file_v0 file;
    if (!fy_std_windows_file_load_v0(file_storage, &file)) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            0
        );
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_CLOSED_V0, ERROR_SUCCESS, 0);
    }
    if (length != 0 && buffer == NULL) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            0
        );
    }
    if (length == 0) {
        return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS, 0);
    }
    DWORD requested = length > (size_t)MAXDWORD ? MAXDWORD : (DWORD)length;
    DWORD completed = 0;
    if (!WriteFile((HANDLE)file.handle, buffer, requested, &completed, NULL)) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            0
        );
    }
    return fy_std_windows_file_result_v0(
        FY_STD_WINDOWS_FILE_OK_V0,
        ERROR_SUCCESS,
        (size_t)completed
    );
}

struct fy_std_windows_file_result_v0 fy_std_windows_file_write_all_v0(
    void *file_storage,
    const uint8_t *buffer,
    size_t length
) {
    size_t written = 0;
    while (written < length) {
        struct fy_std_windows_file_result_v0 result = fy_std_windows_file_write_v0(
            file_storage,
            buffer + written,
            length - written
        );
        if (result.status != FY_STD_WINDOWS_FILE_OK_V0) {
            result.count = written;
            return result;
        }
        if (result.count == 0) {
            return fy_std_windows_file_result_v0(
                FY_STD_WINDOWS_FILE_NO_PROGRESS_V0,
                ERROR_SUCCESS,
                written
            );
        }
        written += result.count;
    }
    return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS, written);
}

struct fy_std_windows_file_result_v0 fy_std_windows_file_sync_v0(void *file_storage) {
    struct fy_std_windows_file_v0 file;
    if (!fy_std_windows_file_load_v0(file_storage, &file)) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            0
        );
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_CLOSED_V0, ERROR_SUCCESS, 0);
    }
    if (!FlushFileBuffers((HANDLE)file.handle)) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            0
        );
    }
    return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS, 0);
}

struct fy_std_windows_file_result_v0 fy_std_windows_file_close_v0(void *file_storage) {
    struct fy_std_windows_file_v0 file;
    if (!fy_std_windows_file_load_v0(file_storage, &file)) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0,
            ERROR_SUCCESS,
            0
        );
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_CLOSED_V0, ERROR_SUCCESS, 0);
    }
    HANDLE handle = (HANDLE)file.handle;
    file.handle = NULL;
    file.open = UINT8_C(0);
    fy_std_windows_file_store_v0(file_storage, &file);
    if (!CloseHandle(handle)) {
        return fy_std_windows_file_result_v0(
            FY_STD_WINDOWS_FILE_IO_ERROR_V0,
            GetLastError(),
            0
        );
    }
    return fy_std_windows_file_result_v0(FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS, 0);
}

uint32_t fy_std_windows_file_drop_v0(void *file_storage) {
    struct fy_std_windows_file_v0 file;
    if (fy_std_windows_file_load_v0(file_storage, &file) && file.open != UINT8_C(0)) {
        HANDLE handle = (HANDLE)file.handle;
        file.handle = NULL;
        file.open = UINT8_C(0);
        fy_std_windows_file_store_v0(file_storage, &file);
        (void)CloseHandle(handle);
    }
    return UINT32_C(0);
}
