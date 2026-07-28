#define _POSIX_C_SOURCE 200809L

#include <fcntl.h>
#include <stddef.h>
#include <stdint.h>
#include <string.h>
#include <unistd.h>

struct fy_std_console_write_result_v0 {
    uint32_t status;
    size_t written;
};

struct fy_std_console_read_result_v0 {
    uint32_t status;
    size_t read;
};

struct fy_std_console_write_result_v0 fy_std_stdout_write_all_v0(
    const uint8_t *data,
    size_t length
);

struct fy_std_console_write_result_v0 fy_std_stderr_write_all_v0(
    const uint8_t *data,
    size_t length
);

struct fy_std_console_read_result_v0 fy_std_stdin_read_v0(
    uint8_t *data,
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

static int fy_std_probe_stdin_v0(void) {
    static const uint8_t payload[] = {UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121)};
    uint8_t buffer[8];
    int saved_descriptor;
    int pipe_descriptors[2];
    struct fy_std_console_read_result_v0 empty = fy_std_stdin_read_v0(NULL, (size_t)0);
    struct fy_std_console_read_result_v0 invalid = fy_std_stdin_read_v0(NULL, (size_t)1);
    struct fy_std_console_read_result_v0 complete;
    struct fy_std_console_read_result_v0 end;
    struct fy_std_console_read_result_v0 unavailable;

    if (empty.status != UINT32_C(0) || empty.read != (size_t)0) {
        return 21;
    }
    if (invalid.status != UINT32_C(1) || invalid.read != (size_t)0) {
        return 22;
    }
    saved_descriptor = dup(STDIN_FILENO);
    if (saved_descriptor < 0 || pipe(pipe_descriptors) != 0) {
        return 23;
    }
    if (write(pipe_descriptors[1], payload, sizeof(payload)) != (ssize_t)sizeof(payload) ||
        close(pipe_descriptors[1]) != 0 || dup2(pipe_descriptors[0], STDIN_FILENO) < 0 ||
        close(pipe_descriptors[0]) != 0) {
        return 24;
    }
    memset(buffer, 0xcc, sizeof(buffer));
    complete = fy_std_stdin_read_v0(buffer, sizeof(buffer));
    if (complete.status != UINT32_C(0) || complete.read != sizeof(payload) ||
        memcmp(buffer, payload, sizeof(payload)) != 0 || buffer[sizeof(payload)] != UINT8_C(0xcc)) {
        return 25;
    }
    end = fy_std_stdin_read_v0(buffer, sizeof(buffer));
    if (end.status != UINT32_C(0) || end.read != (size_t)0) {
        return 26;
    }
    if (close(STDIN_FILENO) != 0) {
        return 27;
    }
    unavailable = fy_std_stdin_read_v0(buffer, (size_t)1);
    if (dup2(saved_descriptor, STDIN_FILENO) < 0 || close(saved_descriptor) != 0) {
        return 28;
    }
    if (unavailable.status != UINT32_C(2) || unavailable.read != (size_t)0) {
        return 29;
    }
    return 0;
}

int main(void) {
    int stdout_result = fy_std_probe_console_stream_v0(
        STDOUT_FILENO,
        fy_std_stdout_write_all_v0,
        0
    );
    int stderr_result;
    if (stdout_result != 0) {
        return stdout_result;
    }
    stderr_result = fy_std_probe_console_stream_v0(
        STDERR_FILENO,
        fy_std_stderr_write_all_v0,
        10
    );
    if (stderr_result != 0) {
        return stderr_result;
    }
    return fy_std_probe_stdin_v0();
}
