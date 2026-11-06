//go:build ignore

/* Standalone single-threaded Fairy CLI only. Append after clock, protocol,
 * producer and reader. No signal handler or another owner may reap this child. */
#include <fcntl.h>
#include <limits.h>
#include <poll.h>
#include <signal.h>
#include <stdlib.h>
#include <sys/prctl.h>
#include <sys/wait.h>
#include <unistd.h>

typedef struct {
    fy_linux_https_reader reader;
    size_t started, deadline, observed;
    pid_t child;
    int pipe_in, pipe_out;
} fy_linux_https_request;

static uint32_t fy_linux_https_budget(fy_linux_https_request *p, int *remaining) {
    size_t now = 0;
    uint32_t code = fairyc_private_linux_monotonic_ms_v1(&now);
    if (code != 0) { return code; }
    if (now < p->observed) { return EIO; }
    p->observed = now;
    if (now >= p->deadline) { return ETIMEDOUT; }
    *remaining = (int)(p->deadline-now);
    return 0;
}

static uint32_t fy_linux_https_pipe_input(void *context, void *data,
        size_t capacity, size_t *copied) {
    fy_linux_https_request *p = context;
    *copied = 0;
    for (;;) {
        int remaining = 0;
        uint32_t code = fy_linux_https_budget(p, &remaining);
        if (code != 0) { return code; }
        struct pollfd descriptor = {p->pipe_in, POLLIN, 0};
        int ready = poll(&descriptor, 1, remaining);
        if (ready < 0) {
            if (errno == EINTR) { continue; }
            return (uint32_t)errno;
        }
        /* Even readiness at/after the deadline does not extend the budget. */
        code = fy_linux_https_budget(p, &remaining);
        if (code != 0) { return code; }
        if (ready == 0) { continue; }
        if ((descriptor.revents & POLLNVAL) != 0) { return EBADF; }
        if ((descriptor.revents & (POLLIN | POLLHUP | POLLERR)) == 0) { return EIO; }
        ssize_t count = read(p->pipe_in, data, capacity);
        if (count < 0) {
            if (errno == EINTR || errno == EAGAIN || errno == EWOULDBLOCK) { continue; }
            return (uint32_t)errno;
        }
        code = fy_linux_https_budget(p, &remaining);
        if (code != 0) { return code; }
        if ((size_t)count > capacity) { return EPROTO; }
        *copied = (size_t)count;
        return 0;
    }
}

static uint32_t fy_linux_https_pipe_output(void *context, const void *data, size_t size) {
    int descriptor = *(const int *)context;
    size_t used = 0;
    while (used < size) {
        ssize_t count = write(descriptor, (const uint8_t *)data+used, size-used);
        if (count < 0) {
            if (errno == EINTR) { continue; }
            return (uint32_t)errno;
        }
        if (count == 0 || (size_t)count > size-used) { return EIO; }
        used += (size_t)count;
    }
    return 0;
}

static uint32_t fy_linux_https_child_boundary(int *descriptor, pid_t parent) {
    /* Install before library/DNS work; cover the fork->prctl parent-death race. */
    if (prctl(PR_SET_PDEATHSIG, SIGKILL) != 0) { return (uint32_t)errno; }
    if (getppid() != parent) { return ECANCELED; }
    if (*descriptor != 3 && dup2(*descriptor, 3) < 0) { return (uint32_t)errno; }
    *descriptor = 3;
    /* close_range reports failure; never silently leave workspace fds inherited.
     * The Linux reference kernel supports it. No /proc directory scan is used. */
    if (close_range(4, UINT_MAX, 0) != 0) { return (uint32_t)errno; }
    for (int fd = 0; fd < 3; ++fd) {
        if (close(fd) != 0 && errno != EBADF) { return (uint32_t)errno; }
    }
    return 0;
}

static void fy_linux_https_child(fy_linux_https_request *p, const uint8_t *address,
        size_t maximum, pid_t parent) {
    int descriptor = p->pipe_out;
    uint32_t code = fy_linux_https_child_boundary(&descriptor, parent);
    int remaining = 0;
    if (code == 0) { code = fy_linux_https_budget(p, &remaining); }
    if (code == 0) {
        code = fy_linux_https_transfer((const char *)address, maximum,
            (size_t)remaining, fy_linux_https_pipe_output, &descriptor);
    } else {
        fy_linux_https_frame frame = {3, 0, code};
        (void)fy_linux_https_pipe_output(&descriptor, &frame, sizeof frame);
    }
    /* No inherited atexit, stdio flush, Fairy cleanup or parent heap mutation. */
    _exit(code == 0 ? 0 : 1);
}

