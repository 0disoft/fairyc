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
    FY_STD_WINDOWS_FILE_PATH_TOO_LONG_V0 = 6
};

struct fy_std_windows_file_result_v0 {
    uint32_t status;
    uint32_t os_code;
    size_t count;
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
           (options->read != UINT8_C(0) || options->write != UINT8_C(0)) &&
           (options->truncate == UINT8_C(0) || options->write != UINT8_C(0));
}

static DWORD fy_std_windows_creation_disposition_v0(
    const struct fy_std_windows_open_options_v0 *options
) {
    if (options->create != UINT8_C(0)) {
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
        access |= GENERIC_WRITE;
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

static struct fy_std_windows_file_result_v0 fy_std_windows_file_write_v0(
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
