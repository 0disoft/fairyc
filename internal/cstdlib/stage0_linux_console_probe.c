#define _POSIX_C_SOURCE 200809L

#include <fcntl.h>
#include <stddef.h>
#include <stdint.h>
#include <unistd.h>

struct fy_std_console_write_result_v0 {
    uint32_t status;
    size_t written;
};

struct fy_std_console_write_result_v0 fy_std_stdout_write_all_v0(
    const uint8_t *data,
    size_t length
);

int main(void) {
    static const uint8_t payload[] = {UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121)};
    struct fy_std_console_write_result_v0 empty = fy_std_stdout_write_all_v0(NULL, (size_t)0);
    struct fy_std_console_write_result_v0 invalid = fy_std_stdout_write_all_v0(NULL, (size_t)1);
    if (empty.status != UINT32_C(0) || empty.written != (size_t)0) {
        return 1;
    }
    if (invalid.status != UINT32_C(1) || invalid.written != (size_t)0) {
        return 2;
    }

    int saved_stdout = dup(STDOUT_FILENO);
    int sink = open("/dev/null", O_WRONLY);
    if (saved_stdout < 0 || sink < 0 || dup2(sink, STDOUT_FILENO) < 0 || close(sink) != 0) {
        return 3;
    }
    struct fy_std_console_write_result_v0 complete = fy_std_stdout_write_all_v0(payload, sizeof(payload));
    if (complete.status != UINT32_C(0) || complete.written != sizeof(payload)) {
        return 4;
    }
    if (close(STDOUT_FILENO) != 0) {
        return 5;
    }
    struct fy_std_console_write_result_v0 unavailable = fy_std_stdout_write_all_v0(payload, sizeof(payload));
    if (dup2(saved_stdout, STDOUT_FILENO) < 0 || close(saved_stdout) != 0) {
        return 6;
    }
    if (unavailable.status != UINT32_C(2) || unavailable.written != (size_t)0) {
        return 7;
    }
    return 0;
}
