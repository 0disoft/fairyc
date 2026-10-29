/* Appended after linux_process.c; private helpers and feature macros are shared. */
#include <fcntl.h>
#include <poll.h>

typedef struct {
    uint32_t pid;
    int readers[3];
    uint8_t receipt[276];
    size_t receipt_length;
    int collected;
} fy_linux_capture;

/* On Linux close releases the descriptor even on EINTR: never retry a reused fd. */
static uint32_t fy_linux_capture_close(int *fd) {
    if (*fd < 0) { return 0; }
    int value = *fd;
    *fd = -1;
    return close(value) == 0 ? 0 : (uint32_t)errno;
}

static uint32_t fy_linux_capture_pipe(int pair[2], int minimum) {
    if (pipe2(pair, O_CLOEXEC) != 0) { return (uint32_t)errno; }
    for (int i = 0; i < 2; ++i) {
        /* Keep dup2 sources away from stdin/stdout/stderr even if those are closed. */
        if (pair[i] < minimum) {
            int moved = fcntl(pair[i], F_DUPFD_CLOEXEC, minimum);
            if (moved < 0) { return (uint32_t)errno; }
            uint32_t code = fy_linux_capture_close(&pair[i]);
            pair[i] = moved;
            if (code != 0) { return code; }
        }
    }
    int flags = fcntl(pair[0], F_GETFL);
    if (flags < 0 || fcntl(pair[0], F_SETFL, flags | O_NONBLOCK) < 0) { return (uint32_t)errno; }
    return 0;
}

static uint32_t fy_linux_capture_open(const uint8_t *data, size_t size,
        size_t argc, const uint8_t *cwd, size_t cwd_size, int run, void **output) {
    if (output == NULL) { return EINVAL; }
    *output = NULL;
    fy_linux_capture *capture = calloc(1, sizeof *capture);
    if (capture == NULL) { return ENOMEM; }
    capture->readers[0] = -1; capture->readers[1] = -1; capture->readers[2] = -1;
    int out[2] = {-1, -1}, err[2] = {-1, -1}, control[2] = {-1, -1};
    uint32_t code = fy_linux_capture_pipe(out, run ? 4 : 3);
    if (code == 0) { code = fy_linux_capture_pipe(err, run ? 4 : 3); }
    if (code == 0 && run) { code = fy_linux_capture_pipe(control, 4); }
    if (code == 0) { code = fy_linux_process_spawn(data, size, argc, cwd, cwd_size, out[1], err[1], control[1], &capture->pid); }
    uint32_t out_closed = fy_linux_capture_close(&out[1]);
    uint32_t err_closed = fy_linux_capture_close(&err[1]);
    uint32_t control_closed = fy_linux_capture_close(&control[1]);
    if (code == 0) { code = out_closed != 0 ? out_closed : err_closed != 0 ? err_closed : control_closed; }
    if (code != 0) {
        if (capture->pid != 0) { (void)fairyc_private_linux_process_discard_v1(capture->pid); }
        (void)fy_linux_capture_close(&out[0]);
        (void)fy_linux_capture_close(&err[0]);
        (void)fy_linux_capture_close(&control[0]);
        free(capture);
        return code;
    }
    capture->readers[0] = out[0]; capture->readers[1] = err[0]; capture->readers[2] = control[0];
    *output = capture;
    return 0;
}

uint32_t fairyc_private_linux_capture_open_v1(const uint8_t *data, size_t size,
        size_t argc, const uint8_t *cwd, size_t cwd_size, void **output) {
    return fy_linux_capture_open(data, size, argc, cwd, cwd_size, 0, output);
}

uint32_t fairyc_private_linux_run_capture_open_v1(const uint8_t *data, size_t size,
        size_t argc, const uint8_t *cwd, size_t cwd_size, void **output) {
    return fy_linux_capture_open(data, size, argc, cwd, cwd_size, 1, output);
}

/* Only a fully drained collection may expose the bounded control record. */
uint32_t fairyc_private_linux_run_capture_receipt_v1(void *raw, uint8_t *data,
        size_t capacity, size_t *length) {
    if (length == NULL) { return EINVAL; }
    *length = 0;
    if (raw == NULL || data == NULL) { return EINVAL; }
    fy_linux_capture *capture = raw;
    if (!capture->collected || capture->pid != 0 || capture->readers[2] >= 0 ||
        capture->receipt_length > capacity) { return EINVAL; }
    memcpy(data, capture->receipt, capture->receipt_length);
    *length = capture->receipt_length;
    return 0;
}

uint32_t fairyc_private_linux_capture_drop_v1(void *raw) {
    if (raw == NULL) { return 0; }
    fy_linux_capture *capture = raw;
    uint32_t code = fairyc_private_linux_process_discard_v1(capture->pid);
    uint32_t a = fy_linux_capture_close(&capture->readers[0]);
    uint32_t b = fy_linux_capture_close(&capture->readers[1]);
    uint32_t c = fy_linux_capture_close(&capture->readers[2]);
    free(capture);
    return code != 0 ? code : a != 0 ? a : b != 0 ? b : c;
}