uint32_t fairyc_private_linux_https_open_v1(const uint8_t *address, size_t size,
        size_t maximum, size_t timeout, size_t *output) {
    if (output == NULL) { return EINVAL; }
    *output = 0;
    if (address == NULL || size < 10 || size > 4096 || address[size-1] != 0 ||
        memchr(address, 0, size-1) != NULL || memcmp(address, "https://", 8) != 0 ||
        maximum == 0 || maximum > 16777216 || timeout == 0 || timeout > 30000) { return EINVAL; }
    size_t now = 0;
    uint32_t code = fairyc_private_linux_monotonic_ms_v1(&now);
    if (code != 0) { return code; }
    if (now > SIZE_MAX-timeout) { return EOVERFLOW; }
    /* Default SIGCHLD guarantees an unreaped PID remains exclusively ours.
     * Reject auto-reap or user handlers before creating a child. */
    struct sigaction action;
    if (sigaction(SIGCHLD, NULL, &action) != 0) { return (uint32_t)errno; }
    if (action.sa_handler != SIG_DFL || (action.sa_flags & SA_NOCLDWAIT) != 0) { return ENOTSUP; }
    fy_linux_https_request *p = calloc(1, sizeof *p);
    if (p == NULL) { return ENOMEM; }
    p->pipe_in = p->pipe_out = -1;
    p->started = p->observed = now; p->deadline = now+timeout;
    p->reader.input = fy_linux_https_pipe_input; p->reader.context = p;
    p->reader.maximum = maximum;
    *output = (size_t)(uintptr_t)p;
    int remaining = 0;
    code = fy_linux_https_budget(p, &remaining);
    if (code != 0) { return code; }
    int descriptors[2] = {-1, -1};
    if (pipe2(descriptors, O_CLOEXEC) != 0) { return (uint32_t)errno; }
    p->pipe_in = descriptors[0]; p->pipe_out = descriptors[1];
    int flags = fcntl(p->pipe_in, F_GETFL);
    if (flags < 0 || fcntl(p->pipe_in, F_SETFL, flags | O_NONBLOCK) < 0) { return (uint32_t)errno; }
    code = fy_linux_https_budget(p, &remaining);
    if (code != 0) { return code; }
    pid_t parent = getpid();
    pid_t child = fork();
    if (child < 0) { return (uint32_t)errno; }
    if (child == 0) { fy_linux_https_child(p, address, maximum, parent); }
    p->child = child;
    /* Linux close consumes the descriptor on EINTR; never retry a number. */
    int descriptor = p->pipe_out; p->pipe_out = -1;
    if (close(descriptor) != 0) { return (uint32_t)errno; }
    return fy_linux_https_budget(p, &remaining);
}

uint32_t fairyc_private_linux_https_headers_v1(size_t raw, uint32_t *status,
        uint8_t *encoding, size_t capacity, size_t *length, size_t *declared) {
    if (status == NULL || length == NULL || declared == NULL) { return EINVAL; }
    *status = 0; *length = 0; *declared = SIZE_MAX;
    if (raw == 0 || encoding == NULL || capacity == 0 || capacity > 128) { return EINVAL; }
    fy_linux_https_request *p = (fy_linux_https_request *)(uintptr_t)raw;
    if (p->child <= 0 || p->pipe_in < 0) { return EINVAL; }
    int remaining = 0;
    uint32_t code = fy_linux_https_budget(p, &remaining);
    if (code == 0) { code = fy_linux_https_reader_headers(&p->reader); }
    if (code != 0) { return code; }
    if (p->reader.head.encoding_length > capacity) { return EOVERFLOW; }
    *status = p->reader.head.status; *length = p->reader.head.encoding_length;
    *declared = p->reader.head.declared;
    memcpy(encoding, p->reader.head.encoding, *length);
    return 0;
}

uint32_t fairyc_private_linux_https_read_v1(size_t raw, uint8_t *data,
        size_t capacity, size_t *copied) {
    if (copied == NULL) { return EINVAL; }
    *copied = 0;
    if (raw == 0 || data == NULL || capacity == 0 || capacity > 65536) { return EINVAL; }
    fy_linux_https_request *p = (fy_linux_https_request *)(uintptr_t)raw;
    if (p->child <= 0 || p->pipe_in < 0) { return EINVAL; }
    int remaining = 0;
    uint32_t code = fy_linux_https_budget(p, &remaining);
    if (code != 0) { return code; }
    return fy_linux_https_reader_read(&p->reader, data, capacity, copied);
}

static uint32_t fy_linux_https_reap(fy_linux_https_request *p) {
    if (p->child == 0) { return 0; }
    siginfo_t info;
    int result;
    do {
        memset(&info, 0, sizeof info);
        result = waitid(P_PID, (id_t)p->child, &info, WEXITED | WNOWAIT | WNOHANG);
    } while (result < 0 && errno == EINTR);
    if (result < 0) { return (uint32_t)errno; } /* ECHILD never authorizes kill. */
    uint32_t code = 0;
    int killed = info.si_pid == 0;
    if (killed && kill(p->child, SIGKILL) != 0 && errno != ESRCH) { code = (uint32_t)errno; }
    int status = 0;
    pid_t waited;
    /* A refused signal must not turn close into an unbounded wait. Retain the
     * refusal; a terminal child is still reaped, and no success is reported. */
    do { waited = waitpid(p->child, &status, code == 0 ? 0 : WNOHANG); } while (waited < 0 && errno == EINTR);
    if (waited < 0 && code == 0) { code = (uint32_t)errno; }
    if (waited == 0 && code == 0) { code = EIO; }
    if (waited == p->child && !killed && p->reader.ended && p->reader.error == 0 &&
        (!WIFEXITED(status) || WEXITSTATUS(status) != 0) && code == 0) { code = EIO; }
    p->child = 0;
    return code;
}

uint32_t fairyc_private_linux_https_close_v1(size_t raw) {
    if (raw == 0) { return EINVAL; }
    fy_linux_https_request *p = (fy_linux_https_request *)(uintptr_t)raw;
    uint32_t code = fy_linux_https_reap(p);
    int descriptors[2] = {p->pipe_in, p->pipe_out};
    p->pipe_in = p->pipe_out = -1;
    for (size_t i = 0; i < 2; ++i) {
        if (descriptors[i] >= 0 && close(descriptors[i]) != 0 && code == 0) { code = (uint32_t)errno; }
    }
    /* Successful collection also charges consuming-close work to the original
     * deadline. Failed/partial operations retain their independent primary
     * failure instead of manufacturing a second timeout during cancellation. */
    if (code == 0 && p->reader.ended && p->reader.error == 0) {
        int remaining = 0;
        code = fy_linux_https_budget(p, &remaining);
    }
    free(p);
    return code;
}
