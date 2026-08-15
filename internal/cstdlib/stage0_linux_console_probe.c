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

struct fy_std_console_write_result_v0 fy_std_stdout_write_i32_v0(int32_t value);

struct fy_std_console_write_result_v0 fy_std_stdout_write_u32_v0(uint32_t value);

struct fy_std_console_write_result_v0 fy_std_stdout_write_usize_v0(uintptr_t value);

struct fy_std_console_write_result_v0 fy_std_stderr_write_all_v0(
    const uint8_t *data,
    size_t length
);

struct fy_std_console_write_result_v0 fy_std_stderr_write_i32_v0(int32_t value);

struct fy_std_console_write_result_v0 fy_std_stderr_write_u32_v0(uint32_t value);

struct fy_std_console_write_result_v0 fy_std_stderr_write_usize_v0(uintptr_t value);

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

static int fy_std_probe_stdout_i32_v0(void) {
    static const uint8_t expected[] = "0-42123456789-21474836482147483647";
    uint8_t observed[sizeof(expected) - 1];
    int saved_descriptor = dup(STDOUT_FILENO);
    int pipe_descriptors[2];
    size_t total = (size_t)0;
    struct fy_std_console_write_result_v0 results[5];

    if (saved_descriptor < 0 || pipe(pipe_descriptors) != 0) {
        return 31;
    }
    if (dup2(pipe_descriptors[1], STDOUT_FILENO) < 0 || close(pipe_descriptors[1]) != 0) {
        return 32;
    }
    results[0] = fy_std_stdout_write_i32_v0(INT32_C(0));
    results[1] = fy_std_stdout_write_i32_v0(-INT32_C(42));
    results[2] = fy_std_stdout_write_i32_v0(INT32_C(123456789));
    results[3] = fy_std_stdout_write_i32_v0(INT32_MIN);
    results[4] = fy_std_stdout_write_i32_v0(INT32_MAX);
    if (dup2(saved_descriptor, STDOUT_FILENO) < 0 || close(saved_descriptor) != 0) {
        return 33;
    }
    for (size_t index = (size_t)0; index < (size_t)5; ++index) {
        if (results[index].status != UINT32_C(0)) {
            return 34;
        }
    }
    if (results[0].written != (size_t)1 || results[1].written != (size_t)3 ||
        results[2].written != (size_t)9 || results[3].written != (size_t)11 ||
        results[4].written != (size_t)10) {
        return 35;
    }
    while (total < sizeof(observed)) {
        ssize_t completed = read(pipe_descriptors[0], observed + total, sizeof(observed) - total);
        if (completed <= 0) {
            return 36;
        }
        total += (size_t)completed;
    }
    if (close(pipe_descriptors[0]) != 0 || memcmp(observed, expected, sizeof(observed)) != 0) {
        return 37;
    }
    return 0;
}

static int fy_std_probe_stdout_unsigned_v0(void) {
    static const uint8_t expected[] = "0142949672950118446744073709551615";
    uint8_t observed[sizeof(expected) - 1];
    int saved_descriptor = dup(STDOUT_FILENO);
    int pipe_descriptors[2];
    size_t total = (size_t)0;
    struct fy_std_console_write_result_v0 results[6];

    if (saved_descriptor < 0 || pipe(pipe_descriptors) != 0) {
        return 41;
    }
    if (dup2(pipe_descriptors[1], STDOUT_FILENO) < 0 || close(pipe_descriptors[1]) != 0) {
        return 42;
    }
    results[0] = fy_std_stdout_write_u32_v0(UINT32_C(0));
    results[1] = fy_std_stdout_write_u32_v0(UINT32_C(1));
    results[2] = fy_std_stdout_write_u32_v0(UINT32_MAX);
    results[3] = fy_std_stdout_write_usize_v0((uintptr_t)0);
    results[4] = fy_std_stdout_write_usize_v0((uintptr_t)1);
    results[5] = fy_std_stdout_write_usize_v0((uintptr_t)UINT64_MAX);
    if (dup2(saved_descriptor, STDOUT_FILENO) < 0 || close(saved_descriptor) != 0) {
        return 43;
    }
    for (size_t index = (size_t)0; index < (size_t)6; ++index) {
        if (results[index].status != UINT32_C(0)) {
            return 44;
        }
    }
    if (results[0].written != (size_t)1 || results[1].written != (size_t)1 ||
        results[2].written != (size_t)10 || results[3].written != (size_t)1 ||
        results[4].written != (size_t)1 || results[5].written != (size_t)20) {
        return 45;
    }
    while (total < sizeof(observed)) {
        ssize_t completed = read(pipe_descriptors[0], observed + total, sizeof(observed) - total);
        if (completed <= 0) {
            return 46;
        }
        total += (size_t)completed;
    }
    if (close(pipe_descriptors[0]) != 0 || memcmp(observed, expected, sizeof(observed)) != 0) {
        return 47;
    }
    return 0;
}

struct fy_std_probe_expected_write_v0 {
    const uint8_t *bytes;
    size_t length;
};

static const uint8_t fy_std_probe_stderr_expected_i32_zero_v0[] = "0";
static const uint8_t fy_std_probe_stderr_expected_i32_negative_v0[] = "-42";
static const uint8_t fy_std_probe_stderr_expected_i32_min_v0[] = "-2147483648";
static const uint8_t fy_std_probe_stderr_expected_i32_max_v0[] = "2147483647";
static const uint8_t fy_std_probe_stderr_expected_u32_zero_v0[] = "0";
static const uint8_t fy_std_probe_stderr_expected_u32_max_v0[] = "4294967295";
static const uint8_t fy_std_probe_stderr_expected_usize_zero_v0[] = "0";
static const uint8_t fy_std_probe_stderr_expected_usize_max_v0[] = "18446744073709551615";