/* One bounded read per stream per round keeps the other stream and clock live. */
static uint32_t fy_linux_capture_read(int *fd, uint8_t *data, size_t capacity,
        size_t *length, int *exceeded) {
    if (*fd < 0) { return 0; }
    size_t room = capacity - *length;
    uint8_t extra = 0;
    size_t request = room == 0 ? 1 : room > 4096 ? 4096 : room;
    ssize_t count = read(*fd, room == 0 ? &extra : data + *length, request);
    if (count < 0) { return errno == EAGAIN || errno == EWOULDBLOCK || errno == EINTR ? 0 : (uint32_t)errno; }
    if (count == 0) { return fy_linux_capture_close(fd); }
    if ((size_t)count > request) { return EIO; }
    if (room == 0) { *exceeded = 1; }
    else { *length += (size_t)count; }
    return 0;
}

static uint32_t fy_linux_capture_stop(fy_linux_capture *capture) {
    if (capture->pid == 0) { return 0; }
    uint32_t code = fy_linux_process_observe(capture->pid, WNOHANG);
    if (code != 0) { return code; }
    int status = 0;
    code = fy_linux_process_finish(capture->pid, &status);
    if (code == 0) { capture->pid = 0; }
    return code;
}

uint32_t fairyc_private_linux_capture_collect_v1(void *raw, uint32_t timeout_ms,
        uint8_t *out, size_t out_capacity, size_t *out_length,
        uint8_t *err, size_t err_capacity, size_t *err_length,
        uint32_t *kind, uint32_t *value, uint32_t *pid) {
    if (kind == NULL || value == NULL || pid == NULL || out_length == NULL || err_length == NULL) { return EINVAL; }
    *kind = 0; *value = 0; *pid = 0; *out_length = 0; *err_length = 0;
    if (raw == NULL || timeout_ms == 0 || timeout_ms > 86400000 || out_capacity > 1048576 ||
        err_capacity > 1048576 - out_capacity || (out_capacity != 0 && out == NULL) ||
        (err_capacity != 0 && err == NULL)) { return EINVAL; }
    fy_linux_capture *capture = raw;
    if (capture->collected || capture->pid <= 1 || capture->pid > INT_MAX) { return EINVAL; }
    capture->collected = 1;
    *pid = capture->pid;
    uint64_t started = 0;
    uint32_t code = fy_linux_process_now(&started);
    if (code != 0) { return code; }
    uint64_t deadline = started + (uint64_t)timeout_ms * UINT64_C(1000000);
    uint32_t completed_kind = 0, completed_value = 0;
    for (;;) {
        uint64_t now = 0;
        code = fy_linux_process_now(&now);
        if (code != 0) { return code; }
        if (now < started) { return EIO; }
        int exceeded = 0;
        code = fy_linux_capture_read(&capture->readers[0], out, out_capacity, out_length, &exceeded);
        uint32_t stream = 1;
        if (code == 0 && !exceeded) {
            code = fy_linux_capture_read(&capture->readers[1], err, err_capacity, err_length, &exceeded);
            stream = 2;
        }
        if (code == 0 && !exceeded && capture->readers[2] >= 0) {
            code = fy_linux_capture_read(&capture->readers[2], capture->receipt,
                sizeof capture->receipt, &capture->receipt_length, &exceeded);
            if (exceeded) { return EOVERFLOW; }
        }
        if (code != 0) { return code; }
        if (exceeded) {
            code = fy_linux_capture_stop(capture);
            if (code != 0) { return code; }
            *kind = 4; *value = stream;
            return 0;
        }
        if (capture->pid != 0) {
            siginfo_t info;
            memset(&info, 0, sizeof info);
            if (waitid(P_PID, (id_t)capture->pid, &info, WEXITED | WNOWAIT | WNOHANG) != 0) {
                code = (uint32_t)errno;
                if (code == EINTR && now < deadline) { continue; }
                return code;
            }
            if (info.si_pid != 0) {
                int status = 0;
                code = fy_linux_process_finish(capture->pid, &status);
                if (code != 0) { return code; }
                capture->pid = 0;
                if (WIFEXITED(status)) { completed_kind = 1; completed_value = (uint32_t)WEXITSTATUS(status); }
                else if (WIFSIGNALED(status)) { completed_kind = 2; completed_value = (uint32_t)WTERMSIG(status); }
                else { return EIO; }
            }
        }
        if (capture->pid == 0 && capture->readers[0] < 0 && capture->readers[1] < 0 && capture->readers[2] < 0) {
            *kind = completed_kind; *value = completed_value;
            return 0;
        }
        if (now >= deadline) {
            code = fy_linux_capture_stop(capture);
            if (code != 0) { return code; }
            *kind = 3;
            return 0;
        }
        struct pollfd fds[3] = {{capture->readers[0], POLLIN, 0}, {capture->readers[1], POLLIN, 0}, {capture->readers[2], POLLIN, 0}};
        uint64_t remaining_ms = (deadline - now + UINT64_C(999999)) / UINT64_C(1000000);
        int waited = poll(fds, capture->readers[2] >= 0 ? 3 : 2, remaining_ms > 10 ? 10 : (int)remaining_ms);
        if (waited < 0 && errno != EINTR) { return (uint32_t)errno; }
        if (waited >= 0 && ((fds[0].revents | fds[1].revents | fds[2].revents) & POLLNVAL)) { return EBADF; }
    }
}
