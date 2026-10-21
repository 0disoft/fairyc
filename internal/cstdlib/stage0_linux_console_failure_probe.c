#define _POSIX_C_SOURCE 200809L
#ifdef NDEBUG
#undef NDEBUG
#endif
#include <assert.h>
#include <errno.h>
#include <stdint.h>
#include <string.h>
#include <unistd.h>

struct fy_std_console_write_result_v0 { uint32_t status; size_t written; };
struct fy_std_console_read_result_v0 { uint32_t status; size_t read; };
struct fy_std_console_write_result_v0 fy_std_stdout_write_all_v0(const uint8_t *, size_t);
struct fy_std_console_write_result_v0 fy_std_stderr_write_all_v0(const uint8_t *, size_t);
struct fy_std_console_read_result_v0 fy_std_stdin_read_v0(uint8_t *, size_t);

/* Each entry specifies the exact syscall, not a substitute console implementation. */
struct fy_probe_step { size_t offset, requested; ssize_t result; int error; };
static const struct fy_probe_step *fy_steps;
static size_t fy_step_count, fy_calls, fy_extent, fy_progress;
static int fy_descriptor, fy_is_write;
static uint8_t *fy_base;
static uint8_t fy_observed[8];
static const uint8_t fy_payload[8] = {70, 0, 255, 97, 98, 99, 165, 165};

static void fy_prepare(const struct fy_probe_step *steps, size_t count,
    int descriptor, int is_write, uint8_t *base, size_t extent) {
    fy_steps = steps;
    fy_step_count = count;
    fy_calls = 0;
    fy_progress = 0;
    fy_descriptor = descriptor;
    fy_is_write = is_write;
    fy_base = base;
    fy_extent = extent;
    memset(fy_observed, 165, sizeof fy_observed);
}

static const struct fy_probe_step *fy_next(int descriptor, int is_write,
    const void *data, size_t requested) {
    assert(fy_calls < fy_step_count);
    const struct fy_probe_step *step = &fy_steps[fy_calls++];
    assert(descriptor == fy_descriptor && is_write == fy_is_write);
    assert(step->offset <= fy_extent && requested <= fy_extent - step->offset);
    assert(data == fy_base + step->offset && requested == step->requested);
    assert(step->result >= -1);
    assert(step->result < 0 || (size_t)step->result <= requested);
    errno = step->error;
    return step;
}

ssize_t fy_std_probe_write_v0(int descriptor, const void *data, size_t requested) {
    const struct fy_probe_step *step = fy_next(descriptor, 1, data, requested);
    assert(step->offset == fy_progress);
    assert(memcmp(data, fy_payload + step->offset, requested) == 0);
    if (step->result > 0) {
        assert(fy_progress + (size_t)step->result <= sizeof fy_observed);
        memcpy(fy_observed + fy_progress, data, (size_t)step->result);
        fy_progress += (size_t)step->result;
    }
    return step->result;
}

ssize_t fy_std_probe_read_v0(int descriptor, void *data, size_t requested) {
    const struct fy_probe_step *step = fy_next(descriptor, 0, data, requested);
    assert(step->offset == 0);
    for (size_t i = 0; i < requested; ++i) assert(((uint8_t *)data)[i] == 165);
    if (step->result > 0) {
        assert((size_t)step->result <= sizeof fy_payload);
        memcpy(data, fy_payload, (size_t)step->result);
    }
    return step->result;
}

static void fy_check_write(int descriptor, const struct fy_probe_step *steps,
    size_t count, uint32_t status, size_t written) {
    uint8_t data[8];
    memcpy(data, fy_payload, sizeof data);
    fy_prepare(steps, count, descriptor, 1, data, 6);
    struct fy_std_console_write_result_v0 result = descriptor == STDOUT_FILENO
        ? fy_std_stdout_write_all_v0(data, 6) : fy_std_stderr_write_all_v0(data, 6);
    assert(result.status == status && result.written == written);
    assert(fy_calls == count && fy_progress == written);
    assert(memcmp(data, fy_payload, sizeof data) == 0);
    assert(memcmp(fy_observed, fy_payload, written) == 0);
    for (size_t i = written; i < sizeof fy_observed; ++i) assert(fy_observed[i] == 165);
}

