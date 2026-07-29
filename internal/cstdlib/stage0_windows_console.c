#define WIN32_LEAN_AND_MEAN
#include <windows.h>

#include <stddef.h>
#include <stdint.h>

struct fy_std_console_write_result_v0 {
    uint32_t status;
    size_t written;
};

struct fy_std_console_read_result_v0 {
    uint32_t status;
    size_t read;
};

enum {
    FY_STD_WINDOWS_WRITE_OK_V0 = 0,
    FY_STD_WINDOWS_WRITE_INVALID_INPUT_V0 = 1,
    FY_STD_WINDOWS_WRITE_STREAM_UNAVAILABLE_V0 = 2,
    FY_STD_WINDOWS_WRITE_FAILED_V0 = 3,
    FY_STD_WINDOWS_WRITE_NO_PROGRESS_V0 = 4
};

enum {
    FY_STD_WINDOWS_READ_OK_V0 = 0,
    FY_STD_WINDOWS_READ_INVALID_INPUT_V0 = 1,
    FY_STD_WINDOWS_READ_STREAM_UNAVAILABLE_V0 = 2,
    FY_STD_WINDOWS_READ_FAILED_V0 = 3
};

static struct fy_std_console_write_result_v0 fy_std_windows_write_all_impl_v0(
    DWORD standard_handle,
    const uint8_t *data,
    size_t length
) {
    struct fy_std_console_write_result_v0 result = {
        FY_STD_WINDOWS_WRITE_OK_V0,
        (size_t)0
    };
    HANDLE output;

    if (length == (size_t)0) {
        return result;
    }
    if (data == NULL) {
        result.status = FY_STD_WINDOWS_WRITE_INVALID_INPUT_V0;
        return result;
    }
    output = GetStdHandle(standard_handle);
    if (output == NULL || output == INVALID_HANDLE_VALUE) {
        result.status = FY_STD_WINDOWS_WRITE_STREAM_UNAVAILABLE_V0;
        return result;
    }
    while (result.written < length) {
        size_t remaining = length - result.written;
        DWORD requested = remaining > (size_t)UINT32_MAX ? UINT32_MAX : (DWORD)remaining;
        DWORD completed = 0;
        BOOL succeeded = WriteFile(output, data + result.written, requested, &completed, NULL);
        if (!succeeded) {
            result.status = FY_STD_WINDOWS_WRITE_FAILED_V0;
            return result;
        }
        if (completed == 0) {
            result.status = FY_STD_WINDOWS_WRITE_NO_PROGRESS_V0;
            return result;
        }
        result.written += (size_t)completed;
    }
    return result;
}

struct fy_std_console_write_result_v0 fy_std_stdout_write_all_v0(
    const uint8_t *data,
    size_t length
) {
    return fy_std_windows_write_all_impl_v0(STD_OUTPUT_HANDLE, data, length);
}

struct fy_std_console_write_result_v0 fy_std_stdout_write_i32_v0(int32_t value) {
    uint8_t buffer[11];
    size_t position = sizeof(buffer);
    uint32_t magnitude = value < INT32_C(0)
        ? UINT32_C(0) - (uint32_t)value
        : (uint32_t)value;

    do {
        uint32_t digit = magnitude % UINT32_C(10);
        buffer[--position] = (uint8_t)(UINT8_C(48) + (uint8_t)digit);
        magnitude /= UINT32_C(10);
    } while (magnitude != UINT32_C(0));
    if (value < INT32_C(0)) {
        buffer[--position] = UINT8_C(45);
    }
    return fy_std_windows_write_all_impl_v0(
        STD_OUTPUT_HANDLE,
        buffer + position,
        sizeof(buffer) - position
    );
}

struct fy_std_console_write_result_v0 fy_std_stderr_write_all_v0(
    const uint8_t *data,
    size_t length
) {
    return fy_std_windows_write_all_impl_v0(STD_ERROR_HANDLE, data, length);
}

struct fy_std_console_read_result_v0 fy_std_stdin_read_v0(
    uint8_t *data,
    size_t length
) {
    struct fy_std_console_read_result_v0 result = {
        FY_STD_WINDOWS_READ_OK_V0,
        (size_t)0
    };
    HANDLE input;
    DWORD requested;
    DWORD completed = 0;
    BOOL succeeded;

    if (length == (size_t)0) {
        return result;
    }
    if (data == NULL) {
        result.status = FY_STD_WINDOWS_READ_INVALID_INPUT_V0;
        return result;
    }
    input = GetStdHandle(STD_INPUT_HANDLE);
    if (input == NULL || input == INVALID_HANDLE_VALUE) {
        result.status = FY_STD_WINDOWS_READ_STREAM_UNAVAILABLE_V0;
        return result;
    }
    requested = length > (size_t)UINT32_MAX ? UINT32_MAX : (DWORD)length;
    succeeded = ReadFile(input, data, requested, &completed, NULL);
    if (!succeeded) {
        DWORD error = GetLastError();
        if (error == ERROR_BROKEN_PIPE || error == ERROR_HANDLE_EOF) {
            return result;
        }
        result.status = error == ERROR_INVALID_HANDLE
            ? FY_STD_WINDOWS_READ_STREAM_UNAVAILABLE_V0
            : FY_STD_WINDOWS_READ_FAILED_V0;
        return result;
    }
    result.read = (size_t)completed;
    return result;
}

struct fy_std_console_write_result_v0 fy_std_windows_stdout_write_all_v0(
    const uint8_t *data,
    size_t length
) {
    return fy_std_windows_write_all_impl_v0(STD_OUTPUT_HANDLE, data, length);
}
