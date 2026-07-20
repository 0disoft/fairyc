#define WIN32_LEAN_AND_MEAN
#include <windows.h>

#include <stddef.h>
#include <stdint.h>

struct fy_std_windows_write_result_v0 {
    uint32_t status;
    size_t written;
};

enum {
    FY_STD_WINDOWS_WRITE_OK_V0 = 0,
    FY_STD_WINDOWS_WRITE_INVALID_INPUT_V0 = 1,
    FY_STD_WINDOWS_WRITE_STDOUT_UNAVAILABLE_V0 = 2,
    FY_STD_WINDOWS_WRITE_FAILED_V0 = 3,
    FY_STD_WINDOWS_WRITE_NO_PROGRESS_V0 = 4
};

struct fy_std_windows_write_result_v0 fy_std_windows_stdout_write_all_v0(
    const uint8_t *data,
    size_t length
) {
    struct fy_std_windows_write_result_v0 result = {
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
    output = GetStdHandle(STD_OUTPUT_HANDLE);
    if (output == NULL || output == INVALID_HANDLE_VALUE) {
        result.status = FY_STD_WINDOWS_WRITE_STDOUT_UNAVAILABLE_V0;
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