static void fy_check_write_no_io(int descriptor) {
    uint8_t byte = 165;
    fy_prepare(NULL, 0, descriptor, 1, &byte, 1);
    struct fy_std_console_write_result_v0 (*write_all)(const uint8_t *, size_t) =
        descriptor == STDOUT_FILENO ? fy_std_stdout_write_all_v0 : fy_std_stderr_write_all_v0;
    struct fy_std_console_write_result_v0 r = write_all(&byte, 0);
    assert(r.status == 0 && r.written == 0);
    r = write_all(NULL, 0);
    assert(r.status == 0 && r.written == 0);
    r = write_all(NULL, 1);
    assert(r.status == 1 && r.written == 0 && byte == 165 && fy_calls == 0);
}

static void fy_check_read(const struct fy_probe_step *steps, size_t count,
    uint32_t status, size_t length) {
    uint8_t guarded[10];
    memset(guarded, 165, sizeof guarded);
    fy_prepare(steps, count, STDIN_FILENO, 0, guarded + 1, 8);
    struct fy_std_console_read_result_v0 result = fy_std_stdin_read_v0(guarded + 1, 8);
    assert(result.status == status && result.read == length && fy_calls == count);
    assert(guarded[0] == 165 && memcmp(guarded + 1, fy_payload, length) == 0);
    for (size_t i = length + 1; i < sizeof guarded; ++i) assert(guarded[i] == 165);
}

int main(void) {
    static const struct fy_probe_step success[] = {
        {0, 6, -1, EINTR}, {0, 6, 2, 0}, {2, 4, -1, EINTR}, {2, 4, 1, 0}, {3, 3, 3, 0}
    };
    static const struct fy_probe_step failure[] = {
        {0, 6, -1, EINTR}, {0, 6, 2, 0}, {2, 4, -1, EIO}
    };
    static const struct fy_probe_step unavailable[] = {{0, 6, -1, EBADF}};
    static const struct fy_probe_step stalled[] = {{0, 6, 2, 0}, {2, 4, 0, 0}};
    for (int descriptor = STDOUT_FILENO; descriptor <= STDERR_FILENO; ++descriptor) {
        fy_check_write(descriptor, success, sizeof success / sizeof success[0], 0, 6);
        fy_check_write(descriptor, failure, sizeof failure / sizeof failure[0], 3, 2);
        fy_check_write(descriptor, unavailable, 1, 2, 0);
        fy_check_write(descriptor, stalled, 2, 4, 2);
        fy_check_write_no_io(descriptor);
    }
    static const struct fy_probe_step input[] = {{0, 8, -1, EINTR}, {0, 8, 3, 0}};
    static const struct fy_probe_step read_failure[] = {{0, 8, -1, EIO}};
    static const struct fy_probe_step read_unavailable[] = {{0, 8, -1, EBADF}};
    static const struct fy_probe_step eof[] = {{0, 8, 0, 0}};
    fy_check_read(input, 2, 0, 3);
    fy_check_read(read_failure, 1, 3, 0);
    fy_check_read(read_unavailable, 1, 2, 0);
    fy_check_read(eof, 1, 0, 0);
    uint8_t byte = 165;
    fy_prepare(NULL, 0, STDIN_FILENO, 0, &byte, 1);
    struct fy_std_console_read_result_v0 r = fy_std_stdin_read_v0(&byte, 0);
    assert(r.status == 0 && r.read == 0);
    r = fy_std_stdin_read_v0(NULL, 0);
    assert(r.status == 0 && r.read == 0);
    r = fy_std_stdin_read_v0(NULL, 1);
    assert(r.status == 1 && r.read == 0 && byte == 165 && fy_calls == 0);
    return 0;
}
