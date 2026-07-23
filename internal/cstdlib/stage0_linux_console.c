#define _POSIX_C_SOURCE 200809L

#include <errno.h>
#include <limits.h>
#include <stddef.h>
#include <stdint.h>
#include <unistd.h>

struct fy_std_console_write_result_v0 {
    uint32_t status;
    size_t written;
};

enum {
    FY_STD_CONSOLE_WRITE_OK_V0 = 0,
    FY_STD_CONSOLE_WRITE_INVALID_INPUT_V0 = 1,
    FY_STD_CONSOLE_WRITE_STDOUT_UNAVAILABLE_V0 = 2,
    FY_STD_CONSOLE_WRITE_FAILED_V0 = 3,
    FY_STD_CONSOLE_WRITE_NO_PROGRESS_V0 = 4
};

struct fy_std_console_write_result_v0 fy_std_stdout_write_all_v0(
    const uint8_t *data,
    size_t length
) {
    struct fy_std_console_write_result_v0 result = {
        FY_STD_CONSOLE_WRITE_OK_V0,
        (size_t)0
    };

    if (length == (size_t)0) {
        return result;
    }
    if (data == NULL) {
        result.status = FY_STD_CONSOLE_WRITE_INVALID_INPUT_V0;
        return result;
    }
    while (result.written < length) {
        size_t remaining = length - result.written;
        size_t requested = remaining > (size_t)SSIZE_MAX ? (size_t)SSIZE_MAX : remaining;
        ssize_t completed;
        do {
            completed = write(STDOUT_FILENO, data + result.written, requested);
        } while (completed < 0 && errno == EINTR);
        if (completed < 0) {
            result.status = errno == EBADF
                ? FY_STD_CONSOLE_WRITE_STDOUT_UNAVAILABLE_V0
                : FY_STD_CONSOLE_WRITE_FAILED_V0;
            return result;
        }
        if (completed == 0) {
            result.status = FY_STD_CONSOLE_WRITE_NO_PROGRESS_V0;
            return result;
        }
        result.written += (size_t)completed;
    }
    return result;
}
