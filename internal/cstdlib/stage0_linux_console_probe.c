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

struct fy_std_console_write_result_v0 fy_std_stderr_write_all_v0(
    const uint8_t *data,
    size_t length
);

typedef struct fy_std_console_write_result_v0 (*fy_std_console_write_all_fn_v0)(
    const uint8_t *,
    size_t
);

static int fy_std_probe_console_stream_v0(
    int descriptor,
    fy_std_console_write_all_fn_v0 write_all,
    int failure_base
) {
    static const uint8_t payload[] = {UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121)};
    struct fy_std_console_write_result_v0 empty = write_all(NULL, (size_t)0);
    struct fy_std_console_write_result_v0 invalid = write_all(NULL, (size_t)1);
    if (empty.status != UINT32_C(0) || empty.written != (size_t)0) {
        return failure_base + 1;
    }
    if (invalid.status != UINT32_C(1) || invalid.written != (size_t)0) {
        return failure_base + 2;
    }

    int saved_descriptor = dup(descriptor);
    int sink = open("/dev/null", O_WRONLY);
    if (saved_descriptor < 0 || sink < 0 || dup2(sink, descriptor) < 0 || close(sink) != 0) {
        return failure_base + 3;
    }
    struct fy_std_console_write_result_v0 complete = write_all(payload, sizeof(payload));
    if (complete.status != UINT32_C(0) || complete.written != sizeof(payload)) {
        return failure_base + 4;
    }
    if (close(descriptor) != 0) {
        return failure_base + 5;
    }
    struct fy_std_console_write_result_v0 unavailable = write_all(payload, sizeof(payload));
    if (dup2(saved_descriptor, descriptor) < 0 || close(saved_descriptor) != 0) {
        return failure_base + 6;
    }
    if (unavailable.status != UINT32_C(2) || unavailable.written != (size_t)0) {
        return failure_base + 7;
    }
    return 0;
}

int main(void) {
    int stdout_result = fy_std_probe_console_stream_v0(
        STDOUT_FILENO,
        fy_std_stdout_write_all_v0,
        0
    );
    if (stdout_result != 0) {
        return stdout_result;
    }
    return fy_std_probe_console_stream_v0(
        STDERR_FILENO,
        fy_std_stderr_write_all_v0,
        10
    );
}