static const struct fy_std_probe_expected_write_v0 fy_std_probe_stderr_expected_v0[] = {
    {
        fy_std_probe_stderr_expected_i32_zero_v0,
        sizeof(fy_std_probe_stderr_expected_i32_zero_v0) - 1,
    },
    {
        fy_std_probe_stderr_expected_i32_negative_v0,
        sizeof(fy_std_probe_stderr_expected_i32_negative_v0) - 1,
    },
    {
        fy_std_probe_stderr_expected_i32_min_v0,
        sizeof(fy_std_probe_stderr_expected_i32_min_v0) - 1,
    },
    {
        fy_std_probe_stderr_expected_i32_max_v0,
        sizeof(fy_std_probe_stderr_expected_i32_max_v0) - 1,
    },
    {
        fy_std_probe_stderr_expected_u32_zero_v0,
        sizeof(fy_std_probe_stderr_expected_u32_zero_v0) - 1,
    },
    {
        fy_std_probe_stderr_expected_u32_max_v0,
        sizeof(fy_std_probe_stderr_expected_u32_max_v0) - 1,
    },
    {
        fy_std_probe_stderr_expected_usize_zero_v0,
        sizeof(fy_std_probe_stderr_expected_usize_zero_v0) - 1,
    },
    {
        fy_std_probe_stderr_expected_usize_max_v0,
        sizeof(fy_std_probe_stderr_expected_usize_max_v0) - 1,
    },
};

enum {
    FY_STD_PROBE_STDERR_EXPECTED_COUNT_V0 =
        sizeof(fy_std_probe_stderr_expected_v0) /
        sizeof(fy_std_probe_stderr_expected_v0[0]),
};

static int fy_std_probe_read_expected_v0(
    int descriptor,
    const struct fy_std_probe_expected_write_v0 *expected
) {
    uint8_t observed[sizeof(fy_std_probe_stderr_expected_usize_max_v0) - 1];
    size_t total = (size_t)0;
    while (total < expected->length) {
        ssize_t completed;
        do {
            completed = read(descriptor, observed + total, expected->length - total);
        } while (completed < 0 && errno == EINTR);
        if (completed <= 0) {
            return 56;
        }
        total += (size_t)completed;
    }
    return memcmp(observed, expected->bytes, expected->length) == 0 ? 0 : 57;
}

static int fy_std_probe_stderr_scalar_v0(void) {
    int saved_descriptor = dup(STDERR_FILENO);
    int pipe_descriptors[2];
    struct fy_std_console_write_result_v0 results[FY_STD_PROBE_STDERR_EXPECTED_COUNT_V0];
    int result = 0;

    if (saved_descriptor < 0 || pipe(pipe_descriptors) != 0) {
        return 51;
    }
    if (dup2(pipe_descriptors[1], STDERR_FILENO) < 0 || close(pipe_descriptors[1]) != 0) {
        return 52;
    }
    results[0] = fy_std_stderr_write_i32_v0(INT32_C(0));
    results[1] = fy_std_stderr_write_i32_v0(-INT32_C(42));
    results[2] = fy_std_stderr_write_i32_v0(INT32_MIN);
    results[3] = fy_std_stderr_write_i32_v0(INT32_MAX);
    results[4] = fy_std_stderr_write_u32_v0(UINT32_C(0));
    results[5] = fy_std_stderr_write_u32_v0(UINT32_MAX);
    results[6] = fy_std_stderr_write_usize_v0((uintptr_t)0);
    results[7] = fy_std_stderr_write_usize_v0((uintptr_t)UINT64_MAX);
    if (dup2(saved_descriptor, STDERR_FILENO) < 0 || close(saved_descriptor) != 0) {
        return 53;
    }
    for (
        size_t index = (size_t)0;
        index < (size_t)FY_STD_PROBE_STDERR_EXPECTED_COUNT_V0;
        ++index
    ) {
        if (results[index].status != UINT32_C(0)) {
            result = 54;
            break;
        }
        if (results[index].written != fy_std_probe_stderr_expected_v0[index].length) {
            result = 55;
            break;
        }
        result = fy_std_probe_read_expected_v0(
            pipe_descriptors[0],
            &fy_std_probe_stderr_expected_v0[index]
        );
        if (result != 0) {
            break;
        }
    }
    if (result == 0) {
        uint8_t unexpected;
        ssize_t completed;
        do {
            completed = read(pipe_descriptors[0], &unexpected, (size_t)1);
        } while (completed < 0 && errno == EINTR);
        if (completed != (ssize_t)0) {
            result = 58;
        }
    }
    if (close(pipe_descriptors[0]) != 0 && result == 0) {
        result = 59;
    }
    return result;
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
    int i32_result = fy_std_probe_stdout_i32_v0();
    if (i32_result != 0) {
        return i32_result;
    }
    int unsigned_result = fy_std_probe_stdout_unsigned_v0();
    if (unsigned_result != 0) {
        return unsigned_result;
    }
    int stderr_scalar_result = fy_std_probe_stderr_scalar_v0();
    if (stderr_scalar_result != 0) {
        return stderr_scalar_result;
    }
    return fy_std_probe_stdin_v0();
}
